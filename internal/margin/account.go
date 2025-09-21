package margin

import (
	"fmt"
	"sync"
	"time"
)

// MarginAccount (保證金帳戶)
type MarginAccount struct {
	UserID string

	// Balance etc.
	Balance          float64 // total balance
	AvailableBalance float64 // available balance
	FrozenBalance    float64 // frozen balance（掛單凍結）

	PositionMargin float64 // 倉位保證金 -> 當訂單成交後，實際持有倉位所占用的保證金
	OrderMargin    float64 // 委託保證金 -> 當 User 下了限價單但還未成交時，凍結的保證金

	// PnL
	UnrealizedPnL float64 // 未實現盈虧
	RealizedPnL   float64 // 已實現盈虧

	// risk
	MarginLevel float64 // 保證金水平
	MarginRatio float64 // 保證金率

	UpdatedAt time.Time

	mu sync.RWMutex
}

func NewMarginAccount(userID string) *MarginAccount {
	return &MarginAccount{
		UserID:    userID,
		UpdatedAt: time.Now(),
	}
}

func (a *MarginAccount) FreezeOrderMargin(amount float64) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	if amount > a.AvailableBalance {
		return fmt.Errorf("available balance not enough")
	}

	a.AvailableBalance -= amount
	if a.AvailableBalance < 0 {
		a.AvailableBalance = 0
	}
	a.FrozenBalance += amount
	a.OrderMargin += amount
	a.UpdatedAt = time.Now()

	return nil
}

func (a *MarginAccount) UnFreezeOrderMargin(amount float64) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	if amount > a.FrozenBalance {
		return fmt.Errorf("frozen balance not enough to unfreeze")
	}

	a.AvailableBalance += amount
	a.FrozenBalance -= amount
	if a.FrozenBalance < 0 {
		a.FrozenBalance = 0
	}
	a.OrderMargin -= amount
	if a.OrderMargin < 0 {
		a.OrderMargin = 0
	}

	a.UpdatedAt = time.Now()

	return nil

}

func (ma *MarginAccount) UpdateMarginAndPnl(margin float64, pnl float64) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	ma.PositionMargin = margin
	ma.UnrealizedPnL = pnl

	// 總餘額 + 未實現損益 - 已有倉位的保證金 - 尚未成交的鎖倉保證金
	availableBalance := ma.Balance + ma.UnrealizedPnL - ma.PositionMargin - ma.OrderMargin

	if availableBalance < 0 {
		availableBalance = 0
	}

	// 計算 margin ratio
	if margin > 0 {
		accountEquity := ma.Balance + ma.UnrealizedPnL
		ma.MarginRatio = accountEquity / margin
	} else {
		ma.MarginRatio = 999.99 // 無倉位時設為最大值
	}

	ma.UpdatedAt = time.Now()
}

func (ma *MarginAccount) GetAccountEquity() float64 {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	return ma.Balance + ma.UnrealizedPnL
}

func (ma *MarginAccount) GetUsedMargin() float64 {
	ma.mu.RLock()
	defer ma.mu.RUnlock()
	// 已開倉保證金 + 未開倉保證金（未成交）
	return ma.PositionMargin + ma.OrderMargin
}

// Deposit
func (ma *MarginAccount) Deposit(amount float64) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	ma.Balance += amount
	ma.AvailableBalance += amount
	ma.UpdatedAt = time.Now()
}

// Withdraw
func (ma *MarginAccount) Withdraw(amount float64) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if ma.AvailableBalance < amount {
		return fmt.Errorf("insufficient available balance: %.2f < %.2f",
			ma.AvailableBalance, amount)
	}

	ma.Balance -= amount
	if ma.Balance < 0 {
		ma.Balance = 0
	}
	ma.AvailableBalance -= amount
	if ma.AvailableBalance < 0 {
		ma.AvailableBalance = 0
	}
	ma.UpdatedAt = time.Now()

	return nil
}

func (ma *MarginAccount) GetSummary() (map[string]interface{}, error) {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	accountEquity := ma.GetAccountEquity()

	summary := map[string]interface{}{
		"user_id":           ma.UserID,
		"balance":           ma.Balance,
		"available_balance": ma.AvailableBalance,
		"position_margin":   ma.PositionMargin,
		"order_margin":      ma.OrderMargin,
		"unrealized_pnl":    ma.UnrealizedPnL,
		"realized_pnl":      ma.RealizedPnL,
		"account_equity":    accountEquity,
		"margin_ratio":      ma.MarginRatio,
		"margin_level":      ma.MarginLevel,
		"updated_at":        ma.UpdatedAt,
	}

	return summary, nil
}
