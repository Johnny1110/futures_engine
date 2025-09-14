package position

import (
	_ "math"
	"testing"
	_ "time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers
func createTestPosition(userID, symbol string) *Position {
	return NewPosition(userID, symbol, ISOLATED, nil)
}

func createCustomPrecisionPosition(sizePrecision, pricePrecision int8) *Position {
	precision := &PrecisionSetting{
		SizePrecision:  sizePrecision,
		PricePrecision: pricePrecision,
	}
	return NewPosition("test_user", "BTCUSDT", ISOLATED, precision)
}

// Test Position Creation
func TestNewPosition(t *testing.T) {
	t.Run("DefaultPrecision", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		assert.Equal(t, "user1", pos.UserID)
		assert.Equal(t, "BTCUSDT", pos.Symbol)
		assert.Equal(t, ISOLATED, pos.MarginMode)
		assert.Equal(t, PositionNormal, pos.Status)
		assert.Equal(t, 0.0, pos.Size)
		assert.Equal(t, int8(2), pos.pricePrecision)
		assert.Equal(t, int8(8), pos.sizePrecision)
	})

	t.Run("CustomPrecision", func(t *testing.T) {
		pos := createCustomPrecisionPosition(4, 1)

		assert.Equal(t, int8(4), pos.sizePrecision)
		assert.Equal(t, int8(1), pos.pricePrecision)
		assert.Equal(t, 0.0001, pos.ZeroSize())
		assert.Equal(t, 0.1, pos.ZeroPrice())
	})
}

// Test Position Opening
func TestPositionOpen(t *testing.T) {
	t.Run("LongPosition", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		assert.Equal(t, LONG, pos.Side)
		assert.Equal(t, 50000.0, pos.EntryPrice)
		assert.Equal(t, 50000.0, pos.MarkPrice)
		assert.Equal(t, 1.0, pos.Size)
		assert.Equal(t, int16(10), pos.Leverage)
		assert.Equal(t, 50000.0, pos.PositionValue)
		assert.Equal(t, 5000.0, pos.InitialMargin) // 50000 / 10
		assert.True(t, pos.MaintenanceMargin > 0)
		assert.True(t, pos.LiquidationPrice > 0)
	})

	t.Run("ShortPosition", func(t *testing.T) {
		pos := createTestPosition("user1", "ETHUSDT")

		err := pos.Open(SHORT, 3000, 5.0, 20)
		require.NoError(t, err)

		assert.Equal(t, SHORT, pos.Side)
		assert.Equal(t, 3000.0, pos.EntryPrice)
		assert.Equal(t, 5.0, pos.Size)
		assert.Equal(t, int16(20), pos.Leverage)
		assert.Equal(t, 15000.0, pos.PositionValue)
		assert.Equal(t, 750.0, pos.InitialMargin) // 15000 / 20
	})

	t.Run("AlreadyOpenedError", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		// Try to open again
		err = pos.Open(SHORT, 51000, 0.5, 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "position already exists")
	})
}

// Test Position Adding (加倉)
func TestPositionAdd(t *testing.T) {
	t.Run("AddToLongPosition", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		// Open initial position
		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		// Add position
		err = pos.Add(51000, 0.5)
		require.NoError(t, err)

		// New average price = (50000*1 + 51000*0.5) / 1.5 = 50333.33
		expectedAvgPrice := (50000*1 + 51000*0.5) / 1.5
		assert.InDelta(t, expectedAvgPrice, pos.EntryPrice, 0.01)
		assert.Equal(t, 1.5, pos.Size)

		// Updated margin should reflect new size
		expectedInitialMargin := pos.EntryPrice * 1.5 / 10
		assert.InDelta(t, expectedInitialMargin, pos.InitialMargin, 0.01)
	})

	t.Run("AddToShortPosition", func(t *testing.T) {
		pos := createTestPosition("user1", "ETHUSDT")

		err := pos.Open(SHORT, 3000, 2.0, 5)
		require.NoError(t, err)

		err = pos.Add(2950, 1.0)
		require.NoError(t, err)

		// New average price = (3000*2 + 2950*1) / 3 = 2983.33
		expectedAvgPrice := (3000*2 + 2950*1) / 3
		assert.InDelta(t, expectedAvgPrice, pos.EntryPrice, 1.0)
		assert.Equal(t, 3.0, pos.Size)
	})

	t.Run("AddToClosedPositionError", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")
		pos.Status = PositionClosed

		err := pos.Add(50000, 1.0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "position status is not normal")
	})
}

