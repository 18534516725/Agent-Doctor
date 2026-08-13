package costs

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

const nexotokenUsageSchemaVersion = 1

// UsageRange uses a start-inclusive, end-exclusive interval.
type UsageRange struct {
	Start time.Time
	End   time.Time
}

// NexoTokenUsageRecord is deliberately a narrow public DTO. The JSON decoder
// ignores unknown fields so no internal routing or supplier field can cross the
// local API boundary through this import path.
type NexoTokenUsageRecord struct {
	PublicModelName   string
	InputTokens       int64
	OutputTokens      int64
	CachedTokens      int64
	RequestTimestamp  time.Time
	TaskCorrelationID string
	Cost              CostRecord
}

type nexotokenUsageResponse struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Records       []nexotokenUsageRecord `json:"records"`
}

type nexotokenUsageRecord struct {
	PublicModelName     string `json:"publicModelName"`
	InputTokens         int64  `json:"inputTokens"`
	OutputTokens        int64  `json:"outputTokens"`
	CachedTokens        int64  `json:"cachedTokens"`
	ChargedAmountMicros int64  `json:"chargedAmountMicros"`
	Currency            string `json:"currency"`
	RequestTimestamp    string `json:"requestTimestamp"`
	TaskCorrelationID   string `json:"taskCorrelationId"`
}

func ParseNexoTokenUsage(body io.Reader, window UsageRange) ([]NexoTokenUsageRecord, error) {
	if body == nil || window.Start.IsZero() || window.End.IsZero() || !window.End.After(window.Start) {
		return nil, fmt.Errorf("usage range is invalid")
	}
	var response nexotokenUsageResponse
	decoder := json.NewDecoder(io.LimitReader(body, 4<<20))
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("usage response is invalid: %w", err)
	}
	if response.SchemaVersion != nexotokenUsageSchemaVersion {
		return nil, fmt.Errorf("usage schema version is unsupported")
	}

	result := make([]NexoTokenUsageRecord, 0, len(response.Records))
	for _, source := range response.Records {
		timestamp, err := time.Parse(time.RFC3339, source.RequestTimestamp)
		if err != nil || source.PublicModelName == "" || source.Currency == "" || source.InputTokens < 0 || source.OutputTokens < 0 || source.CachedTokens < 0 || source.ChargedAmountMicros < 0 || timestamp.Before(window.Start) || !timestamp.Before(window.End) {
			return nil, fmt.Errorf("usage record is invalid")
		}
		amount := Money{Currency: source.Currency, Micros: source.ChargedAmountMicros}
		if !amount.Valid() {
			return nil, fmt.Errorf("usage record currency is invalid")
		}
		result = append(result, NexoTokenUsageRecord{
			PublicModelName: source.PublicModelName, InputTokens: source.InputTokens, OutputTokens: source.OutputTokens, CachedTokens: source.CachedTokens,
			RequestTimestamp: timestamp, TaskCorrelationID: source.TaskCorrelationID,
			Cost: CostRecord{SessionID: source.TaskCorrelationID, Amount: amount, Precision: events.PrecisionExact, Provenance: "user-scoped-nexotoken-api"},
		})
	}
	return result, nil
}
