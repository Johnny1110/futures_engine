package margin

// MarginConfig
type MarginConfig struct {
	DefaultInitialMarginRate     float64 // 默認初始保證金率
	DefaultMaintenanceMarginRate float64 // 默認維持保證金率
	MinTransferAmount            float64 // 最小劃轉金額
	AutoBorrowEnabled            bool    // 是否自動借貸
	NegativeBalanceProtection    bool    // 負餘額保護
}
