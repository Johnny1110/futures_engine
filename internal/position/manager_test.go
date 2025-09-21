package position

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var symbols = []string{"BTCUSDT", "ETHUSDT"}

// TestBasicPositionLifecycle basic lifecycle
func TestBasicPositionLifecycle(t *testing.T) {
	pm := NewPositionManager(symbols)
	userID := "user123"
	symbol := "BTCUSDT"

	// 1. 開多倉
	fmt.Println("=== 開多倉 ===")
	position, err := pm.OpenPosition(ISOLATED, userID, symbol, LONG, 50000, 1.0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, position)

	fmt.Printf("開倉成功: %+v\n", position.GetDisplayInfo())
	fmt.Println("開倉時的收益率:", position.GetRoi())

	// 驗證初始值
	assert.Equal(t, 1.0, position.Size)
	assert.Equal(t, 50000.0, position.EntryPrice)
	assert.Equal(t, int16(10), position.Leverage)

	// 初始保證金應該是 50000 / 10 = 5000
	expectedMargin := 5000.0
	fmt.Println("position.InitialMargin:", position.InitialMargin)
	assert.Equal(t, expectedMargin, position.InitialMargin)

	fmt.Println("MaintenanceMargin:", position.MaintenanceMargin)

	assert.True(t, position.MaintenanceMargin > 0)
	assert.InDelta(t, 10.0, position.GetMarginRatio(), 1.0)

	// 2. 更新標記價格，測試未實現盈虧
	fmt.Println("\n=== 價格上漲到 51000 ===")
	position.UpdateMarkPrice(51000)

	fmt.Println("when price -> 51000: MarginRatio:", position.GetMarginRatio())
	assert.True(t, position.GetMarginRatio() > 10.0)

	// 多倉，價格上漲，應該盈利 1000
	expectedPnL := 1000.0
	assert.Equal(t, expectedPnL, position.UnrealizedPnL)
	fmt.Printf("未實現盈虧: %f\n", position.UnrealizedPnL)
	fmt.Println("價格到 51000 的收益率:", position.GetRoi())

	// 3. 加倉
	fmt.Println("\n=== 加倉 0.5 BTC @ 51000 ===")
	err = position.Add(51000, 0.5)
	assert.NoError(t, err)

	// 新均價 = (50000*1 + 51000*0.5) / 1.5 = 50333.33
	expectedEntryPrice := 50333.333333333336
	assert.InDelta(t, expectedEntryPrice, position.EntryPrice, 0.01)
	assert.Equal(t, 1.5, position.Size)

	fmt.Printf("加倉後倉位: %+v\n", position.GetDisplayInfo())

	// 4. 部分平倉
	fmt.Println("\n=== 部分平倉 0.5 BTC @ 52000 ===")
	pnl, err := position.Reduce(52000, 0.5)
	assert.NoError(t, err)

	// 平倉盈虧 = (52000 - 50333.33) * 0.5 = 833.33
	expectedReducePnL := 833.335
	assert.InDelta(t, expectedReducePnL, pnl, 1.0)

	fmt.Printf("部分平倉已實現盈虧: %f\n", pnl)
	fmt.Printf("剩餘倉位: %+v\n", position.GetDisplayInfo())

	// 5. 全部平倉
	fmt.Println("\n=== 全部平倉 @ 53000 ===")
	finalPnL, err := position.Close(53000)
	assert.NoError(t, err)

	// 最終平倉盈虧 = (53000 - 50333.33) * 1.0 = 2666.67
	expectedFinalPnL := 2666.67
	assert.InDelta(t, expectedFinalPnL, finalPnL, 1.0)

	fmt.Printf("最終平倉盈虧: %f\n", finalPnL)
	fmt.Printf("總已實現盈虧: %f\n", position.RealizedPnL)

	// 驗證倉位已關閉
	assert.Equal(t, PositionClosed, position.Status)
	assert.True(t, position.Size <= position.ZeroSize())
}

// TestShortPosition 測試空倉
func TestShortPosition(t *testing.T) {
	pm := NewPositionManager(symbols)
	userID := "user456"
	symbol := "ETHUSDT"

	// 1. 開空倉
	fmt.Println("=== 開空倉 ===")
	position, err := pm.OpenPosition(ISOLATED, userID, symbol, SHORT, 3000, 10, 20)
	assert.NoError(t, err)

	fmt.Printf("開空倉: %+v\n", position.GetDisplayInfo())

	// 2. 價格下跌（空倉盈利）
	fmt.Println("\n=== 價格下跌到 2900 ===")
	position.UpdateMarkPrice(2900)

	// 空倉，價格下跌，應該盈利 (3000-2900)*10 = 1000
	expectedPnL := 1000.0
	assert.Equal(t, expectedPnL, position.UnrealizedPnL)
	fmt.Printf("空倉盈利: %f\n", position.UnrealizedPnL)

	// 3. 價格上漲（空倉虧損）
	fmt.Println("\n=== 價格上漲到 3100 ===")
	position.UpdateMarkPrice(3100)

	// 空倉，價格上漲，應該虧損 (3000-3100)*10 = -1000
	expectedLoss := -1000.0
	assert.Equal(t, expectedLoss, position.UnrealizedPnL)
	fmt.Printf("空倉虧損: %f\n", position.UnrealizedPnL)
}

