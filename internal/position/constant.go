package position

import (
	"github.com/shopspring/decimal"
	"math"
)

var DefaultMarginTiers = []MarginTier{
	{0, 50000, decimal.NewFromFloat(0.004), 125},           // 0.4% for positions < 50k USDT
	{50000, 250000, decimal.NewFromFloat(0.005), 100},      // 0.5% for 50k-250k
	{250000, 1000000, decimal.NewFromFloat(0.01), 50},      // 1.0% for 250k-1M
	{1000000, 5000000, decimal.NewFromFloat(0.025), 20},    // 2.5% for 1M-5M
	{5000000, 10000000, decimal.NewFromFloat(0.05), 10},    // 5.0% for 5M-10M
	{10000000, 20000000, decimal.NewFromFloat(0.1), 5},     // 10% for 10M-20M
	{20000000, 50000000, decimal.NewFromFloat(0.125), 4},   // 12.5% for 20M-50M
	{50000000, math.Inf(1), decimal.NewFromFloat(0.15), 3}, // 15% for > 50M
}
