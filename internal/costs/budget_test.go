package costs

import (
	"math"
	"strings"
	"testing"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

func TestEvaluateBudgetSeparatesExactAndEstimatedSpend(t *testing.T) {
	result := EvaluateBudget(Budget{Currency: "USD", LimitMicros: 10_000_000, WarningPercent: 80}, []CostRecord{
		{Amount: Money{Currency: "USD", Micros: 6_000_000}, Precision: events.PrecisionExact},
		{Amount: Money{Currency: "USD", Micros: 3_000_000}, Precision: events.PrecisionEstimated},
		{Amount: Money{Currency: "CNY", Micros: 1}, Precision: events.PrecisionExact},
	})
	if result.ExactMicros != 6_000_000 || result.EstimatedMicros != 3_000_000 || result.UnavailableRecords != 1 {
		t.Fatalf("result=%+v", result)
	}
	if !result.Warning || result.Exceeded || result.PercentUsed != 90 {
		t.Fatalf("result=%+v", result)
	}
}

func TestEvaluateBudgetDoesNotTurnUnavailableAmountsIntoZero(t *testing.T) {
	result := EvaluateBudget(Budget{Currency: "USD", LimitMicros: 1_000_000, WarningPercent: 80}, []CostRecord{{Amount: Money{Currency: "USD"}, Precision: events.PrecisionUnavailable}})
	if result.UnavailableRecords != 1 || result.Warning || result.Exceeded || result.PercentUsed != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestEvaluateBudgetRejectsInvalidPolicy(t *testing.T) {
	result := EvaluateBudget(Budget{Currency: "US", LimitMicros: -1, WarningPercent: 101}, nil)
	if result.Valid {
		t.Fatalf("result=%+v", result)
	}
}

func TestEvaluateBudgetRejectsOverflowingPercentage(t *testing.T) {
	result := EvaluateBudget(Budget{Currency: "USD", LimitMicros: 1, WarningPercent: 80}, []CostRecord{{Amount: Money{Currency: "USD", Micros: math.MaxInt64}, Precision: events.PrecisionExact}})
	if result.Valid {
		t.Fatalf("result=%+v", result)
	}
}

func TestEvaluateBudgetCreatesRecommendationsForScopeQuotaAndP90(t *testing.T) {
	for _, scope := range []BudgetScope{BudgetDaily, BudgetWeekly, BudgetMonthly, BudgetPerTask} {
		t.Run(string(scope), func(t *testing.T) {
			remainingQuota := int64(5)
			result := EvaluateBudget(Budget{
				Scope: scope, Currency: "USD", LimitMicros: 1_000, WarningPercent: 80,
				RemainingQuotaPercent: &remainingQuota, P90Micros: 850,
			}, []CostRecord{{Amount: Money{Currency: "USD", Micros: 900}, Precision: events.PrecisionExact}})
			if !result.Valid || len(result.Recommendations) < 3 {
				t.Fatalf("result=%+v", result)
			}
			for _, recommendation := range result.Recommendations {
				if strings.Contains(strings.ToLower(recommendation), "stop") {
					t.Fatalf("recommendations must never stop a user process: %q", recommendation)
				}
			}
		})
	}
}
