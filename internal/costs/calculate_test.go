package costs

import (
	"testing"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

func TestCalculateUsesIntegerMicrosAndCachePrice(t *testing.T) {
	record, err := Calculate(CalculationInput{
		SessionID: "session-1", Usage: Usage{InputTokens: 1_000_000, OutputTokens: 2_000_000, CachedTokens: 500_000},
		Price: Price{Currency: "USD", InputMicrosPerMTok: 500_000, OutputMicrosPerMTok: 3_000_000, CachedMicrosPerMTok: 50_000, Version: "catalog-2026-08-13"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Amount != (Money{Currency: "USD", Micros: 6_525_000}) {
		t.Fatalf("amount=%+v", record.Amount)
	}
	if record.Precision != events.PrecisionEstimated || record.Provenance != "local-price-catalog" {
		t.Fatalf("record=%+v", record)
	}
}

func TestCalculateExactBillTakesPrecedence(t *testing.T) {
	record, err := Calculate(CalculationInput{
		SessionID: "session-1", Usage: Usage{InputTokens: 999}, Price: Price{Currency: "USD", InputMicrosPerMTok: 1},
		ExactBill: &ExactBill{Amount: Money{Currency: "CNY", Micros: 123_456}, Provenance: "user-scoped-billing-api", PriceVersion: "bill-12", ExchangeRateVersion: "rate-8"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Amount != (Money{Currency: "CNY", Micros: 123_456}) || record.Precision != events.PrecisionExact || record.PriceVersion != "bill-12" {
		t.Fatalf("record=%+v", record)
	}
}

func TestCalculateMissingOrIncompatiblePriceIsUnavailable(t *testing.T) {
	for name, input := range map[string]CalculationInput{
		"missing price":     {SessionID: "s", Usage: Usage{InputTokens: 1}},
		"currency mismatch": {SessionID: "s", Usage: Usage{InputTokens: 1}, Price: Price{Currency: "USD", InputMicrosPerMTok: 1}, TargetCurrency: "CNY"},
	} {
		t.Run(name, func(t *testing.T) {
			record, err := Calculate(input)
			if err != nil {
				t.Fatal(err)
			}
			if record.Precision != events.PrecisionUnavailable || record.Amount.Micros != 0 {
				t.Fatalf("record=%+v", record)
			}
		})
	}
}

func TestCalculateRejectsNegativeInputAndRoundingOnlyAtDisplay(t *testing.T) {
	if _, err := Calculate(CalculationInput{SessionID: "s", Usage: Usage{InputTokens: -1}}); err == nil {
		t.Fatal("expected negative tokens to fail")
	}
	money := Money{Currency: "USD", Micros: 1_234_567}
	if got := money.Display(2); got != "USD 1.23" {
		t.Fatalf("display=%q", got)
	}
	if money.Micros != 1_234_567 {
		t.Fatal("display must not mutate amount")
	}
}

func FuzzCostNeverNegative(f *testing.F) {
	f.Add(int64(0), int64(1), int64(2), int64(1))
	f.Fuzz(func(t *testing.T, input, output, cached, price int64) {
		input, output, cached, price = nonNegative(input), nonNegative(output), nonNegative(cached), nonNegative(price)
		record, err := Calculate(CalculationInput{SessionID: "s", Usage: Usage{InputTokens: input, OutputTokens: output, CachedTokens: cached}, Price: Price{Currency: "USD", InputMicrosPerMTok: price, OutputMicrosPerMTok: price, CachedMicrosPerMTok: price}})
		if err != nil {
			t.Fatal(err)
		}
		if record.Amount.Micros < 0 {
			t.Fatalf("negative cost: %+v", record)
		}
	})
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return -(value + 1)
	}
	return value
}
