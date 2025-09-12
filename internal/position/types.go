package position

import "github.com/shopspring/decimal"

// PositionSide LONG or SHORT
type PositionSide int

const (
	LONG  PositionSide = 1
	SHORT PositionSide = -1
)

func (ps PositionSide) String() string {
	switch ps {
	case LONG:
		return "long"
	case SHORT:
		return "short"
	default:
		return "unknown"
	}
}

// ========================================================

// PositionMode (持倉模式) OneWayMode HedgeMode
type PositionMode int

const (
	OneWayMode PositionMode = iota // 單向持倉模式
	HedgeMode                      // 雙向持倉模式
)

func (pm PositionMode) String() string {
	switch pm {
	case OneWayMode:
		return "one-way"
	case HedgeMode:
		return "hedge"
	default:
		return "unknown"
	}
}

// ========================================================

// PositionStatus Normal Liquidating Closed
type PositionStatus int

const (
	PositionNormal      PositionStatus = iota // 正常
	PositionLiquidating                       // 強平中
	PositionClosed                            // 已平倉
)

func (ps PositionStatus) String() string {
	switch ps {
	case PositionNormal:
		return "normal"
	case PositionLiquidating:
		return "liquidating"
	case PositionClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// ========================================================

type MarginMode int

const (
	CROSS MarginMode = iota
	ISOLATED
)

func (mode MarginMode) String() string {
	switch mode {
	case CROSS:
		return "CROSS"
	case ISOLATED:
		return "ISOLATED"
	default:
		return "unknown"
	}
}

// ========================================================

// MarginTier for calculate MaintenanceMargin
type MarginTier struct {
	MinValue        float64         // 最小倉位價值
	MaxValue        float64         // 最大倉位價值
	MaintenanceRate decimal.Decimal // 維持保證金率
	MaxLeverage     uint            // 最大槓桿
}
