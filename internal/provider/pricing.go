package provider

// CostCalculator computes USD expenditure for tokens.
type CostCalculator struct {
	CostPerMillionInput  float64
	CostPerMillionOutput float64
}

// NewCostCalculator creates a calculator with given rates.
func NewCostCalculator(costPerMillionIn, costPerMillionOut float64) *CostCalculator {
	return &CostCalculator{
		CostPerMillionInput:  costPerMillionIn,
		CostPerMillionOutput: costPerMillionOut,
	}
}

// ComputeCost returns the total USD cost for prompt and completion tokens.
func (c *CostCalculator) ComputeCost(promptTokens, completionTokens int) float64 {
	inputCost := (float64(promptTokens) / 1000000.0) * c.CostPerMillionInput
	outputCost := (float64(completionTokens) / 1000000.0) * c.CostPerMillionOutput
	return inputCost + outputCost
}

// ComputeSavings calculates USD savings and percentage compared to baseline frontier pricing.
func ComputeSavings(baselineCost, actualCost float64) (savingsUSD float64, savingsPct float64) {
	savingsUSD = baselineCost - actualCost
	if baselineCost > 0 {
		savingsPct = (savingsUSD / baselineCost) * 100.0
	}
	return savingsUSD, savingsPct
}
