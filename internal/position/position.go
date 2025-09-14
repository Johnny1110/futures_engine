package position

import (
	"fmt"
	"frizo/futures_engine/common"
	"math"
	"sync"
	"time"
)

// Position 倉位
type Position struct {
	// basic info
	ID     string         `json:"id"`
	UserID string         `json:"user_id"`
	Symbol string         `json:"symbol"`
	Side   PositionSide   `json:"side"`
	Status PositionStatus `json:"status"`

	// position info (decimal)
	Size             float64 `json:"size"`
	EntryPrice       float64 `json:"entry_price"`       // 開倉價格
	MarkPrice        float64 `json:"mark_price"`        // 標記價格
	PositionValue    float64 `json:"position_value"`    // 倉位價值 cache (MarkPrice*Size)
	LiquidationPrice float64 `json:"liquidation_price"` // 強平價格

	// margin info (decimal)
	InitialMargin     float64    `json:"initial_margin"`     // 初始保證金 (放進倉位鎖定的錢)
	MaintenanceMargin float64    `json:"maintenance_margin"` // 維持保證金
	Leverage          int16      `json:"leverage"`
	MarginMode        MarginMode `json:"margin_mode"`

	// PnL info (decimal)
	RealizedPnL   float64 `json:"realized_pnl"`   // 已實現盈虧
	UnrealizedPnL float64 `json:"unrealized_pnl"` // 未實現盈虧

	// Timestamp
	OpenTime   time.Time `json:"open_time"`
	UpdateTime time.Time `json:"update_time"`

	// === Precision Control ===
	sizePrecision  int8
	pricePrecision int8

	// Lock
	mu sync.RWMutex
}

// NewPosition create a init position
func NewPosition(userID, symbol string, mode MarginMode, precisionSetting *PrecisionSetting) *Position {
	if precisionSetting == nil {
		precisionSetting = DefaultPrecisionSetting
	}

	return &Position{
		ID:             common.GeneratePositionID(),
		UserID:         userID,
		Symbol:         symbol,
		MarginMode:     mode,
		Status:         PositionNormal,
		Size:           0.0,
		EntryPrice:     0.0,
		RealizedPnL:    0.0,
		UnrealizedPnL:  0.0,
		pricePrecision: precisionSetting.PricePrecision,
		sizePrecision:  precisionSetting.SizePrecision,
		OpenTime:       time.Now(),
		UpdateTime:     time.Now(),
	}
}

// Open Position (開倉)
func (p *Position) Open(side PositionSide, price float64, size float64, leverage int16) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Status != PositionNormal || p.Size > p.ZeroSize() {
		return fmt.Errorf("position already exists, can not open again")
	}

	p.Side = side
	p.EntryPrice = price
	p.MarkPrice = price
	p.Size = size
	p.Leverage = leverage

	// Calculate Margin
	p.updateMarkPriceAndPositionVal(price)
	p.InitialMargin = p.PositionValue / float64(leverage)
	p.MaintenanceMargin = p.calculateMaintenanceMargin()
	// Calculate Liquidation Price
	p.LiquidationPrice = p.calculateLiquidationPrice()
	// time
	p.UpdateTime = time.Now()

	return nil
}

// Add position (加倉)
func (p *Position) Add(price float64, size float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Status != PositionNormal {
		return fmt.Errorf("add position failed, position status is not normal")
	}

	// calculate new open price
	// formula: new average price = (current position val + new position val) / (current position + new position)
	oldValue := p.EntryPrice * p.Size // 舊倉位額度
	newValue := price * size          // 補倉倉位額度
	totalValue := oldValue + newValue // 合併倉位額度
	totalSize := p.Size + size        // 合併 Size

	// update entry-price & size
	p.EntryPrice = totalValue / totalSize
	p.Size = totalSize

	// update mark price & position value (no lock)
	p.updateMarkPriceAndPositionVal(price)

	// update margin
	marginValue := p.EntryPrice * totalSize
	p.InitialMargin = marginValue / float64(p.Leverage)
	p.MaintenanceMargin = p.calculateMaintenanceMargin()
	// update l price
	p.LiquidationPrice = p.calculateLiquidationPrice()
	// update time
	p.UpdateTime = time.Now()

	return nil
}

// Reduce position (減倉) return pnl, error
func (p *Position) Reduce(price float64, size float64) (pnl float64, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Status != PositionNormal {
		return pnl, fmt.Errorf("reduce position failed, position status is not normal")
	}

	if size > p.Size {
		return pnl, fmt.Errorf("reduce position failed, reduce size exceeds position size")
	}

	// calculate and update Realized PnL
	if p.Side == LONG { // calculate long side
		// long_pnl = (price - EntryPrice) * size
		pnl = (price - p.EntryPrice) * size
	} else { // calculate short side
		// short_pnl = (EntryPrice - price) * size
		pnl = (p.EntryPrice - price) * size
	}
	p.RealizedPnL = p.RealizedPnL + pnl

	// reduce position size
	p.Size = p.Size - size

	// update markPrice and position val
	p.updateMarkPriceAndPositionVal(price)

	if p.Size <= p.ZeroSize() { // is size is zero -> close position
		p.Status = PositionClosed
		p.Size = 0.0
		p.PositionValue = 0.0
		p.InitialMargin = 0.0
		p.MaintenanceMargin = 0.0
	} else { // update maintenance margin
		marginValue := p.EntryPrice * p.Size
		p.InitialMargin = marginValue / float64(p.Leverage)
		p.MaintenanceMargin = p.calculateMaintenanceMargin()
	}

	p.calculateLiquidationPrice()

	// update time
	p.UpdateTime = time.Now()

	return pnl, nil
}

