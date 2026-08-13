package costs

import (
	"math"
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
