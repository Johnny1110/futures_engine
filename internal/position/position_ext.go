package position

import (
	"fmt"
	"sync"
)

// atomic position slice ==================================================================

type AtomicPositions struct {
	slice []*Position
	mutex sync.RWMutex
}

func (ap *AtomicPositions) Len() int {
	ap.mutex.RLock()
	defer ap.mutex.RUnlock()
	return len(ap.slice)
}

func (ap *AtomicPositions) Append(p *Position) {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()
	ap.slice = append(ap.slice, p)
}

func (ap *AtomicPositions) Remove(index int) *Position {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()
	return ap.remove(index)
}

func (ap *AtomicPositions) remove(index int) *Position {
	pos := ap.slice[index]
	// swap to last one and pop last
	ap.slice[index] = ap.slice[len(ap.slice)-1]
	ap.slice = ap.slice[:len(ap.slice)-1]
	return pos
}

func (ap *AtomicPositions) UpdateMarkPrice(price float64) []*Position {
	ap.mutex.Lock()
	defer ap.mutex.Unlock()

	liquidateList := make([]*Position, 0)

	for idx, pos := range ap.slice {

		if pos.Status != PositionNormal { // if status is not normal, just skip.
			continue
		}

		pos.UpdateMarkPrice(price) // update mark price.

		// clean the position slice.
		switch pos.Status {
		case PositionClosed:
			ap.remove(idx)
			break
		case PositionLiquidating:
			liquidateList = append(liquidateList, pos)
			ap.remove(idx)
			break
		default:
			break
		}
	}

	return liquidateList
}

// symbol: userPositions ==================================================================

type SymbolPositions struct {
	container map[string]*AtomicPositions
	mu        sync.RWMutex
}

func NewSymbolPositions(symbols []string) *SymbolPositions {
	sp := &SymbolPositions{
		container: make(map[string]*AtomicPositions),
	}

	for _, symbol := range symbols {
		sp.AddSymbol(symbol)
	}

	return sp
}

func (s *SymbolPositions) AddSymbol(symbol string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.container[symbol]; !ok {
		s.container[symbol] = &AtomicPositions{
			slice: make([]*Position, 0),
		}
	}
}

func (s *SymbolPositions) RemoveSymbol(symbol string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.container[symbol]; ok {
		delete(s.container, symbol)
	}
}

func (s *SymbolPositions) AddPosition(symbol string, position *Position) error {
	if atomicPositions, ok := s.container[symbol]; ok {
		atomicPositions.Append(position)
		return nil
	} else {
		return fmt.Errorf("symbol %s not exist", symbol)
	}
}

// UpdateMarkPrice return liquidateList
func (s *SymbolPositions) UpdateMarkPrice(symbol string, price float64) ([]*Position, error) {
	if atomicPositions, ok := s.container[symbol]; ok {
		return atomicPositions.UpdateMarkPrice(price), nil
	} else {
		return nil, fmt.Errorf("symbol %s not exist", symbol)
	}
}
