package provider

import (
	"testing"
)

func TestPricingCalculator(t *testing.T) {
	calc := NewCostCalculator(0.075, 0.30)

	// 1000 input tokens, 500 output tokens
	cost := calc.ComputeCost(1000, 500)
	expected := (1000.0/1000000.0)*0.075 + (500.0/1000000.0)*0.30

	if cost != expected {
		t.Errorf("expected cost %.8f, got %.8f", expected, cost)
	}

	// Compute savings
	baseline := 0.0150 // $0.015
	actual := 0.0015   // $0.0015
	savedUSD, savedPct := ComputeSavings(baseline, actual)

	if savedUSD != 0.0135 {
		t.Errorf("expected saved USD 0.0135, got %.4f", savedUSD)
	}
	if savedPct != 90.0 {
		t.Errorf("expected saved pct 90%%, got %.1f", savedPct)
	}
}
