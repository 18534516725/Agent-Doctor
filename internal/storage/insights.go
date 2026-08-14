package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/insights"
)

func insightDays(days int) int {
	if days == 7 {
		return 7
	}
	return 30
}

func (database *DB) CostIntelligence(ctx context.Context, days int) (insights.CostIntelligence, error) {
	days = insightDays(days)
	result := insights.CostIntelligence{Days: days, Rankings: []insights.Ranking{}, Unknown: []insights.UnknownCost{}, Limitations: []string{}}
	cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format(time.RFC3339Nano)
	err := database.sql.QueryRowContext(ctx, `SELECT COALESCE(SUM(input_tokens),0),COALESCE(SUM(cached_tokens),0),COALESCE(SUM(output_tokens),0),COALESCE(SUM(reasoning_tokens),0),COALESCE(MAX(cost_currency),'USD'),COALESCE(SUM(CASE WHEN cost_precision='exact' THEN cost_amount_micros ELSE 0 END),0),COALESCE(SUM(CASE WHEN cost_precision='estimated' THEN cost_amount_micros ELSE 0 END),0),COALESCE(SUM(CASE WHEN cost_precision='unavailable' OR cost_amount_micros IS NULL THEN 1 ELSE 0 END),0) FROM model_requests WHERE started_at>=?`, cutoff).Scan(&result.Usage.InputTokens, &result.Usage.CachedTokens, &result.Usage.OutputTokens, &result.Usage.ReasoningTokens, &result.Cost.Currency, &result.Cost.ExactMicros, &result.Cost.EstimatedMicros, &result.Cost.UnknownRequests)
	if err != nil {
		return result, fmt.Errorf("query cost intelligence: %w", err)
	}
	result.Usage.UncachedInputTokens = result.Usage.InputTokens - result.Usage.CachedTokens
	if result.Usage.InputTokens > 0 {
		result.Usage.CacheRate = float64(result.Usage.CachedTokens) / float64(result.Usage.InputTokens) * 100
	}
	switch {
	case result.Cost.UnknownRequests == 0:
		result.Cost.Availability = "complete"
	case result.Cost.ExactMicros+result.Cost.EstimatedMicros > 0:
		result.Cost.Availability = "partial"
	default:
		result.Cost.Availability = "unavailable"
	}
	if result.Cost.UnknownRequests > 0 {
		result.Limitations = append(result.Limitations, "部分请求没有可信价格目录，金额保持未知，不按零元计算。")
	}
	for _, group := range []struct{ dimension, expression, join string }{{"project", "r.project_id", ""}, {"session", "r.session_id", ""}, {"client", "c.name", " JOIN clients c ON c.id=r.client_id"}, {"model", "m.display_name", " JOIN models m ON m.id=r.model_id"}} {
		query := fmt.Sprintf(`SELECT %s,COUNT(*),COALESCE(SUM(COALESCE(r.input_tokens,0)-COALESCE(r.cached_tokens,0)),0),COALESCE(SUM(r.output_tokens),0),COALESCE(SUM(CASE WHEN r.cost_precision='exact' THEN r.cost_amount_micros ELSE 0 END),0),COALESCE(SUM(CASE WHEN r.cost_precision='unavailable' OR r.cost_amount_micros IS NULL THEN 1 ELSE 0 END),0) FROM model_requests r%s WHERE r.started_at>=? GROUP BY %s ORDER BY 3 DESC LIMIT 5`, group.expression, group.join, group.expression)
		rows, queryErr := database.sql.QueryContext(ctx, query, cutoff)
		if queryErr != nil {
			return result, fmt.Errorf("query %s rankings: %w", group.dimension, queryErr)
		}
		for rows.Next() {
			var item insights.Ranking
			item.Dimension = group.dimension
			if err := rows.Scan(&item.Label, &item.Requests, &item.UncachedInputTokens, &item.OutputTokens, &item.ExactMicros, &item.UnknownCosts); err != nil {
				_ = rows.Close()
				return result, err
			}
			result.Rankings = append(result.Rankings, item)
		}
		if err := rows.Close(); err != nil {
			return result, err
		}
	}
	rows, err := database.sql.QueryContext(ctx, `SELECT r.id,m.display_name,c.name,r.started_at,r.cost_provenance FROM model_requests r JOIN models m ON m.id=r.model_id JOIN clients c ON c.id=r.client_id WHERE r.started_at>=? AND (r.cost_precision='unavailable' OR r.cost_amount_micros IS NULL) ORDER BY r.started_at DESC LIMIT 20`, cutoff)
	if err != nil {
		return result, fmt.Errorf("query unknown costs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item insights.UnknownCost
		if err := rows.Scan(&item.RequestID, &item.Model, &item.Client, &item.StartedAt, &item.Provenance); err != nil {
			return result, err
		}
		result.Unknown = append(result.Unknown, item)
	}
	return result, rows.Err()
}

type trendBucket struct {
	point     insights.TrendPoint
	latencies []int64
	input     int64
}

func (database *DB) RequestTrends(ctx context.Context, days int) (insights.RequestTrends, error) {
	days = insightDays(days)
	result := insights.RequestTrends{Days: days, Points: []insights.TrendPoint{}, Limitations: []string{"只有请求返回的 Token 和延迟会进入趋势。"}}
	cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format(time.RFC3339Nano)
	rows, err := database.sql.QueryContext(ctx, `SELECT substr(started_at,1,10),status_code,duration_ms,input_tokens,cached_tokens,output_tokens FROM model_requests WHERE started_at>=? ORDER BY started_at`, cutoff)
	if err != nil {
		return result, fmt.Errorf("query request trends: %w", err)
	}
	defer rows.Close()
	buckets := map[string]*trendBucket{}
	order := []string{}
	for rows.Next() {
		var date string
		var status int
		var duration int64
		var input, cached, output sql.NullInt64
		if err := rows.Scan(&date, &status, &duration, &input, &cached, &output); err != nil {
			return result, err
		}
		b := buckets[date]
		if b == nil {
			b = &trendBucket{point: insights.TrendPoint{Date: date}}
			buckets[date] = b
			order = append(order, date)
		}
		b.point.Requests++
		if status == 0 || status >= 400 {
			b.point.Failed++
		}
		b.latencies = append(b.latencies, duration)
		b.input += nullableValue(input)
		b.point.CachedTokens += nullableValue(cached)
		b.point.OutputTokens += nullableValue(output)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	sort.Strings(order)
	for _, date := range order {
		b := buckets[date]
		sort.Slice(b.latencies, func(i, j int) bool { return b.latencies[i] < b.latencies[j] })
		b.point.P50LatencyMS = percentile(b.latencies, .50)
		b.point.P95LatencyMS = percentile(b.latencies, .95)
		b.point.UncachedInputTokens = b.input - b.point.CachedTokens
		if b.point.Requests > 0 {
			b.point.FailureRate = float64(b.point.Failed) / float64(b.point.Requests) * 100
		}
		if b.input > 0 {
			b.point.CacheRate = float64(b.point.CachedTokens) / float64(b.input) * 100
		}
		result.Points = append(result.Points, b.point)
	}
	return result, nil
}

func percentile(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}
	index := int(float64(len(values)-1)*p + .5)
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func nullableValue(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}
