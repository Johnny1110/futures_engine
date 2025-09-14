# Position Performance Optimization

## Overview

專注於 Position 結構的效能優化，目標是實現高頻交易場景下的極致效能。

## 當前 Position 效能分析

![2.png](imgs/2.png)

### 效能瓶頸識別

```go
// 當前實現的問題
type Position struct {
    // 1. Decimal 計算慢
    Size             decimal.Decimal  // 每次運算都要分配記憶體
    EntryPrice       decimal.Decimal  // 無法利用 CPU 浮點運算優化
    UnrealizedPnL    decimal.Decimal  // 複雜的字串解析和轉換
    
    // 2. 鎖粒度太大
    mu sync.RWMutex                   // 每個操作都要搶鎖
    
    // 3. 記憶體浪費
    sizeFloat       float64           // 重複儲存相同數據
    entryPriceFloat float64           // 增加記憶體使用量
}
```

**基準測試結果問題：**
- `UpdateMarkPrice`: 單次操作耗時過長
- `LiquidationCheck`: 大量倉位時效能劣化嚴重
- 記憶體分配頻繁，GC 壓力大

## Position 極速重構計劃

### Phase 1: 數據結構優化

```go
// 目標：極致效能的 Position
type FastPosition struct {
    // === Core Data (Cache-Friendly Layout) ===
    size         float64  // 8 bytes
    entryPrice   float64  // 8 bytes  
    markPrice    float64  // 8 bytes
    unrealizedPnL float64 // 8 bytes
    // 32 bytes cache line alignment
    
    // === Metadata ===
    userID       string
    symbol       string
    side         PositionSide
    status       PositionStatus
    leverage     uint16        // 節省記憶體
    
    // === Precision Control ===
    sizePrecision  int8       // 1 byte
    pricePrecision int8       // 1 byte
    
    // === Margin Info ===
    initialMargin     float64
    maintenanceMargin float64
    liquidationPrice  float64
    
    // === PnL ===
    realizedPnL float64
    
    // === Timestamps ===
    openTime   int64  // Unix timestamp, 8 bytes
    updateTime int64  // Unix timestamp, 8 bytes
    
    // === Lock-Free or Minimal Locking ===
    // 使用 atomic 操作替代 mutex
    dirty int32  // atomic flag for cache invalidation
}
```

### Phase 2: 無鎖/最小鎖定設計

```go
// 核心操作使用 atomic
func (p *FastPosition) UpdateMarkPrice(price float64) {
    // 1. Atomic update mark price
    atomic.StoreUint64((*uint64)(unsafe.Pointer(&p.markPrice)), math.Float64bits(price))
    
    // 2. Calculate PnL without locks
    var newPnL float64
    if p.side == LONG {
        newPnL = (price - p.entryPrice) * p.size
    } else {
        newPnL = (p.entryPrice - price) * p.size
    }
    
    // 3. Atomic update PnL
    atomic.StoreUint64((*uint64)(unsafe.Pointer(&p.unrealizedPnL)), math.Float64bits(newPnL))
    
    // 4. Mark dirty for cache invalidation
    atomic.StoreInt32(&p.dirty, 1)
}

// 讀取操作也是 lock-free
func (p *FastPosition) GetUnrealizedPnL() float64 {
    return math.Float64frombits(atomic.LoadUint64((*uint64)(unsafe.Pointer(&p.unrealizedPnL))))
}
```

### Phase 3: 計算優化

```go
// 預計算和快取熱路徑數據
type PositionCache struct {
    // 預計算的值，減少重複計算
    positionValue    float64
    marginRatio      float64
    isLiquidatable   bool
    
    // 快取時間戳
    lastUpdate       int64
}

// 批量計算優化
func BatchUpdatePositions(positions []*FastPosition, prices map[string]float64) {
    // 使用 SIMD 或向量化操作
    for symbol, price := range prices {
        // 找到所有該 symbol 的倉位
        symbolPositions := getPositionsBySymbol(positions, symbol)
        
        // 批量更新，利用 CPU cache locality
        for _, pos := range symbolPositions {
            pos.UpdateMarkPrice(price)
        }
    }
}
```

## 實施計劃

### 🎯 Phase 1: Float64 Migration (Week 1)
- [ ] 創建 `FastPosition` 結構體
- [ ] 實現基本 CRUD 操作
- [ ] 精度控制機制
- [ ] 單元測試覆蓋

### ⚡ Phase 2: Lock-Free Operations (Week 2)  
- [ ] Atomic 操作實現 `UpdateMarkPrice`
- [ ] Lock-free 讀取操作
- [ ] 線程安全驗證
- [ ] 併發測試

### 🚀 Phase 3: Calculation Optimization (Week 3)
- [ ] 批量操作優化
- [ ] 預計算快取機制
- [ ] SIMD 向量化（可選）
- [ ] 效能基準測試


## 預期效能提升目標

| 操作 | 當前效能 | 目標效能 | 提升倍數 |
|------|----------|----------|----------|
| UpdateMarkPrice | ~500ns | ~50ns | **10x** |
| PnL Calculation | ~300ns | ~30ns | **10x** |
| Liquidation Check | ~200ns | ~20ns | **10x** |
| Memory Usage | 100% | 60% | **40% 減少** |
| GC Pressure | 高 | 極低 | **90% 減少** |

## 風險控制

1. **精度測試**：Float64 vs Decimal 精度比較測試
2. **回歸測試**：確保所有現有測試通過
3. **並發安全**：Lock-free 操作的正確性驗證
4. **逐步遷移**：先並行運行，驗證無誤後切換

---
