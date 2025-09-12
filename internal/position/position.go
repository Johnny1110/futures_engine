package position

import (
	"fmt"
	"frizo/futures_engine/common"
	"github.com/shopspring/decimal"
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
	Size             decimal.Decimal `json:"size"`
	EntryPrice       decimal.Decimal `json:"entry_price"`       // 開倉價格
	MarkPrice        decimal.Decimal `json:"mark_price"`        // 標記價格
	LiquidationPrice decimal.Decimal `json:"liquidation_price"` // 強平價格

	// margin info (decimal)
	InitialMargin     decimal.Decimal `json:"initial_margin"`     // 初始保證金 (放進倉位鎖定的錢)
	MaintenanceMargin decimal.Decimal `json:"maintenance_margin"` // 維持保證金
	Leverage          uint            `json:"leverage"`
	MarginMode        MarginMode      `json:"margin_mode"`

	// PnL info (decimal)
	RealizedPnL   decimal.Decimal `json:"realized_pnl"`   // 已實現盈虧
	UnrealizedPnL decimal.Decimal `json:"unrealized_pnl"` // 未實現盈虧

	// Timestamp
	OpenTime   time.Time `json:"open_time"`
	UpdateTime time.Time `json:"update_time"`

	// Internal Usage - float64 for calculating
	sizeFloat       float64
	entryPriceFloat float64

	// Lock
	mu sync.RWMutex
}

// NewPosition create a init position
func NewPosition(userID, symbol string, mode MarginMode) *Position {
	return &Position{
		ID:              common.GeneratePositionID(),
		UserID:          userID,
		Symbol:          symbol,
		MarginMode:      mode,
		Status:          PositionNormal,
		Size:            decimal.Zero,
		EntryPrice:      decimal.Zero,
		RealizedPnL:     decimal.Zero,
		UnrealizedPnL:   decimal.Zero,
		OpenTime:        time.Now(),
		UpdateTime:      time.Now(),
		sizeFloat:       0.0,
		entryPriceFloat: 0.0,
	}
}

// Open Position (開倉)
func (p *Position) Open(side PositionSide, price float64, size float64, leverage uint) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Status != PositionNormal || !p.Size.IsZero() {
		return fmt.Errorf("position already exists, can not open again")
	}

	priceD := decimal.NewFromFloat(price)
	sizeD := decimal.NewFromFloat(size)

	p.Side = side
	p.EntryPrice = priceD
	p.Size = sizeD
	p.Leverage = leverage

	// Calculate Margin
	positionValue := priceD.Mul(sizeD) // position value
	p.InitialMargin = positionValue.Div(decimal.NewFromUint64(uint64(leverage)))
	p.MaintenanceMargin = p.calculateMaintenanceMargin(positionValue)

	// cache price & size with float64
	p.entryPriceFloat = price
	p.sizeFloat = size
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

	priceD := decimal.NewFromFloat(price)
	sizeD := decimal.NewFromFloat(size)

	// calculate new open price
	// formula: new average price = (current position val + new position val) / (current position + new position)
	oldValue := p.EntryPrice.Mul(p.Size) // 舊倉位額度
	newValue := priceD.Mul(sizeD)        // 補倉倉位額度
	totalValue := oldValue.Add(newValue) // 合併倉位額度
	totalSize := p.Size.Add(sizeD)       // 合併 Size

	// update entry-price & size
	p.EntryPrice = totalValue.Div(totalSize)
	p.Size = totalSize

	// update margin
	positionValue := p.EntryPrice.Mul(totalSize)
	p.InitialMargin = positionValue.Div(decimal.NewFromUint64(uint64(p.Leverage)))
	p.MaintenanceMargin = p.calculateMaintenanceMargin(positionValue)

	// update cache
	p.entryPriceFloat = p.EntryPrice.InexactFloat64()
	p.sizeFloat = p.Size.InexactFloat64()
	// update time
	p.UpdateTime = time.Now()

	return nil
}

// Reduce position (減倉) return pnl, error
func (p *Position) Reduce(price float64, size float64) (pnl decimal.Decimal, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Status != PositionNormal {
		return pnl, fmt.Errorf("reduce position failed, position status is not normal")
	}

	priceD := decimal.NewFromFloat(price)
	sizeD := decimal.NewFromFloat(size)
	if sizeD.GreaterThan(p.Size) {
		return pnl, fmt.Errorf("reduce position failed, reduce size exceeds position size")
	}

	// calculate and update Realized PnL
	if p.Side == LONG { // calculate long side
		// long_pnl = (price - EntryPrice) * size
		pnl = priceD.Sub(p.EntryPrice).Mul(sizeD)
	} else { // calculate short side
		// short_pnl = (EntryPrice - price) * size
		pnl = p.EntryPrice.Sub(priceD).Mul(sizeD)
	}
	p.RealizedPnL = p.RealizedPnL.Add(pnl)

	// reduce position size
	p.Size = p.Size.Sub(sizeD)

	if p.Size.IsZero() { // is size is zero -> close position
		p.Status = PositionClosed
		p.InitialMargin = decimal.Zero
		p.MaintenanceMargin = decimal.Zero
	} else { // update maintenance margin
		positionValue := p.EntryPrice.Mul(p.Size)
		p.InitialMargin = positionValue.Div(decimal.NewFromUint64(uint64(p.Leverage)))
		p.MaintenanceMargin = p.calculateMaintenanceMargin(positionValue)
	}

	// update cache
	p.sizeFloat = p.Size.InexactFloat64()
	// update time
	p.UpdateTime = time.Now()

	return pnl, nil
}

