package costs

import (
	"fmt"
	"strconv"

	"github.com/18534516725/Agent-Doctor/internal/events"
)

type Money struct {
	Currency string
	Micros   int64
}

func (value Money) Valid() bool { return len(value.Currency) == 3 && value.Micros >= 0 }

func (value Money) Display(decimals int) string {
	if decimals < 0 {
		decimals = 0
	}
	if decimals > 6 {
		decimals = 6
	}
	divisor := int64(1)
	if decimals < 6 {
		for step := 0; step < 6-decimals; step++ {
			divisor *= 10
		}
	}
	rounded := (value.Micros + divisor/2) / divisor
	whole := rounded
	fraction := int64(0)
	if decimals > 0 {
		scale := int64(1)
		for step := 0; step < decimals; step++ {
			scale *= 10
		}
		whole = rounded / scale
		fraction = rounded % scale
		return fmt.Sprintf("%s %d.%0"+strconv.Itoa(decimals)+"d", value.Currency, whole, fraction)
	}
	return fmt.Sprintf("%s %d", value.Currency, whole)
}

type Usage struct {
	InputTokens  int64
	OutputTokens int64
	CachedTokens int64
}
type Price struct {
	Currency            string
	InputMicrosPerMTok  int64
	OutputMicrosPerMTok int64
	CachedMicrosPerMTok int64
	Version             string
	ExchangeRateVersion string
}
type ExactBill struct {
	Amount              Money
	Provenance          string
	PriceVersion        string
	ExchangeRateVersion string
}
type CalculationInput struct {
	SessionID      string
	Usage          Usage
	Price          Price
	ExactBill      *ExactBill
	TargetCurrency string
}
type CostRecord struct {
	SessionID           string
	Amount              Money
	Precision           events.Precision
	Provenance          string
	PriceVersion        string
	ExchangeRateVersion string
}