// Test Position Reducing (減倉)
func TestPositionReduce(t *testing.T) {
	t.Run("PartialReduceLongProfit", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 2.0, 10)
		require.NoError(t, err)

		// Reduce at higher price (profit)
		pnl, err := pos.Reduce(52000, 1.0)
		require.NoError(t, err)

		expectedPnL := (52000 - 50000) * 1.0 // 2000
		assert.Equal(t, expectedPnL, pnl)
		assert.Equal(t, expectedPnL, pos.RealizedPnL)
		assert.Equal(t, 1.0, pos.Size) // Remaining size
		assert.Equal(t, PositionNormal, pos.Status)
	})

	t.Run("PartialReduceShortLoss", func(t *testing.T) {
		pos := createTestPosition("user1", "ETHUSDT")

		err := pos.Open(SHORT, 3000, 2.0, 5)
		require.NoError(t, err)

		// Reduce at higher price (loss for short)
		pnl, err := pos.Reduce(3100, 0.5)
		require.NoError(t, err)

		expectedPnL := (3000 - 3100) * 0.5 // -50
		assert.Equal(t, expectedPnL, pnl)
		assert.Equal(t, expectedPnL, pos.RealizedPnL)
		assert.Equal(t, 1.5, pos.Size)
	})

	t.Run("FullReduceClosesPosition", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		pnl, err := pos.Reduce(51000, 1.0)
		require.NoError(t, err)

		assert.Equal(t, 1000.0, pnl)
		assert.Equal(t, PositionClosed, pos.Status)
		assert.Equal(t, 0.0, pos.Size)
		assert.Equal(t, 0.0, pos.PositionValue)
		assert.Equal(t, 0.0, pos.InitialMargin)
	})

	t.Run("ReduceExceedsSizeError", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		_, err = pos.Reduce(51000, 2.0) // Reduce more than size
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reduce size exceeds position size")
	})
}

// Test Position Closing
func TestPositionClose(t *testing.T) {
	t.Run("CloseLongPosition", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		pnl, err := pos.Close(52000)
		require.NoError(t, err)

		assert.Equal(t, 2000.0, pnl)
		assert.Equal(t, PositionClosed, pos.Status)
		assert.Equal(t, 0.0, pos.Size)
	})

	t.Run("CloseShortPosition", func(t *testing.T) {
		pos := createTestPosition("user1", "ETHUSDT")

		err := pos.Open(SHORT, 3000, 2.0, 5)
		require.NoError(t, err)

		pnl, err := pos.Close(2950)
		require.NoError(t, err)

		expectedPnL := (3000 - 2950) * 2.0 // 100
		assert.Equal(t, expectedPnL, pnl)
		assert.Equal(t, PositionClosed, pos.Status)
	})
}

// Test Mark Price Updates
func TestUpdateMarkPrice(t *testing.T) {
	t.Run("LongPositionPriceIncrease", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		pos.UpdateMarkPrice(51000)

		assert.Equal(t, 51000.0, pos.MarkPrice)
		assert.Equal(t, 51000.0, pos.PositionValue)
		assert.Equal(t, 1000.0, pos.UnrealizedPnL) // (51000 - 50000) * 1
	})

	t.Run("ShortPositionPriceDecrease", func(t *testing.T) {
		pos := createTestPosition("user1", "ETHUSDT")

		err := pos.Open(SHORT, 3000, 2.0, 5)
		require.NoError(t, err)

		pos.UpdateMarkPrice(2900)

		assert.Equal(t, 2900.0, pos.MarkPrice)
		assert.Equal(t, 5800.0, pos.PositionValue) // 2900 * 2
		assert.Equal(t, 200.0, pos.UnrealizedPnL)  // (3000 - 2900) * 2
	})

	t.Run("ZeroSizeNoUnrealizedPnL", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		pos.UpdateMarkPrice(50000)

		assert.Equal(t, 0.0, pos.PositionValue)
		assert.Equal(t, 0.0, pos.UnrealizedPnL)
	})
}

// Test Margin Calculations
func TestMarginCalculations(t *testing.T) {
	t.Run("GetMarginRatio", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		// Initial margin ratio should be 10% (no unrealized PnL)
		marginRatio := pos.GetMarginRatio()
		assert.InDelta(t, 10.0, marginRatio, 1.0)

		// Price goes up, margin ratio should increase
		pos.UpdateMarkPrice(51000)
		marginRatio = pos.GetMarginRatio()
		assert.True(t, marginRatio > 10.0)

		// Price goes down significantly, margin ratio should decrease
		pos.UpdateMarkPrice(48000)
		marginRatio = pos.GetMarginRatio()
		assert.True(t, marginRatio < 10.0)
	})

	t.Run("MarginRatioZeroPosition", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		marginRatio := pos.GetMarginRatio()
		assert.Equal(t, 100.0, marginRatio) // Safe value for zero position
	})
}

