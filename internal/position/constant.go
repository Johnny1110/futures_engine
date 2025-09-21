package position

import (
	"math"
)

var DefaultMarginTiers = []MarginTier{
	{0, 50000, 0.004, 125},           // 0.4% for userPositions < 50k USDT
	{50000, 250000, 0.005, 100},      // 0.5% for 50k-250k
	{250000, 1000000, 0.01, 50},      // 1.0% for 250k-1M
	{1000000, 5000000, 0.025, 20},    // 2.5% for 1M-5M
	{5000000, 10000000, 0.05, 10},    // 5.0% for 5M-10M
	{10000000, 20000000, 0.1, 5},     // 10% for 10M-20M
	{20000000, 50000000, 0.125, 4},   // 12.5% for 20M-50M
	{50000000, math.Inf(1), 0.15, 3}, // 15% for > 50M
}

var DefaultPrecisionSetting = &PrecisionSetting{
	PricePrecision: 2,
	SizePrecision:  8,
}
