package app

import "testing"

func TestCalculateLine(t *testing.T) {
	got, err := CalculateLine(2000, 15000, 1000, 2000)
	if err != nil || got.SubtotalCents != 27000 || got.VatCents != 5400 || got.TotalCents != 32400 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
func TestStockNoDoubleDebit(t *testing.T) {
	if stockDelta("DELIVERY", false, 2000) != -2000 || stockDelta("INVOICE", true, 2000) != 0 || stockDelta("INVOICE", false, 2000) != -2000 {
		t.Fatal("inventory semantics wrong")
	}
}
func TestConvert(t *testing.T) {
	if !allowedConversion("DELIVERY", "INVOICE") || allowedConversion("INVOICE", "DELIVERY") {
		t.Fatal("conversion rules")
	}
}
