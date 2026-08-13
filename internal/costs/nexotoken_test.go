package costs

import (
	"strings"
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

func TestParseNexoTokenUsageAllowsOnlyPublicFields(t *testing.T) {
	body := strings.NewReader(`{
  "schemaVersion": 1,
  "records": [{
    "publicModelName": "Example Code Model",
    "inputTokens": 120,
    "outputTokens": 80,
    "cachedTokens": 10,
    "chargedAmountMicros": 123456,
    "currency": "CNY",
    "requestTimestamp": "2026-08-13T08:00:00+08:00",
    "taskCorrelationId": "my-task-42",
    "upstream_provider_id": "must-not-pass-through",
    "api_keys.channel": "must-not-pass-through"
  }]
}`)
	records, err := ParseNexoTokenUsage(body, UsageRange{Start: mustTime(t, "2026-08-13T00:00:00+08:00"), End: mustTime(t, "2026-08-14T00:00:00+08:00")})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("records=%+v", records)
	}
	if records[0].PublicModelName != "Example Code Model" || records[0].TaskCorrelationID != "my-task-42" || records[0].Cost.Precision != events.PrecisionExact || records[0].Cost.Amount != (Money{Currency: "CNY", Micros: 123456}) {
		t.Fatalf("record=%+v", records[0])
	}
}

func TestParseNexoTokenUsageRejectsUnsupportedOrUnsafeRecords(t *testing.T) {
	window := UsageRange{Start: mustTime(t, "2026-08-13T00:00:00Z"), End: mustTime(t, "2026-08-14T00:00:00Z")}
	for name, payload := range map[string]string{
		"unsupported schema": `{"schemaVersion":2,"records":[]}`,
		"bad currency":       `{"schemaVersion":1,"records":[{"publicModelName":"Example","currency":"US","requestTimestamp":"2026-08-13T08:00:00Z"}]}`,
		"negative amount":    `{"schemaVersion":1,"records":[{"publicModelName":"Example","currency":"USD","chargedAmountMicros":-1,"requestTimestamp":"2026-08-13T08:00:00Z"}]}`,
		"outside range":      `{"schemaVersion":1,"records":[{"publicModelName":"Example","currency":"USD","requestTimestamp":"2026-08-15T08:00:00Z"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseNexoTokenUsage(strings.NewReader(payload), window); err == nil {
				t.Fatal("expected parser rejection")
			}
		})
	}
}

func TestParseNexoTokenUsageDoesNotMutateRequestedRange(t *testing.T) {
	window := UsageRange{Start: time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)}
	_, _ = ParseNexoTokenUsage(strings.NewReader(`{"schemaVersion":1,"records":[]}`), window)
	if window.Start.Location() != time.UTC || window.End.Location() != time.UTC {
		t.Fatalf("range was mutated: %+v", window)
	}
}
