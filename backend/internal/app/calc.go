package app

import (
	"errors"
	"math"
)

// Currency is stored in integer centimes (MAD x 100); quantity in thousandths.
// The single rounding stage per line prevents floating-point rounding differences.
func roundDiv(n, d int64) int64 {
	if d <= 0 {
		panic("invalid divisor")
	}
	if n >= 0 {
		return (n + d/2) / d
	}
	return -((-n + d/2) / d)
}

type LineTotals struct {
	SubtotalCents int64 `json:"subtotalCents"`
	VatCents      int64 `json:"vatCents"`
	TotalCents    int64 `json:"totalCents"`
}

func CalculateLine(qtyMilli, priceCents int64, discountBps, vatBps int) (LineTotals, error) {
	if qtyMilli <= 0 || priceCents < 0 || discountBps < 0 || discountBps > 10000 || vatBps < 0 || vatBps > 10000 {
		return LineTotals{}, errors.New("invalid quantity, price or rate")
	}
	if priceCents > 0 && qtyMilli > math.MaxInt64/priceCents {
		return LineTotals{}, errors.New("amount overflow")
	}
	gross := roundDiv(qtyMilli*priceCents, 1000)
	if gross > math.MaxInt64/10000 {
		return LineTotals{}, errors.New("amount overflow")
	}
	sub := roundDiv(gross*int64(10000-discountBps), 10000)
	if sub > math.MaxInt64/10000 {
		return LineTotals{}, errors.New("amount overflow")
	}
	vat := roundDiv(sub*int64(vatBps), 10000)
	return LineTotals{sub, vat, sub + vat}, nil
}

func stockDelta(docType string, sourcedFromDelivery bool, qtyMilli int64) int64 {
	switch docType {
	case "DELIVERY", "RETURN":
		if docType == "RETURN" {
			return qtyMilli
		}
		return -qtyMilli
	case "PURCHASE_RECEIPT":
		return qtyMilli
	case "INVOICE":
		if !sourcedFromDelivery {
			return -qtyMilli
		}
	}
	return 0
}
func allowedConversion(from, to string) bool {
	switch from {
	case "QUOTE":
		return to == "ORDER" || to == "DELIVERY" || to == "INVOICE"
	case "ORDER":
		return to == "DELIVERY" || to == "INVOICE"
	case "DELIVERY":
		return to == "INVOICE" || to == "RETURN"
	case "INVOICE":
		return to == "CREDIT" || to == "RETURN"
	case "PURCHASE_ORDER":
		return to == "PURCHASE_RECEIPT" || to == "PURCHASE_INVOICE"
	case "PURCHASE_RECEIPT":
		return to == "PURCHASE_INVOICE"
	}
	return false
}
