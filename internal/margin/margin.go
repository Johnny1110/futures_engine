package margin

import (
	"fmt"
	"frizo/futures_engine/internal/position"
	"sync"
)

// MarginSystem Main type
type MarginSystem struct {
	accounts     map[string]*MarginAccount     // userID -> account
	requirements map[string]*MarginRequirement // symbol -> requirement

	// position manager
	positionMgr *position.PositionManager
	// config
	config *MarginConfig

	mu sync.RWMutex
}

// NewMarginSystem
func NewMarginSystem(positionMgr *position.PositionManager, config *MarginConfig) *MarginSystem {
	if config == nil {
		config = &MarginConfig{
			DefaultInitialMarginRate:     0.10, // 10%
			DefaultMaintenanceMarginRate: 0.05, // 5%
			MinTransferAmount:            1.0,
			NegativeBalanceProtection:    true,
		}
	}

	return &MarginSystem{
		accounts:     make(map[string]*MarginAccount),
		requirements: make(map[string]*MarginRequirement),
		positionMgr:  positionMgr,
		config:       config,
	}
}

func (ms *MarginSystem) GetAccount(userID string) (*MarginAccount, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if account, ok := ms.accounts[userID]; ok {
		return account, nil
	} else {
		return nil, fmt.Errorf("account not found")
	}
}

func (ms *MarginSystem) CreateAccount(userID string) (*MarginAccount, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, ok := ms.accounts[userID]; ok {
		return nil, fmt.Errorf("account already exists")
	} else {
		// create margin account
		ma := NewMarginAccount(userID)
		ms.accounts[userID] = ma
		return ma, nil
	}
}

// =====================================================
// Calculate Margin
// =====================================================

// CalculateInitialMargin
func (ms *MarginSystem) CalculateInitialMargin(symbol string, size, price float64, leverage int16) (float64, error) {
	requirement := ms.getRequirement(symbol)

	positionValue := size * price

	initialMarginByLeverage := positionValue / float64(leverage)
	initialMarginByRate := positionValue * requirement.InitialMarginRate

	initialMargin := max(initialMarginByLeverage, initialMarginByRate)

	// check min margin
	if initialMargin < requirement.MinInitialMargin {
		initialMargin = requirement.MinInitialMargin
	}

	return initialMargin, nil
}

// CalculateMaintenanceMargin
func (ms *MarginSystem) CalculateMaintenanceMargin(symbol string, positionValue float64) float64 {
	requirement := ms.getRequirement(symbol)
	for _, tier := range requirement.TierBrackets {
		if positionValue >= tier.MinValue && positionValue < tier.MaxValue {
			return positionValue * tier.MaintenanceRate
		}
	}

	// return default
	return positionValue * requirement.MaintenanceMarginRate
}

// =====================================================
// price check
// =====================================================

// CheckOrderMargin
func (ms *MarginSystem) CheckOrderMargin(userID, symbol string, size, price float64, leverage int16) error {

	ms.mu.Lock()
	defer ms.mu.Unlock()

	account, err := ms.GetAccount(userID)
	if err != nil {
		return err
	}

	// calculate initial Margin
	requiredMargin, err := ms.CalculateInitialMargin(symbol, size, price, leverage)
	if err != nil {
		return err
	}

	if account.AvailableBalance < requiredMargin {
		return fmt.Errorf("insufficient margin: required %.2f, available %.2f",
			requiredMargin, account.AvailableBalance)
	}

	return nil
}

// FreezeOrderMargin (開倉凍結)
func (ms *MarginSystem) FreezeOrderMargin(userID string, amount float64) error {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if account, ok := ms.accounts[userID]; ok {
		return account.FreezeOrderMargin(amount)
	} else {
		return fmt.Errorf("account not found")
	}
}

// UnfreezeOrderMargin (解凍)
func (ms *MarginSystem) UnfreezeOrderMargin(userID string, amount float64) error {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if account, ok := ms.accounts[userID]; ok {
		return account.UnFreezeOrderMargin(amount)
	} else {
		return fmt.Errorf("account not found")
	}
}

// =====================================================
// Position Margin Management
// =====================================================

// UpdatePositionMargin
func (ms *MarginSystem) UpdatePositionMargin(userID string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	account, exists := ms.accounts[userID]
	if !exists {
		return fmt.Errorf("account not found")
	}

	positions, err := ms.positionMgr.GetUserPositions(userID)
	if err != nil {
		return err
	}

	totalPositionMargin := 0.0
	totalUnrealizedPnL := 0.0

	for _, pos := range positions {
		if pos.Status != position.PositionClosed {
			// 計算倉位保證金
			positionMargin := pos.InitialMargin
			totalPositionMargin += positionMargin

			// 累計未實現盈虧
			totalUnrealizedPnL += pos.UnrealizedPnL
		}
	}

	account.UpdateMarginAndPnl(totalPositionMargin, totalUnrealizedPnL)

	return nil
}

// =====================================================
// About Risk
// =====================================================

// GetMarginLevel
func (ms *MarginSystem) GetMarginLevel(userID string) (float64, error) {
	account, err := ms.GetAccount(userID)
	if err != nil {
		return 0, err
	}

	accountEquity := account.GetAccountEquity()
	usedMargin := account.GetUsedMargin()

	if usedMargin <= 0 {
		return 999, nil // no order and position -> return max
	} else {
		// 保證金水平 = 賬戶權益 / 已用保證金
		return accountEquity / usedMargin, nil
	}
}

// IsLiquidatable
func (ms *MarginSystem) IsLiquidatable(userID string) (bool, error) {
	marginLevel, err := ms.GetMarginLevel(userID)
	if err != nil {
		return false, err
	}

	// 保證金水平低於 1.0 時可被強平
	return marginLevel < 1.0, nil
}

// =====================================================
// Settlement
// =====================================================

// Deposit
func (ms *MarginSystem) Deposit(userID string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}
	account, err := ms.GetAccount(userID)
	if err != nil {
		return err
	}

	account.Deposit(amount)
	return nil
}

// Withdraw
func (ms *MarginSystem) Withdraw(userID string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	account, err := ms.GetAccount(userID)
	if err != nil {
		return err
	}

	return account.Withdraw(amount)
}

// =====================================================
// support methods
// =====================================================

// SetSymbolRequirement
func (ms *MarginSystem) SetSymbolRequirement(symbol string, req *MarginRequirement) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.requirements[symbol] = req
}

// GetAccountSummary
func (ms *MarginSystem) GetAccountSummary(userID string) (map[string]interface{}, error) {
	account, err := ms.GetAccount(userID)
	if err != nil {
		return nil, err
	}

	return account.GetSummary()
}

// =====================================================
// tool methods
// =====================================================

func (ms *MarginSystem) getRequirement(symbol string) *MarginRequirement {
	if req, exists := ms.requirements[symbol]; exists {
		return req
	}

	// default
	return &MarginRequirement{
		Symbol:                symbol,
		InitialMarginRate:     ms.config.DefaultInitialMarginRate,
		MaintenanceMarginRate: ms.config.DefaultMaintenanceMarginRate,
		MinInitialMargin:      1.0,
		MaxLeverage:           125,
	}
}
