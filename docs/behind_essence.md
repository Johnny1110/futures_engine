# 永續合約的本質：零和遊戲

<br>

---

<br>

## 一、永續合約不涉及真實 BTC 交割

核心概念：永續合約是純粹的差價合約（CFD）

```go
現貨交易：
Alice 買入 1 BTC → Alice 真的擁有 1 BTC
Bob 賣出 1 BTC → Bob 真的交出 1 BTC

永續合約：
Alice 開多 1 BTC → Alice 持有 1 BTC 的"多頭倉位"（只是數字）
Bob 開空 1 BTC → Bob 持有 1 BTC 的"空頭倉位"（只是數字）
沒有真實的 BTC 易手！
```

<br>
<br>

## 二、平倉的真實機制

1. 部分平倉場景

當 Alice 部分平倉 0.5 BTC 時：

```go
// Alice 的平倉訂單
type ClosePositionOrder struct {
    UserID      string
    Symbol      string
    Side        string  // "SELL" (平多倉)
    Quantity    float64 // 0.5 BTC
    OrderType   string  // "LIMIT"
    Price       float64 // 50,600 USDT
}

// 這個訂單進入 OrderBook
OrderBook 狀態：
賣單（Asks）:
50,700 - 2 BTC
50,600 - 0.5 BTC ← Alice 的平倉單
50,500 - 1 BTC

// 需要有人願意開多倉來接這個單
```

關鍵點：Alice 的平倉單需要找到對手方

* 可能是 Charlie 想新開多倉 0.5 BTC
* 或者是 David（持有空倉）想平空倉 0.5 BTC

<br>

```go
// 撮合後的倉位變化
撮合前：
- Alice: Long 1 BTC @ 49,800
- Charlie: 無倉位

撮合後：
- Alice: Long 0.5 BTC @ 49,800 (減倉)
- Charlie: Long 0.5 BTC @ 50,600 (新開倉)

// 盈虧結算
Alice 已實現盈虧 = (50,600 - 49,800) × 0.5 = +400 USDT
Charlie 未實現盈虧 = 0 (剛開倉)
```

<br>
<br>

## 三、強平的特殊處理

強平是最複雜的場景，因為它是被動和緊急的。

1. 標準強平流程

```go
func (le *LiquidationEngine) ExecuteLiquidation(position *Position) {
    // Step 1: 接管倉位
    position.Status = "LIQUIDATING"
    position.Owner = "LIQUIDATION_ENGINE"
    
    // Step 2: 創建強平訂單
    liquidationOrder := &Order{
        ID:        GenerateOrderID(),
        Type:      "MARKET", // 通常是市價單
        Side:      position.GetOpposingSide(), // 多倉則賣，空倉則買
        Quantity:  position.Size,
        Price:     0, // 市價單不指定價格
        Tag:       "LIQUIDATION",
    }
    
    // Step 3: 注入到 OrderBook 優先撮合
    // 這裡有多種策略...
}
```

<br>

2. 強平訂單的對手方問題

場景：市場暴跌，大量多頭被強平

```go
問題：
- 100個多頭倉位同時觸發強平
- 都需要賣出（平多倉）
- 但誰來買？市場已經恐慌了！

OrderBook 可能的狀態：
賣單（大量強平單）:
45,100 - 50 BTC (強平)
45,090 - 30 BTC (強平)
45,080 - 20 BTC (強平)
...

買單（稀少）:
44,900 - 0.5 BTC
44,800 - 0.3 BTC
```

<br>
<br>

## 四、交易所的解決方案

### 方案 1：強平接管機制

```go
type LiquidationHandler struct {
    insuranceFund *InsuranceFund
    marketMaker   *MarketMaker
}

func (lh *LiquidationHandler) HandleLiquidation(position *Position) {
    // 嘗試 1: 正常市場撮合
    if success := lh.tryMarketExecution(position); success {
        return
    }
    
    // 嘗試 2: 交易所做市商接單
    if success := lh.marketMaker.TakeLiquidation(position); success {
        lh.marketMaker.HedgePosition(position) // 做市商去其他市場對沖
        return
    }
    
    // 嘗試 3: 部分成交 + 自動減倉（ADL）
    lh.executeAutoDeleveraging(position)
}
```

<br>

### 方案 2：自動減倉（Auto-Deleveraging, ADL）

當無法找到對手方時，強制平掉獲利最多的反向倉位：

```go
func (adl *ADLEngine) Execute(liquidatingPosition *Position) {
    // 找出反向倉位（如果強平的是多倉，就找空倉）
    oppositePositions := adl.getRankedOppositePositions(liquidatingPosition)
    
    remainingSize := liquidatingPosition.Size
    
    for _, opponent := range oppositePositions {
        if remainingSize <= 0 {
            break
        }
        
        // 強制撮合
        matchSize := math.Min(remainingSize, opponent.Size)
        
        // 直接結算，不經過 OrderBook
        settlement := &Settlement{
            LiquidatedUser: liquidatingPosition.UserID,
            ADLUser:        opponent.UserID,
            Price:          markPrice, // 使用標記價格
            Quantity:       matchSize,
        }
        
        adl.processSettlement(settlement)
        remainingSize -= matchSize
    }
}

// ADL 優先級排序（誰先被 ADL）
func (adl *ADLEngine) getRankedOppositePositions(position *Position) []*Position {
    // 按 PnL% × 槓桿 排序，獲利最多 + 槓桿最高的優先
    // 這樣設計是為了公平：獲利最多的人先平倉
}
```

<br>

### 方案 3：保險基金直接接管


```go
func (lf *LiquidationFund) DirectTakeover(position *Position) {
    if position.IsLong() {
        // 保險基金成為空頭方
        lf.CreateVirtualPosition("SHORT", position.Size, markPrice)
    } else {
        // 保險基金成為多頭方  
        lf.CreateVirtualPosition("LONG", position.Size, markPrice)
    }
    
    // 基金承擔後續的盈虧
    // 可能稍後在市場平靜時慢慢平掉
}
```

<br>
<br>

## 五、具體實例對比

<br>

## 場景 A：正常市場（有充足流動性）

```go
Alice 平倉/強平 1 BTC：

[OrderBook]
        ↓ Alice 賣出訂單
    撮合引擎匹配
        ↓
    Bob 買入訂單（開多或平空）
        ↓
    正常成交

結果：通過 OrderBook 真實撮合
```

<br>

場景 B：極端市場（流動性枯竭）

```go
Alice 被強平 1 BTC：

[OrderBook]
        ↓ 強平單注入
    沒有對手方！
        ↓
    ADL 系統啟動
        ↓
    強制 Carol（高盈利空頭）平倉

結果：不經過 OrderBook，直接配對結算
```

<br>
<br>

## 六、關鍵理解點

<br>

1. 永續合約的每筆交易都需要對手方

   * 開多需要有人開空或平空
   * 平多需要有人開多或平空


2. 強平的特殊性

   * 強平是被動的，不能等待
   * 必須立即執行，否則虧損擴大
   * 可能需要特殊機制保證成交


3. 不涉及真實資產

   * 全程只是 USDT 的盈虧結算
   * 沒有真實的 BTC 易手
   * 這就是為什麼叫"永續合約"而不是"期貨"


4. 系統設計的複雜性

   * 需要多重保障機制
   * ADL 是最後的防線
   * 保險基金是緩衝墊