// Close position（全部平倉）
func (p *Position) Close(price float64) (float64, error) {
	// reduce all size left.
	return p.Reduce(price, p.Size)
}

// UpdateMarkPrice (更新標記價格)
func (p *Position) UpdateMarkPrice(markPrice float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.updateMarkPriceAndPositionVal(markPrice)
}

// GetMarginRatio (保證金率)
func (p *Position) GetMarginRatio() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.getMarginRatio()
}

// IsLiquidatable (可清算)
func (p *Position) IsLiquidatable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.Status != PositionNormal || p.Size <= p.ZeroSize() || p.MarkPrice <= p.ZeroPrice() {
		return false
	}

	marginRatio := p.getMarginRatio()

	// marginRatio:      (MarginAccountEquity / PositionValue) * 100%
	// maintenanceRatio: (MaintenanceMargin / PositionValue) * 100%
	maintenanceRatio := p.MaintenanceMargin / p.PositionValue * 100
	// 如果沒有 MaintenanceMargin，可以理解為 marginRatio 降低到 0 既為可被清算
	return marginRatio <= maintenanceRatio
}

// GetRoi (投資報酬率)
func (p *Position) GetRoi() float64 {
	return p.UnrealizedPnL / p.InitialMargin
}

// GetDisplayInfo（用於顯示）
func (p *Position) GetDisplayInfo() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"id":                p.ID,
		"user_id":           p.UserID,
		"symbol":            p.Symbol,
		"side":              p.Side.String(),
		"size":              p.Size,
		"entry_price":       p.EntryPrice,
		"mark_price":        p.MarkPrice,
		"initial_margin":    p.InitialMargin,
		"liquidation_price": p.LiquidationPrice,
		"leverage":          p.Leverage,
		"margin_mode":       p.MarginMode,
		"unrealized_pnl":    p.UnrealizedPnL,
		"realized_pnl":      p.RealizedPnL,
		"margin_ratio":      fmt.Sprintf("%2f", math.Round(p.getMarginRatio())) + "%",
		"is_liquidatable":   p.IsLiquidatable(),
	}
}

// --------------------------------------------------------------------------------------------
// private func
// --------------------------------------------------------------------------------------------

// calculateMaintenanceMargin calculate Maintenance Margin value
func (p *Position) calculateMaintenanceMargin() float64 {
	for _, t := range DefaultMarginTiers {
		if p.PositionValue >= t.MinValue && p.PositionValue <= t.MaxValue {
			return p.PositionValue * t.MaintenanceRate
		}
	}
	return 0
}

// calculateLiquidationPrice (強平價格)
func (p *Position) calculateLiquidationPrice() float64 {
	if p.Size <= 0 {
		return 0
	}

	// Liquidation Price Formula:
	// (LONG) :  LiquidationPrice = EntryPrice - (InitialMargin - MaintenanceMargin) / Size
	// (SHORT):  LiquidationPrice = EntryPrice + (InitialMargin - MaintenanceMargin) / Size

	// 理解公式：假如初始押金我投入 10000 USDT，我開倉數量為 10 顆 FZO 幣，不考慮維持保證金情況下，我每一顆 FZO 的押金是 10000/10 = 1000
	// 		   相當於我每一個 FZO 幣最多虧損 1000 元就應概要被強制平倉．
	//         假如我的開倉價為 100000 USDT：
	//        		多頭強平價就是 100000-1000 = 99000  USDT
	//        		空頭強平價就是 100000+1000 = 110000 USDT

	marginBuffer := p.InitialMargin - p.MaintenanceMargin // 保證金緩衝額 = 初始放入的押金 - 滑價保險額度
	priceBuffer := marginBuffer / p.Size                  // 價格緩衝額   = 保證金緩衝額 / 倉位數量

	if p.Side == LONG { // LONG side
		p.LiquidationPrice = p.EntryPrice - priceBuffer
	} else { // SHORT side
		p.LiquidationPrice = p.EntryPrice + priceBuffer
	}

	return p.LiquidationPrice
}

// UpdateMarkPrice (更新標記價格) 無鎖
func (p *Position) updateMarkPriceAndPositionVal(markPrice float64) {
	p.MarkPrice = markPrice

	if p.Size <= p.ZeroSize() {
		p.PositionValue = 0
		p.UnrealizedPnL = 0
	}

	// calculate unrealized PnL
	if p.Side == LONG { // long side
		// formula = (markPrice - entryPrice) * size
		p.UnrealizedPnL = (markPrice - p.EntryPrice) * p.Size
	} else { // short side
		// formula = (entryPrice - markPrice) * size
		p.UnrealizedPnL = (p.EntryPrice - markPrice) * p.Size
	}

	p.PositionValue = p.MarkPrice * p.Size
}

// getMarginRatio (保證金率) no lock
func (p *Position) getMarginRatio() float64 {
	if p.Size <= p.ZeroSize() || p.MarkPrice <= p.ZeroPrice() {
		return 100 // safe
	}

	if p.PositionValue <= 0 {
		return 100
	}

	// MarginRatio Formula:
	// MarginRatio = (MarginAccount Equity Value / Position Value) * 100%

	// cross: TODO
	// Isolated: (InitialMargin + UnrealizedPnL) / (MarkPrice * Size)
	accountEquity := 0.0
	switch p.MarginMode {
	case CROSS:
		panic("not implemented cross mode yet")
	case ISOLATED:
		accountEquity = p.InitialMargin + p.UnrealizedPnL
	}

	return accountEquity / p.PositionValue * 100
}

func (p *Position) ZeroSize() float64 {
	return math.Pow(10, -float64(p.sizePrecision))
}

func (p *Position) ZeroPrice() float64 {
	return math.Pow(10, -float64(p.pricePrecision))
}