// TestLiquidation 測試強平
func TestLiquidation(t *testing.T) {
	position := NewPosition("user789", "BTCUSDT", ISOLATED, nil)

	// 開一個高槓桿多倉
	err := position.Open(LONG, 50000, 1, 100) // 100倍槓桿
	assert.NoError(t, err)

	fmt.Println("=== 測試強平 ===")
	fmt.Printf("初始倉位: %+v\n", position.GetDisplayInfo())

	// 計算強平價格
	liquidationPrice := position.LiquidationPrice
	fmt.Printf("強平價格: %f\n", liquidationPrice)

	// 100倍槓桿，初始保證金 1%，維持保證金 0.5%
	// 強平價格應該約等於 50000 * (1 - 0.01 + 0.004) = 49700
	expectedLiqPrice := 49700.0
	assert.InDelta(t, expectedLiqPrice, liquidationPrice, 100.0)

	// 測試接近強平
	fmt.Println("\n=== 價格接近強平價 ===")
	position.UpdateMarkPrice(49710)
	fmt.Println("接近強平時，倉位資料：", position.GetDisplayInfo())
	fmt.Println("接近強平時，倉位價值：", position.PositionValue)
	fmt.Println("初始保證金：", position.InitialMargin, "未實現損益:", position.UnrealizedPnL)
	// MarginRatio = (MarginAccount Equity Value / Position Value) * 100%
	//  (初始保證金 + 未實現損益) /49710 * 100% -> (500 - -290) / 49710
	marginRatio := position.GetMarginRatio()
	fmt.Printf("保證金率: %f%%\n", marginRatio)
	fmt.Printf("是否可強平: %v\n", position.IsLiquidatable())

	// 測試觸發強平
	fmt.Println("\n=== 價格跌破強平價 ===")
	position.UpdateMarkPrice(49700)
	marginRatio = position.GetMarginRatio()
	fmt.Printf("保證金率: %f%%\n", marginRatio)
	fmt.Printf("維持保證金: %f\n", position.MaintenanceMargin)
	fmt.Printf("是否可強平: %v\n", position.IsLiquidatable())
	fmt.Printf("當前倉位狀態: %s\n", position.Status)
	assert.True(t, position.IsLiquidatable())
}

// TestPositionModes 測試倉位模式
func TestPositionModes(t *testing.T) {
	pm := NewPositionManager(symbols)
	userID := "user_mode_test"
	symbol := "BTCUSDT"

	// 1. 單向持倉模式（默認）
	fmt.Println("=== 單向持倉模式 ===")

	// 開多倉
	pos1, err := pm.OpenPosition(ISOLATED, userID, symbol, LONG, 50000, 1, 10)
	assert.NoError(t, err)

	// 嘗試開空倉（單向模式下應該失敗或關閉多倉）
	// 這裡的邏輯取決於業務需求
	fmt.Printf("單向模式多倉: %+v\n", pos1.GetDisplayInfo())

	// 2. 切換到雙向持倉模式
	fmt.Println("\n=== 嘗試切換到雙向持倉模式 ===")
	err = pm.SetPositionMode(userID, HedgeMode)
	assert.Error(t, err) // 應該失敗，因為有未平倉位
	fmt.Printf("切換失敗（預期）: %v\n", err)

	// 平掉所有倉位
	_, _, err = pm.ClosePosition(userID, symbol, LONG, 50000)
	assert.NoError(t, err)

	// 現在可以切換
	err = pm.SetPositionMode(userID, HedgeMode)
	assert.NoError(t, err)
	fmt.Println("成功切換到雙向持倉模式")

	// 3. 雙向持倉模式下同時持有多空倉位
	fmt.Println("\n=== 雙向持倉模式 ===")

	// 開多倉
	longPos, err := pm.OpenPosition(ISOLATED, userID, symbol, LONG, 50000, 1, 10)
	assert.NoError(t, err)

	// 開空倉
	shortPos, err := pm.OpenPosition(ISOLATED, userID, symbol, SHORT, 50100, 0.5, 10)
	assert.NoError(t, err)

	fmt.Printf("多倉: %+v\n", longPos.GetDisplayInfo())
	fmt.Printf("空倉: %+v\n", shortPos.GetDisplayInfo())

	// 驗證是兩個獨立的倉位
	assert.NotEqual(t, longPos.ID, shortPos.ID)
	assert.Equal(t, LONG, longPos.Side)
	assert.Equal(t, SHORT, shortPos.Side)
}

