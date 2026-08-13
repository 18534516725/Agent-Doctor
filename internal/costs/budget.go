package costs

import (
	"math"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

type Budget struct {
	Currency       string
	LimitMicros    int64
	WarningPercent int64
}

type BudgetResult struct {
	Valid              bool
	ExactMicros        int64
	EstimatedMicros    int64
	UnavailableRecords int
	PercentUsed        int64
	Warning            bool
	Exceeded           bool
}

// EvaluateBudget keeps verified bills separate from estimates and never makes
// unavailable or foreign-currency data look like a zero-value cost.
func EvaluateBudget(budget Budget, records []CostRecord) BudgetResult {
	if len(budget.Currency) != 3 || budget.LimitMicros <= 0 || budget.WarningPercent < 0 || budget.WarningPercent > 100 {
		return BudgetResult{}
	}
	result := BudgetResult{Valid: true}
	for _, record := range records {
		if record.Precision == events.PrecisionUnavailable || !record.Amount.Valid() || record.Amount.Currency != budget.Currency {
			result.UnavailableRecords++
			continue
		}
		switch record.Precision {
		case events.PrecisionExact:
			if result.ExactMicros > math.MaxInt64-record.Amount.Micros {
				result.Valid = false
				return result
			}
			result.ExactMicros += record.Amount.Micros
		case events.PrecisionEstimated:
			if result.EstimatedMicros > math.MaxInt64-record.Amount.Micros {
				result.Valid = false
				return result
			}
			result.EstimatedMicros += record.Amount.Micros
		default:
			result.UnavailableRecords++
		}
	}
	if result.ExactMicros > math.MaxInt64-result.EstimatedMicros {
		result.Valid = false
		return result
	}
	total := result.ExactMicros + result.EstimatedMicros
	if total > math.MaxInt64/100 {
		result.Valid = false
		return result
	}
	result.PercentUsed = total * 100 / budget.LimitMicros
	result.Warning = result.PercentUsed >= budget.WarningPercent
	result.Exceeded = total > budget.LimitMicros
	return result
}
