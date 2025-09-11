package position

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