// TestBatchLiquidationCheck 測試批量強平檢查
func TestBatchLiquidationCheck(t *testing.T) {
	pm := NewPositionManager(symbols)

	// 創建多個用戶的倉位
	fmt.Println("=== 創建多個測試倉位 ===")

	// 用戶1：安全倉位（低槓桿）
	position1, err := pm.OpenPosition(ISOLATED, "user1", "BTCUSDT", LONG, 50000, 1, 5)
	assert.Nil(t, err)
	fmt.Println("user1 開倉: ", position1.GetDisplayInfo())

	// 用戶2：高風險倉位（高槓桿）
	position2, err := pm.OpenPosition(ISOLATED, "user2", "BTCUSDT", LONG, 50000, 1, 100)
	assert.Nil(t, err)
	fmt.Println("user2 開倉: ", position2.GetDisplayInfo())

	// 用戶3：空倉高風險
	position3, err := pm.OpenPosition(ISOLATED, "user3", "BTCUSDT", SHORT, 50000, 1, 75)
	assert.Nil(t, err)
	fmt.Println("user3 開倉: ", position3.GetDisplayInfo())

	// 模擬市場價格變動
	prices := map[string]float64{
		"BTCUSDT": 49500, // 價格下跌
	}

	// 獲取所有可強平倉位
	liquidatable := []*Position{}
	for symbol, price := range prices {
		ls, _ := pm.UpdateMarkPrices(symbol, price)
		liquidatable = append(liquidatable, ls...)
	}

	fmt.Println("=== 假個下跌至 49500 時 ===")
	fmt.Println("user1 倉位: ", position1.GetDisplayInfo())
	fmt.Println("user2 倉位: ", position2.GetDisplayInfo())
	fmt.Println("user3 倉位: ", position3.GetDisplayInfo())

	fmt.Printf("\n發現 %d 個可強平倉位\n", len(liquidatable))
	for _, pos := range liquidatable {
		fmt.Printf("用戶 %s 的 %s %s 倉位需要強平\n",
			pos.UserID, pos.Symbol, pos.Side.String())
		fmt.Printf("  - 保證金率: %f%%\n", pos.GetMarginRatio())
		fmt.Printf("  - 未實現虧損: %f\n", pos.UnrealizedPnL)
	}

	// 至少用戶2的高槓桿多倉應該被強平
	assert.GreaterOrEqual(t, len(liquidatable), 1)
}

// TestPrecisionAndRounding 測試精度和四捨五入
func TestPrecisionAndRounding(t *testing.T) {
	position := NewPosition("precision_test", "BTCUSDT", ISOLATED, nil)

	fmt.Println("=== 測試精度處理 ===")

	// 使用會產生無限小數的價格
	err := position.Open(LONG, 33333.33, 0.123456789, 10)
	assert.NoError(t, err)

	// 多次加倉，測試精度累積
	err = position.Add(33334.44, 0.111111111)
	assert.NoError(t, err)

	err = position.Add(33335.55, 0.099999999)
	assert.NoError(t, err)

	fmt.Printf("複雜計算後的倉位:\n")
	fmt.Printf("  - 倉位大小: %f\n", position.Size)
	fmt.Printf("  - 開倉均價: %f\n", position.EntryPrice)

	// 測試盈虧計算精度
	position.UpdateMarkPrice(33340.00)
	fmt.Printf("  - 未實現盈虧: %f\n", position.UnrealizedPnL)

	// 部分平倉測試精度
	pnl, err := position.Reduce(33350.00, 0.123456789)
	assert.NoError(t, err)
	fmt.Printf("  - 平倉盈虧: %f\n", pnl)

	// 驗證所有計算都保持精確
	assert.True(t, position.Size > position.ZeroSize())
	assert.True(t, position.RealizedPnL > 0)
}

// BenchmarkPositionOperations 性能測試
func BenchmarkPositionOperations(b *testing.B) {
	pm := NewPositionManager(symbols)

	b.Run("OpenPosition", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			userID := fmt.Sprintf("user_%d", i)
			pm.OpenPosition(ISOLATED, userID, "BTCUSDT", LONG, 50000, 1, 10)
		}
	})

	b.Run("UpdateMarkPrice", func(b *testing.B) {
		// 準備測試數據
		for i := 0; i < 1000; i++ {
			userID := fmt.Sprintf("bench_user_%d", i)
			pm.OpenPosition(ISOLATED, userID, "BTCUSDT", LONG, 50000, 1, 10)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pm.UpdateMarkPrices("BTCUSDT", 51000)
		}
	})
}

// Example 使用範例
func ExamplePositionManager() {
	pm := NewPositionManager(symbols)

	// 開倉
	position, _ := pm.OpenPosition(ISOLATED, "alice", "BTCUSDT", LONG, 50000, 1, 10)
	fmt.Printf("Alice 開多倉 1 BTC @ $50,000，10倍槓桿\n")

	// 市場上漲
	position.UpdateMarkPrice(51000)
	fmt.Printf("價格漲到 $51,000，未實現盈利: %f\n", position.UnrealizedPnL)

	// 加倉
	position.Add(51000, 0.5)
	fmt.Printf("加倉 0.5 BTC @ $51,000\n")

	// 部分平倉
	pnl, _ := position.Reduce(52000, 0.5)
	fmt.Printf("平倉 0.5 BTC @ $52,000，已實現盈利: %f\n", pnl)

	// 檢查強平風險
	position.UpdateMarkPrice(46000)
	if position.IsLiquidatable() {
		fmt.Printf("警告：倉位面臨強平風險！\n")
	}
}
