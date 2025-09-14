package position

import (
	"fmt"
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
	positions map[string]UserPositions // userID -> UserPosition
	mode      map[string]PositionMode  // userID -> position mode
	mu        sync.RWMutex
}

// NewPositionManager new
func NewPositionManager() *PositionManager {
	return &PositionManager{
		positions: make(map[string]UserPositions),
		mode:      make(map[string]PositionMode),
	}
}

// GetPosition
func (pm *PositionManager) GetPosition(userID string, symbol string, side PositionSide) (*Position, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	userPositions, exists := pm.positions[userID]
	if !exists {
		return nil, fmt.Errorf("user does not have any positions")
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
func (pm *PositionManager) OpenPosition(marginMode MarginMode, userID, symbol string, side PositionSide, price, size float64, leverage uint) (*Position, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// make sure userID in positions
	if _, exists := pm.positions[userID]; !exists {
		pm.positions[userID] = make(map[string]*Position)
		pm.mode[userID] = OneWayMode // default using 單向持倉
	}

	positionKey := getPositionKey(symbol, side, pm.mode[userID])

	// check user's position is exist
	if existingPosition, exists := pm.positions[userID][positionKey]; exists && existingPosition.Size > existingPosition.ZeroSize() {
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
		pm.positions[userID][positionKey] = position
		return position, nil
	}
}

// ClosePosition (關倉/全部平倉) return PnL
func (pm *PositionManager) ClosePosition(userID, symbol string, side PositionSide, price float64) (float64, error) {
	position, err := pm.GetPosition(userID, symbol, side)
	if err != nil {
		return 0.0, err
	}
	return position.Close(price)
}

// ReducePosition (減倉/部分平倉) return PnL
func (pm *PositionManager) ReducePosition(userID, symbol string, side PositionSide, price, size float64) (float64, error) {
	position, err := pm.GetPosition(userID, symbol, side)
	if err != nil {
		return 0.0, err
	}
	return position.Reduce(price, size)
}

// UpdateMarkPrices batch update mark price - input prices (symbol: markPrice)
func (pm *PositionManager) UpdateMarkPrices(prices map[string]float64) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// TODO: 效能瓶頸在這裡，updateMarkPrice 被註解起來，一樣會非常耗時。把下面程式碼全都註解，就可以非常快
	// TODO: 走訪 pm.position 很慢
	// TODO: goroutine 也慢
	// block until all position complete price update.
	var wg sync.WaitGroup
	for _, userPositions := range pm.positions { // 這裏耗時
		wg.Add(1)
		go func(up UserPositions) {
			defer wg.Done()
			up.updateMarkPrice(prices) // 這個動作幾乎不會耗時，問題不在這裡
		}(userPositions)
	}
	wg.Wait()
}

// GetLiquidatablePositions (取得所有可強平倉位)
func (pm *PositionManager) GetLiquidatablePositions() []*Position {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var liquidatable []*Position
	for _, userPositions := range pm.positions {
		liquidatable = append(liquidatable, userPositions.getLiquidatablePositions()...)
	}

	return liquidatable
}

// SetPositionMode (設定雙向/單向持倉)
func (pm *PositionManager) SetPositionMode(userID string, mode PositionMode) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// check if still have open position:
	if userPositions, exists := pm.positions[userID]; exists {
		if userPositions.hasOpenPosition() {
			return fmt.Errorf("cannot change position mode with open positions")
		}
	}

	pm.mode[userID] = mode

	return nil
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
