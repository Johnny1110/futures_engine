package margin

import "frizo/futures_engine/internal/position"

// MarginRequirement (保證金要求)
type MarginRequirement struct {
	Symbol                string
	InitialMarginRate     float64 // 初始保證金率
	MaintenanceMarginRate float64 // 維持保證金率
	MinInitialMargin      float64 // 最小初始保證金
	MaxLeverage           int16   // 最大槓桿

	// 階梯費率
	TierBrackets []position.MarginTier
}
