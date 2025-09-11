package common

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