// Close position（全部平倉）
func (p *Position) Close(price float64) (decimal.Decimal, error) {
	// reduce all size left.
	return p.Reduce(price, p.sizeFloat)
}

// UpdateMarkPrice (更新標記價格)
func (p *Position) UpdateMarkPrice(markPrice float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	markPriceD := decimal.NewFromFloat(markPrice)
	p.MarkPrice = markPriceD

	if p.Size.IsZero() {
		p.UnrealizedPnL = decimal.Zero
	}

	// calculate unrealized PnL
	if p.Side == LONG { // long side
		// formula = (markPrice - entryPrice) * size
		p.UnrealizedPnL = markPriceD.Sub(p.EntryPrice).Mul(p.Size)
	} else { // short side
		// formula = (entryPrice - markPrice) * size
		p.UnrealizedPnL = p.EntryPrice.Sub(markPriceD).Mul(p.Size)
	}
}

// CalculateLiquidationPrice (強平價格)
func (p *Position) CalculateLiquidationPrice() decimal.Decimal {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Size.IsZero() {
		return decimal.Zero
	}

	// Liquidation Price Formula:
	// (LONG) :  LiquidationPrice = EntryPrice - (InitialMargin - MaintenanceMargin) / Size
	// (SHORT):  LiquidationPrice = EntryPrice + (InitialMargin - MaintenanceMargin) / Size

	// 理解公式：假如初始押金我投入 10000 USDT，我開倉數量為 10 顆 FZO 幣，不考慮維持保證金情況下，我每一顆 FZO 的押金是 10000/10 = 1000
	// 		   相當於我每一個 FZO 幣最多虧損 1000 元就應概要被強制平倉．
	//         假如我的開倉價為 100000 USDT：
	//        		多頭強平價就是 100000-1000 = 99000  USDT
	//        		空頭強平價就是 100000+1000 = 110000 USDT

	marginBuffer := p.InitialMargin.Sub(p.MaintenanceMargin) // 保證金緩衝額 = 初始放入的押金 - 滑價保險額度
	priceBuffer := marginBuffer.Div(p.Size)                  // 價格緩衝額   = 保證金緩衝額 / 倉位數量

	if p.Side == LONG { // LONG side
		p.LiquidationPrice = p.EntryPrice.Sub(priceBuffer)
	} else { // SHORT side
		p.LiquidationPrice = p.EntryPrice.Add(priceBuffer)
	}

	return p.LiquidationPrice
}

// GetMarginRatio (保證金率)
func (p *Position) GetMarginRatio() decimal.Decimal {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.Size.IsZero() || p.MarkPrice.IsZero() {
		return decimal.NewFromInt(100) // safe
	}

	// MarginRatio Formula:
	// MarginRatio = (MarginAccount Equity Value / Position Value) * 100%

	positionValue := p.MarkPrice.Mul(p.Size)
	if positionValue.IsZero() {
		return decimal.NewFromInt(100)
	}
	// cross: TODO
	// Isolated: (InitialMargin + UnrealizedPnL) / (MarkPrice * Size)
	accountEquity := decimal.Zero
	switch p.MarginMode {
	case CROSS:
		panic("not implemented cross mode yet")
	case ISOLATED:
		accountEquity = p.InitialMargin.Add(p.UnrealizedPnL)
	}

	return accountEquity.Div(positionValue).Mul(decimal.NewFromInt(100))
}

// IsLiquidatable (可清算)
func (p *Position) IsLiquidatable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.Status != PositionNormal || p.Size.IsZero() {
		return false
	}

	marginRatio := p.GetMarginRatio()

	// marginRatio:      (MarginAccountEquity / PositionValue) * 100%
	// maintenanceRatio: (MaintenanceMargin / PositionValue) * 100%
	positionValue := p.MarkPrice.Mul(p.Size)
	maintenanceRatio := p.MaintenanceMargin.Div(positionValue).Mul(decimal.NewFromInt(100))
	// 如果沒有 MaintenanceMargin，可以理解為 marginRatio 降低到 0 既為可被清算
	return marginRatio.LessThanOrEqual(maintenanceRatio)
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
		"size":              p.Size.String(),
		"entry_price":       p.EntryPrice.String(),
		"mark_price":        p.MarkPrice.String(),
		"liquidation_price": p.LiquidationPrice.String(),
		"leverage":          p.Leverage,
		"margin_mode":       p.MarginMode,
		"unrealized_pnl":    p.UnrealizedPnL.String(),
		"realized_pnl":      p.RealizedPnL.String(),
		"margin_ratio":      p.GetMarginRatio().String() + "%",
		"is_liquidatable":   p.IsLiquidatable(),
	}
}

// --------------------------------------------------------------------------------------------
// private func
// --------------------------------------------------------------------------------------------

// calculateMaintenanceMargin calculate Maintenance Margin value
func (p *Position) calculateMaintenanceMargin(positionValue decimal.Decimal) decimal.Decimal {
	positionValueF := positionValue.InexactFloat64()
	maintenanceRate := decimal.Zero
	for _, t := range DefaultMarginTiers {
		if positionValueF >= t.MinValue && positionValueF <= t.MaxValue {
			maintenanceRate = t.MaintenanceRate
			break
		}
	}
	return positionValue.Mul(maintenanceRate)
}
