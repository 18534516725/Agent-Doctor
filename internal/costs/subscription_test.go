package costs

import (
	"testing"
	"time"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

func TestSubscriptionUnknownTotalIsUnavailable(t *testing.T) {
	result := AnalyzeSubscription(SubscriptionInput{QuotaName: "Pro", Now: mustTime(t, "2026-08-13T12:00:00+08:00"), UsedMicros: 100})
	if result.Precision != events.PrecisionUnavailable || result.UsedPercent != nil || result.EstimatedExhaustion != nil {
		t.Fatalf("result=%+v", result)
	}
}

func TestSubscriptionUsesShanghaiRefreshAndBurnForecast(t *testing.T) {
	result := AnalyzeSubscription(SubscriptionInput{
		QuotaName: "Pro", TotalMicros: 2_000, UsedMicros: 1_200,
		PeriodStart: mustTime(t, "2026-08-10T08:00:00+08:00"), PeriodEnd: mustTime(t, "2026-08-17T08:00:00+08:00"),
		Now: mustTime(t, "2026-08-13T08:00:00+08:00"), Precision: events.PrecisionExact, Provenance: "user-subscription-snapshot",
	})
	if result.UsedPercent == nil || *result.UsedPercent != 60 || result.RemainingPercent == nil || *result.RemainingPercent != 40 {
		t.Fatalf("percent=%+v", result)
	}
	if result.NextRefresh == nil || result.NextRefresh.Format(time.RFC3339) != "2026-08-17T00:00:00Z" {
		t.Fatalf("refresh=%v", result.NextRefresh)
	}
	if result.BurnRateMicrosPerDay != 400 || result.EstimatedExhaustion == nil || result.EstimatedExhaustion.Format(time.RFC3339) != "2026-08-15T08:00:00+08:00" || result.Confidence != "medium" {
		t.Fatalf("forecast=%+v", result)
	}
}

func TestSubscriptionPartialPeriodsAndUpgradeDoNotInventAllocation(t *testing.T) {
	result := AnalyzeSubscription(SubscriptionInput{
		QuotaName: "Pro", TotalMicros: 1_000, UsedMicros: 0, PeriodStart: mustTime(t, "2026-08-13T00:00:00+08:00"), PeriodEnd: mustTime(t, "2026-08-20T00:00:00+08:00"), Now: mustTime(t, "2026-08-13T12:00:00+08:00"), Precision: events.PrecisionEstimated,
		UpgradeAt: timePtr(mustTime(t, "2026-08-14T00:00:00+08:00")),
	})
	if result.EstimatedExhaustion != nil || result.BurnRateMicrosPerDay != 0 || result.TaskAllocation != nil {
		t.Fatalf("result=%+v", result)
	}
}

func TestTaskAllocationRequiresBothNumeratorAndDenominator(t *testing.T) {
	if allocation := AllocateTaskQuota(100, 0); allocation != nil {
		t.Fatalf("allocation=%v", allocation)
	}
	if allocation := AllocateTaskQuota(0, 100); allocation != nil {
		t.Fatalf("allocation=%v", allocation)
	}
	allocation := AllocateTaskQuota(25, 100)
	if allocation == nil || *allocation != 25 {
		t.Fatalf("allocation=%v", allocation)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
func timePtr(value time.Time) *time.Time { return &value }
