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
	InitialMargin     decimal.Decimal   `json:"initial_margin"`     // 初始保證金
	MaintenanceMargin decimal.Decimal   `json:"maintenance_margin"` // 維持保證金
	Leverage          uint              `json:"leverage"`
	MarginMode        common.MarginMode `json:"margin_mode"`

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
func NewPosition(userID, symbol string) *Position {
	return &Position{
		ID:              common.GeneratePositionID(),
		UserID:          userID,
		Symbol:          symbol,
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

// Open Position
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

// --------------------------------------------------------------------------------------------
// private func
// --------------------------------------------------------------------------------------------

// calculateMaintenanceMargin calculate Maintenance Margin value
func (p *Position) calculateMaintenanceMargin(positionValue decimal.Decimal) decimal.Decimal {
	// TODO
	return decimal.Zero
}