// Test Liquidation Logic
func Test_Liquidation(t *testing.T) {
	t.Run("HighLeverageLongLiquidation", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 100) // High leverage
		require.NoError(t, err)

		// Price should be close to liquidation price
		liquidationPrice := pos.LiquidationPrice
		assert.True(t, liquidationPrice > 0)
		assert.True(t, liquidationPrice < pos.EntryPrice)

		// Test near liquidation
		pos.UpdateMarkPrice(liquidationPrice + 10) // Just above liquidation
		assert.False(t, pos.IsLiquidatable())

		// Test at liquidation
		pos.UpdateMarkPrice(liquidationPrice - 10) // Below liquidation
		assert.True(t, pos.IsLiquidatable())
	})

	t.Run("HighLeverageShortLiquidation", func(t *testing.T) {
		pos := createTestPosition("user1", "ETHUSDT")

		err := pos.Open(SHORT, 3000, 1.0, 50)
		require.NoError(t, err)

		liquidationPrice := pos.LiquidationPrice
		assert.True(t, liquidationPrice > pos.EntryPrice) // Short liquidation price above entry

		// Test liquidation
		pos.UpdateMarkPrice(liquidationPrice + 10)
		assert.True(t, pos.IsLiquidatable())
	})

	t.Run("ClosedPositionNotLiquidatable", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")
		pos.Status = PositionClosed

		assert.False(t, pos.IsLiquidatable())
	})
}

// Test ROI Calculation
func TestROI(t *testing.T) {
	t.Run("PositiveROI", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		pos.UpdateMarkPrice(55000)

		roi := pos.GetRoi()
		expectedROI := 5000.0 / 5000.0 // UnrealizedPnL / InitialMargin
		assert.Equal(t, expectedROI, roi)
	})

	t.Run("NegativeROI", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		pos.UpdateMarkPrice(45000)

		roi := pos.GetRoi()
		expectedROI := -5000.0 / 5000.0 // -1.0
		assert.Equal(t, expectedROI, roi)
	})
}

// Test Display Info
func TestGetDisplayInfo(t *testing.T) {
	pos := createTestPosition("user1", "BTCUSDT")

	err := pos.Open(LONG, 50000, 1.0, 10)
	require.NoError(t, err)

	info := pos.GetDisplayInfo()

	assert.Equal(t, "user1", info["user_id"])
	assert.Equal(t, "BTCUSDT", info["symbol"])
	assert.Equal(t, "long", info["side"])
	assert.Equal(t, 1.0, info["size"])
	assert.Equal(t, 50000.0, info["entry_price"])
	assert.Equal(t, int16(10), info["leverage"])
	assert.Equal(t, ISOLATED, info["margin_mode"])
}

// Test Precision Functions
func TestPrecisionFunctions(t *testing.T) {
	t.Run("DefaultPrecision", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		assert.Equal(t, 0.01, pos.ZeroPrice())      // 10^-2
		assert.Equal(t, 0.00000001, pos.ZeroSize()) // 10^-8
	})

	t.Run("CustomPrecision", func(t *testing.T) {
		pos := createCustomPrecisionPosition(3, 1)

		assert.Equal(t, 0.1, pos.ZeroPrice())  // 10^-1
		assert.Equal(t, 0.001, pos.ZeroSize()) // 10^-3
	})
}

// Test Thread Safety
func TestThreadSafety(t *testing.T) {
	pos := createTestPosition("user1", "BTCUSDT")

	err := pos.Open(LONG, 50000, 1.0, 10)
	require.NoError(t, err)

	// Simulate concurrent operations
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			pos.UpdateMarkPrice(50000 + float64(i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			pos.GetMarginRatio()
			pos.IsLiquidatable()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Should not panic and position should be valid
	assert.True(t, pos.MarkPrice >= 50000)
	assert.Equal(t, 1.0, pos.Size)
}

// Test Edge Cases
func TestEdgeCases(t *testing.T) {
	t.Run("VerySmallPosition", func(t *testing.T) {
		pos := createCustomPrecisionPosition(8, 2)

		err := pos.Open(LONG, 50000, 0.00000001, 10) // Minimum size
		require.NoError(t, err)

		assert.True(t, pos.Size > 0)
		assert.True(t, pos.InitialMargin > 0)
	})

	t.Run("VeryHighLeverage", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 125) // Max leverage
		require.NoError(t, err)

		assert.Equal(t, int16(125), pos.Leverage)
		assert.True(t, pos.LiquidationPrice > 0)
		assert.True(t, pos.LiquidationPrice < pos.EntryPrice)
	})

	t.Run("ZeroPriceHandling", func(t *testing.T) {
		pos := createTestPosition("user1", "BTCUSDT")

		err := pos.Open(LONG, 50000, 1.0, 10)
		require.NoError(t, err)

		pos.UpdateMarkPrice(0) // Should not crash
		assert.Equal(t, 0.0, pos.MarkPrice)
	})
}
