package position

import (
	"fmt"
	"frizo/futures_engine/internal/common"
	"sync"
)

// UserPositions positionKey: Position
type UserPositions map[string]*Position

func (p *UserPositions) updateMarkPrice(prices map[string]float64) {
	for _, position := range *p {
		if markPrice, exists := prices[position.Symbol]; exists {
			position.UpdateMarkPrice(markPrice)
		}
	}
}

func (p *UserPositions) getLiquidatablePositions() []*Position {
	var liquidatable []*Position
	for _, position := range *p {
		if position.IsLiquidatable() {
			liquidatable = append(liquidatable, position)
		}
	}
	return liquidatable
}

func (p *UserPositions) hasOpenPosition() bool {
	for _, position := range *p {
		if position.Size > position.ZeroSize() {
			// if any position has non-zero size, means have open position.
			return true
		}
	}
	return false
}

// ==========================================================================================

// PositionManager (倉位管理器)
type PositionManager struct {
	userPositions   map[string]UserPositions // userID -> UserPosition
	symbolPositions *SymbolPositions         // symbol : *Position
	mode            map[string]PositionMode  // userID -> position mode
	mu              sync.RWMutex
}

// NewPositionManager new
func NewPositionManager(symbols []string) *PositionManager {
	return &PositionManager{
		userPositions:   make(map[string]UserPositions),
		symbolPositions: NewSymbolPositions(symbols),
		mode:            make(map[string]PositionMode),
	}
}

// GetPosition
func (pm *PositionManager) GetPosition(userID string, symbol string, side PositionSide) (*Position, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	userPositions, exists := pm.userPositions[userID]
	if !exists {
		return nil, fmt.Errorf("user does not have any userPositions")
	}

	mode, exists := pm.mode[userID]
	if !exists {
		return nil, fmt.Errorf("missing user position mode param")
	}
	// gen KEY by mode: 雙向持倉/單向持倉
	positionKey := getPositionKey(symbol, side, mode)

	position, exists := userPositions[positionKey]
	if !exists {
		return nil, fmt.Errorf("position does not exist")
	}

	return position, nil
}

// OpenPosition (開倉)
func (pm *PositionManager) OpenPosition(marginMode common.MarginMode, userID, symbol string, side PositionSide, price, size float64, leverage uint) (*Position, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// make sure userID in userPositions
	if _, exists := pm.userPositions[userID]; !exists {
		pm.userPositions[userID] = make(map[string]*Position)
		pm.mode[userID] = OneWayMode // default using 單向持倉
	}

	positionKey := getPositionKey(symbol, side, pm.mode[userID])

	// check user's position is exist
	if existingPosition, exists := pm.userPositions[userID][positionKey]; exists && existingPosition.Size > existingPosition.ZeroSize() {
		// if existing: Add() - 加倉
		err := existingPosition.Add(price, size)
		return existingPosition, err
	} else {
		// not exist: Open() - 開倉
		position := NewPosition(userID, symbol, marginMode, nil)
		err := position.Open(side, price, size, int16(leverage))
		if err != nil {
			return nil, err
		}
		// add position into manager cache.
		pm.userPositions[userID][positionKey] = position
		// add position into symbol array
		if err = pm.symbolPositions.AddPosition(symbol, position); err != nil {
			return nil, err
		}

		return position, nil
	}
}

// ClosePosition (關倉/全部平倉) return PnL
func (pm *PositionManager) ClosePosition(userID, symbol string, side PositionSide, price float64) (*Position, float64, error) {
	position, err := pm.GetPosition(userID, symbol, side)
	if err != nil {
		return position, 0.0, err
	}
	pnl, err := position.Close(price)
	if err != nil {
		return position, 0.0, err
	}

	// remove position from pm
	positionKey := getPositionKey(symbol, side, pm.mode[userID])
	if userPosition, exists := pm.userPositions[positionKey]; exists {
		delete(userPosition, positionKey)
	}

	return position, pnl, nil
}

// ReducePosition (減倉/部分平倉) return PnL
func (pm *PositionManager) ReducePosition(userID, symbol string, side PositionSide, price, size float64) (*Position, float64, error) {
	position, err := pm.GetPosition(userID, symbol, side)
	if err != nil {
		return position, 0.0, err
	}
	pnl, err := position.Reduce(price, size)
	if err != nil {
		return position, pnl, err
	}

	if position.Status == PositionClosed {
		// remove position from pm
		positionKey := getPositionKey(symbol, side, pm.mode[userID])
		if userPosition, exists := pm.userPositions[positionKey]; exists {
			delete(userPosition, positionKey)
		}
	}

	return position, pnl, nil
}

// UpdateMarkPrices batch update mark price - input prices (symbol: markPrice)
func (pm *PositionManager) UpdateMarkPrices(symbol string, price float64) ([]*Position, error) {
	return pm.symbolPositions.UpdateMarkPrice(symbol, price)
}

// GetLiquidatablePositions (取得所有可強平倉位)
func (pm *PositionManager) GetLiquidatablePositions() []*Position {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var liquidatable []*Position
	for _, userPositions := range pm.userPositions {
		liquidatable = append(liquidatable, userPositions.getLiquidatablePositions()...)
	}

	return liquidatable
}

// SetPositionMode (設定雙向/單向持倉)
func (pm *PositionManager) SetPositionMode(userID string, mode PositionMode) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// check if still have open position:
	if userPositions, exists := pm.userPositions[userID]; exists {
		if userPositions.hasOpenPosition() {
			return fmt.Errorf("cannot change position mode with open userPositions")
		}
	}

	pm.mode[userID] = mode

	return nil
}

func (pm *PositionManager) GetUserPositions(userID string) ([]*Position, error) {
	if userPositions, exists := pm.userPositions[userID]; exists {
		positions := make([]*Position, 0, len(userPositions))
		for _, position := range userPositions {
			positions = append(positions, position)
		}
		return positions, nil
	} else {
		return nil, fmt.Errorf("cannot get positions for user %s", userID)
	}
}

// ============================================================================================================
// private func
// ============================================================================================================

// getPositionKey get position key by symbol, side, mode
func getPositionKey(symbol string, side PositionSide, mode PositionMode) string {
	switch mode {
	case HedgeMode:
		// 雙向持倉模式: 區分多空 long_BTC-USDT or short-BTC-USDT
		return fmt.Sprintf("%s_%s", side.String(), symbol)
	case OneWayMode:
		// 單向持倉模式: 不區分多空
		return symbol
	default:
		panic(fmt.Sprintf("unknown position mode %v", mode))
	}
}
