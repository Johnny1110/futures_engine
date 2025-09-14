package position

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// Benchmark helpers
func setupBenchPosition() *Position {
	pos := NewPosition("bench_user", "BTCUSDT", ISOLATED, nil)
	pos.Open(LONG, 50000, 1.0, 10)
	return pos
}

func setupBenchPositions(count int) []*Position {
	positions := make([]*Position, count)
	for i := 0; i < count; i++ {
		pos := NewPosition(fmt.Sprintf("user_%d", i), "BTCUSDT", ISOLATED, nil)
		pos.Open(LONG, 50000+float64(i), 1.0, 10)
		positions[i] = pos
	}
	return positions
}

// Core Operations Benchmarks
func BenchmarkPositionOpen(b *testing.B) {
	b.Run("LONG", func(b *testing.B) {
		positions := make([]*Position, b.N)
		for i := 0; i < b.N; i++ {
			positions[i] = NewPosition(fmt.Sprintf("user_%d", i), "BTCUSDT", ISOLATED, nil)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			positions[i].Open(LONG, 50000, 1.0, 10)
		}
	})

	b.Run("SHORT", func(b *testing.B) {
		positions := make([]*Position, b.N)
		for i := 0; i < b.N; i++ {
			positions[i] = NewPosition(fmt.Sprintf("user_%d", i), "ETHUSDT", ISOLATED, nil)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			positions[i].Open(SHORT, 3000, 2.0, 20)
		}
	})

	b.Run("HighLeverage", func(b *testing.B) {
		positions := make([]*Position, b.N)
		for i := 0; i < b.N; i++ {
			positions[i] = NewPosition(fmt.Sprintf("user_%d", i), "BTCUSDT", ISOLATED, nil)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			positions[i].Open(LONG, 50000, 1.0, 125)
		}
	})
}

func BenchmarkPositionAdd(b *testing.B) {
	positions := setupBenchPositions(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		positions[i].Add(50100, 0.5)
	}
}

func BenchmarkPositionReduce(b *testing.B) {
	positions := setupBenchPositions(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		positions[i].Reduce(51000, 0.5)
	}
}

func BenchmarkPositionClose(b *testing.B) {
	positions := setupBenchPositions(b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		positions[i].Close(52000)
	}
}

// Mark Price Updates (Critical Hot Path)
func BenchmarkUpdateMarkPrice(b *testing.B) {
	b.Run("SinglePosition", func(b *testing.B) {
		pos := setupBenchPosition()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos.UpdateMarkPrice(50000 + float64(i%1000))
		}
	})

	b.Run("BatchUpdate_100", func(b *testing.B) {
		positions := setupBenchPositions(100)
		prices := make([]float64, 100)
		for i := 0; i < 100; i++ {
			prices[i] = 50000 + float64(i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j, pos := range positions {
				pos.UpdateMarkPrice(prices[j] + float64(i%100))
			}
		}
	})

	b.Run("BatchUpdate_1000", func(b *testing.B) {
		positions := setupBenchPositions(1000)
		prices := make([]float64, 1000)
		for i := 0; i < 1000; i++ {
			prices[i] = 50000 + float64(i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j, pos := range positions {
				pos.UpdateMarkPrice(prices[j] + float64(i%100))
			}
		}
	})

	b.Run("BatchUpdate_10000", func(b *testing.B) {
		positions := setupBenchPositions(10000)
		prices := make([]float64, 10000)
		for i := 0; i < 10000; i++ {
			prices[i] = 50000 + float64(i)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j, pos := range positions {
				pos.UpdateMarkPrice(prices[j] + float64(i%100))
			}
		}
	})
}

// Calculation Benchmarks
func BenchmarkGetMarginRatio(b *testing.B) {
	pos := setupBenchPosition()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos.GetMarginRatio()
	}
}

