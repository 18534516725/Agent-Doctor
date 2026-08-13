package costs

import (
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

// SubscriptionInput contains a user-provided or client-reported quota snapshot.
// All amounts use integer micro-units; unavailable source data stays unavailable.
type SubscriptionInput struct {
	QuotaName       string
	TotalMicros     int64
	UsedMicros      int64
	PeriodStart     time.Time
	PeriodEnd       time.Time
	Now             time.Time
	Precision       events.Precision
	Provenance      string
	UpgradeAt       *time.Time
	TaskUsedMicros  int64
	TaskTotalMicros int64
}

type SubscriptionAnalysis struct {
	QuotaName            string
	Precision            events.Precision
	Provenance           string
	UsedPercent          *int64
	RemainingPercent     *int64
	NextRefresh          *time.Time
	BurnRateMicrosPerDay int64
	EstimatedExhaustion  *time.Time
	Confidence           string
	TaskAllocation       *int64
}

// AnalyzeSubscription describes known quota state without inventing a forecast
// when a new plan, incomplete period, or unavailable snapshot makes it unsafe.
func AnalyzeSubscription(input SubscriptionInput) SubscriptionAnalysis {
	result := SubscriptionAnalysis{
		QuotaName:  input.QuotaName,
		Precision:  input.Precision,
		Provenance: input.Provenance,
		Confidence: "unavailable",
	}

	if input.TotalMicros <= 0 || input.UsedMicros < 0 || input.PeriodStart.IsZero() || input.PeriodEnd.IsZero() || !input.PeriodEnd.After(input.PeriodStart) || input.Now.IsZero() {
		result.Precision = events.PrecisionUnavailable
		return result
	}
	if result.Precision != events.PrecisionExact && result.Precision != events.PrecisionEstimated {
		result.Precision = events.PrecisionUnavailable
		return result
	}

	used := input.UsedMicros * 100 / input.TotalMicros
	if used > 100 {
		used = 100
	}
	remaining := int64(100) - used
	result.UsedPercent = &used
	result.RemainingPercent = &remaining
	refresh := input.PeriodEnd.UTC()
	result.NextRefresh = &refresh
	result.TaskAllocation = AllocateTaskQuota(input.TaskUsedMicros, input.TaskTotalMicros)

	elapsed := input.Now.Sub(input.PeriodStart)
	if elapsed <= 0 || input.UsedMicros == 0 || input.UpgradeAt != nil {
		result.Confidence = "low"
		return result
	}

	result.BurnRateMicrosPerDay = input.UsedMicros * int64(24*time.Hour) / int64(elapsed)
	if result.BurnRateMicrosPerDay <= 0 {
		result.Confidence = "low"
		return result
	}

	remainingMicros := input.TotalMicros - input.UsedMicros
	daysUntilExhaustion := (remainingMicros + result.BurnRateMicrosPerDay - 1) / result.BurnRateMicrosPerDay
	exhaustion := input.Now.Add(time.Duration(daysUntilExhaustion) * 24 * time.Hour)
	if exhaustion.Before(input.PeriodEnd) {
		result.EstimatedExhaustion = &exhaustion
	}

	if result.Precision == events.PrecisionExact {
		result.Confidence = "medium"
	} else {
		result.Confidence = "low"
	}
	return result
}

// AllocateTaskQuota reports a task's share only when both values came from the
// same known quota period. A nil value deliberately means "not attributable".
func AllocateTaskQuota(numerator, denominator int64) *int64 {
	if numerator <= 0 || denominator <= 0 || numerator > denominator {
		return nil
	}
	percent := numerator * 100 / denominator
	return &percent
}
