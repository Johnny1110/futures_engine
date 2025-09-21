package position

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAtomicPositions(t *testing.T) {
	atomicPositions := &AtomicPositions{
		slice: make([]*Position, 0),
	}

	// create some testing position
	position_1 := NewPosition("user_1", "BTCUSDT", ISOLATED, nil)
	err := position_1.Open(LONG, 100000, 1, 100)
	assert.Nil(t, err)
	atomicPositions.Append(position_1)

	position_2 := NewPosition("user_2", "BTCUSDT", ISOLATED, nil)
	err = position_2.Open(LONG, 100000, 1, 50)
	assert.Nil(t, err)
	atomicPositions.Append(position_2)

	position_3 := NewPosition("user_3", "BTCUSDT", ISOLATED, nil)
	err = position_3.Open(LONG, 100000, 1, 10)
	assert.Nil(t, err)
	atomicPositions.Append(position_3)

	fmt.Println("=== all position opened ===")

	fmt.Println("pos_1", position_1.GetDisplayInfo())
	fmt.Println("pos_2", position_2.GetDisplayInfo())
	fmt.Println("pos_3", position_3.GetDisplayInfo())

	lp := atomicPositions.UpdateMarkPrice(98000)

	fmt.Println("=== after update mark price ===")

	fmt.Println("pos_1", position_1.GetDisplayInfo())
	fmt.Println("pos_2", position_2.GetDisplayInfo())
	fmt.Println("pos_3", position_3.GetDisplayInfo())

	assert.Equal(t, 2, len(lp))
	assert.Equal(t, 1, atomicPositions.Len())

}