func BenchmarkIsLiquidatable(b *testing.B) {
	b.Run("SafePosition", func(b *testing.B) {
		pos := setupBenchPosition()
		pos.UpdateMarkPrice(50500) // Safe position

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos.IsLiquidatable()
		}
	})

	b.Run("RiskyPosition", func(b *testing.B) {
		pos := setupBenchPosition()
		pos.UpdateMarkPrice(49000) // Risky position

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos.IsLiquidatable()
		}
	})

	b.Run("HighLeveragePosition", func(b *testing.B) {
		pos := NewPosition("bench_user", "BTCUSDT", ISOLATED, nil)
		pos.Open(LONG, 50000, 1.0, 125) // High leverage
		pos.UpdateMarkPrice(49900)       // Near liquidation

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos.IsLiquidatable()
		}
	})
}

func BenchmarkBatchLiquidationCheck(b *testing.B) {
	b.Run("1000_Positions", func(b *testing.B) {
		positions := setupBenchPositions(1000)
		
		// Set some positions to risky prices
		for i := 0; i < len(positions); i += 10 {
			positions[i].UpdateMarkPrice(49000) // Make some risky
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			liquidatable := 0
			for _, pos := range positions {
				if pos.IsLiquidatable() {
					liquidatable++
				}
			}
		}
	})

	b.Run("10000_Positions", func(b *testing.B) {
		positions := setupBenchPositions(10000)
		
		// Set some positions to risky prices
		for i := 0; i < len(positions); i += 50 {
			positions[i].UpdateMarkPrice(49000) // Make some risky
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			liquidatable := 0
			for _, pos := range positions {
				if pos.IsLiquidatable() {
					liquidatable++
				}
			}
		}
	})
}

func BenchmarkGetRoi(b *testing.B) {
	pos := setupBenchPosition()
	pos.UpdateMarkPrice(52000) // Some profit

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos.GetRoi()
	}
}

// Precision Function Benchmarks
func BenchmarkPrecisionFunctions(b *testing.B) {
	pos := setupBenchPosition()

	b.Run("ZeroSize", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos.ZeroSize()
		}
	})

	b.Run("ZeroPrice", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos.ZeroPrice()
		}
	})
}

// Memory and GC Benchmarks
func BenchmarkPositionMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	b.Run("NewPosition", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewPosition(fmt.Sprintf("user_%d", i), "BTCUSDT", ISOLATED, nil)
		}
	})

	b.Run("OpenPosition", func(b *testing.B) {
		positions := make([]*Position, b.N)
		for i := 0; i < b.N; i++ {
			positions[i] = NewPosition(fmt.Sprintf("user_%d", i), "BTCUSDT", ISOLATED, nil)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			positions[i].Open(LONG, 50000, 1.0, 10)
		}
	})

	b.Run("UpdateMarkPrice", func(b *testing.B) {
		pos := setupBenchPosition()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos.UpdateMarkPrice(50000 + float64(i%1000))
		}
	})
}

// Concurrent Access Benchmarks
func BenchmarkConcurrentAccess(b *testing.B) {
	b.Run("ReadOperations", func(b *testing.B) {
		pos := setupBenchPosition()
		pos.UpdateMarkPrice(51000)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				pos.GetMarginRatio()
				pos.IsLiquidatable()
				pos.GetRoi()
			}
		})
	})

	b.Run("WriteOperations", func(b *testing.B) {
		pos := setupBenchPosition()

		b.RunParallel(func(pb *testing.PB) {
			prices := []float64{50000, 50500, 51000, 49500, 49000}
			i := 0
			for pb.Next() {
				pos.UpdateMarkPrice(prices[i%len(prices)])
				i++
			}
		})
	})

	b.Run("MixedOperations", func(b *testing.B) {
		pos := setupBenchPosition()

		b.RunParallel(func(pb *testing.PB) {
			prices := []float64{50000, 50500, 51000, 49500, 49000}
			i := 0
			for pb.Next() {
				if i%3 == 0 {
					pos.UpdateMarkPrice(prices[i%len(prices)])
				} else {
					pos.GetMarginRatio()
				}
				i++
			}
		})
	})
}

