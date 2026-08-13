package costs

import (
	"fmt"
	"math"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

const tokensPerMillion int64 = 1_000_000

func Calculate(input CalculationInput) (CostRecord, error) {
	if input.SessionID == "" {
		return CostRecord{}, fmt.Errorf("session ID is required")
	}
	if input.Usage.InputTokens < 0 || input.Usage.OutputTokens < 0 || input.Usage.CachedTokens < 0 {
		return CostRecord{}, fmt.Errorf("usage tokens cannot be negative")
	}
	if input.ExactBill != nil {
		if !input.ExactBill.Amount.Valid() || input.ExactBill.Provenance == "" {
			return CostRecord{}, fmt.Errorf("exact bill is invalid")
		}
		return CostRecord{SessionID: input.SessionID, Amount: input.ExactBill.Amount, Precision: events.PrecisionExact, Provenance: input.ExactBill.Provenance, PriceVersion: input.ExactBill.PriceVersion, ExchangeRateVersion: input.ExactBill.ExchangeRateVersion}, nil
	}
	if !validPrice(input.Price) || (input.TargetCurrency != "" && input.TargetCurrency != input.Price.Currency) {
		return CostRecord{SessionID: input.SessionID, Amount: Money{Currency: input.TargetCurrency}, Precision: events.PrecisionUnavailable, Provenance: "cost-not-available"}, nil
	}
	amount, ok := multiplyDivide(input.Usage.InputTokens, input.Price.InputMicrosPerMTok)
	if !ok {
		return CostRecord{}, fmt.Errorf("input cost overflows")
	}
	for _, component := range [][2]int64{{input.Usage.OutputTokens, input.Price.OutputMicrosPerMTok}, {input.Usage.CachedTokens, input.Price.CachedMicrosPerMTok}} {
		partial, safe := multiplyDivide(component[0], component[1])
		if !safe || amount > math.MaxInt64-partial {
			return CostRecord{}, fmt.Errorf("cost overflows")
		}
		amount += partial
	}
	return CostRecord{SessionID: input.SessionID, Amount: Money{Currency: input.Price.Currency, Micros: amount}, Precision: events.PrecisionEstimated, Provenance: "local-price-catalog", PriceVersion: input.Price.Version, ExchangeRateVersion: input.Price.ExchangeRateVersion}, nil
}

func validPrice(price Price) bool {
	return len(price.Currency) == 3 && price.InputMicrosPerMTok >= 0 && price.OutputMicrosPerMTok >= 0 && price.CachedMicrosPerMTok >= 0 && price.Version != ""
}
func multiplyDivide(tokens, microsPerMillion int64) (int64, bool) {
	if tokens == 0 || microsPerMillion == 0 {
		return 0, true
	}
	if tokens > math.MaxInt64/microsPerMillion {
		return 0, false
	}
	return tokens * microsPerMillion / tokensPerMillion, true
}