// Real-world Scenario Benchmarks
func BenchmarkRealWorldScenarios(b *testing.B) {
	b.Run("HighFrequencyTrading", func(b *testing.B) {
		positions := setupBenchPositions(100)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate rapid price updates
			basePrice := 50000.0
			priceChange := rand.Float64()*100 - 50 // -50 to +50
			newPrice := basePrice + priceChange

			for _, pos := range positions {
				pos.UpdateMarkPrice(newPrice)
			}

			// Check liquidations every 10 updates
			if i%10 == 0 {
				for _, pos := range positions {
					pos.IsLiquidatable()
				}
			}
		}
	})

	b.Run("MarketMaking", func(b *testing.B) {
		longPos := setupBenchPosition()
		shortPos := NewPosition("market_maker", "BTCUSDT", ISOLATED, nil)
		shortPos.Open(SHORT, 50100, 1.0, 10)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			price := 50000 + float64(i%200-100) // Price oscillates

			longPos.UpdateMarkPrice(price)
			shortPos.UpdateMarkPrice(price)

			// Calculate combined risk
			longPos.GetMarginRatio()
			shortPos.GetMarginRatio()
		}
	})

	b.Run("PortfolioRiskCalculation", func(b *testing.B) {
		// Simulate portfolio with different symbols and sizes
		btcPos := NewPosition("portfolio", "BTCUSDT", ISOLATED, nil)
		btcPos.Open(LONG, 50000, 0.5, 10)

		ethPos := NewPosition("portfolio", "ETHUSDT", ISOLATED, nil)
		ethPos.Open(SHORT, 3000, 5.0, 20)

		adaPos := NewPosition("portfolio", "ADAUSDT", ISOLATED, nil)
		adaPos.Open(LONG, 1.5, 1000.0, 5)

		positions := []*Position{btcPos, ethPos, adaPos}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Update all positions with correlated price movements
			btcPrice := 50000 + float64(i%1000-500)
			ethPrice := 3000 + btcPrice*0.06  // ETH loosely follows BTC
			adaPrice := 1.5 + btcPrice*0.00003

			btcPos.UpdateMarkPrice(btcPrice)
			ethPos.UpdateMarkPrice(ethPrice)
			adaPos.UpdateMarkPrice(adaPrice)

			// Calculate portfolio risk
			totalMarginRatio := 0.0
			liquidatableCount := 0

			for _, pos := range positions {
				totalMarginRatio += pos.GetMarginRatio()
				if pos.IsLiquidatable() {
					liquidatableCount++
				}
			}
		}
	})
}

// Stress Test Benchmarks
func BenchmarkStressTests(b *testing.B) {
	b.Run("ExtremePriceVolatility", func(b *testing.B) {
		pos := setupBenchPosition()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate extreme price swings
			multiplier := 1.0 + (rand.Float64()-0.5)*0.2 // ±20% swings
			newPrice := 50000 * multiplier

			pos.UpdateMarkPrice(newPrice)
			pos.IsLiquidatable()
		}
	})

	b.Run("MassiveBatchOperations", func(b *testing.B) {
		positions := setupBenchPositions(50000)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Update all positions
			basePrice := 50000 + float64(i%100)
			for j, pos := range positions {
				price := basePrice + float64(j%10)
				pos.UpdateMarkPrice(price)
			}
		}
	})
}

// Comparison Benchmarks (for before/after performance measurement)
func BenchmarkLegacyComparison(b *testing.B) {
	// These would be used to compare against decimal.Decimal implementation
	b.Run("Float64Operations", func(b *testing.B) {
		pos := setupBenchPosition()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate the operations that were slow with decimal
			price := 50000 + float64(i%1000)
			pos.UpdateMarkPrice(price)
			pos.GetMarginRatio()
			pos.IsLiquidatable()
		}
	})
}

// Initialize random seed for benchmarks
func init() {
	rand.Seed(time.Now().UnixNano())
}