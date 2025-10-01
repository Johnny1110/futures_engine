# Funding Rate

<br>

---

<br>

## 1. Funding Rate Model Overview

The funding rate is determined by two main factors: interest rate and premium index. 
This rate is used to keep the contract price close to the spot price.

<br>
<br>

## 2. Basic Funding Rate Parameters

### Interest Rate (I)

* Setting: The default value is 0.03% per day, with funding rate calculation executed every 8 hours, resulting in an interest rate per funding interval of 0.01%.
  * ie: the funding frequency is n hour(configurable), interest rate 0.03% per day, so interest rate per funding interval = 0.03%/(24/n)

* if n = 8, r=0.01%. n=4 r = 0.005% 

<br>

### Premium Index (P)

* The premium index is used to reflect the difference between the contract price and the spot price.

* Premium index calculation formula:

    ```
    Premium Index (P) = [Max(0, Impact Bid Price - Price Index) - Max(0, Price Index - Impact Ask Price)] / Price Index
    ```

    Here, the impact bid price and impact ask price are derived by simulating trades for a specific margin amount on the order book to obtain average prices.

<br>
<br>

## 3. Funding Rate Calculation Steps

<br>

### Step 1: Calculate Impact Bid/Ask Price

* The impact margin amount (IMN) is used to determine the average impact bid or ask price in the order book. For USDT-margined contracts, IMN is the value equivalent to 200 USDT for trading

    Impact Margin Amount (IMN) = 200 USDT * Highest Leverage.
    (200U is configurable on admin, see 7.3 for suggestion default)

    ie: `MKRUSDT max leverage =20, IMN = 200*20 = 4000 USDT`

* The system simulates trades based on the corresponding impact margin amount (IMN) in the order book to calculate the average impact bid and ask prices.

* Exception 1: if accumulated order books size is not enough for IMN

  * if total bids < IMN,  Impact Bid Price = max (impact bid price of total bids , bid1 * (1-2%))

  * if total asks < IMN,  Impact Ask Price = min(impact ask price of total asks, ask1 * (1+2%))

* Exception 2: if no bid or ask on order books

  * if no bids,  Impact Bid Price = Mark price * (1-2%)

  * if no asks,  Impact Ask Price = Mark price * (1+2%)

<br>

### Step 2: Calculate Premium Index Series

* The system calculates the premium index every 30 seconds and collects all premium index data points within the funding rate period.

* The number of data points used for calculating the funding rate is: n = (60 / 30) * 60 * number of hours in the funding rate interval.
ie: 8 hours period, n = 2*60*8 = 960,  4 hours period, n = 2*60*4 = 480

* Please note that the funding rate shown represents an estimation of the “last 8 hours of the premium index”.
For example, from 09:00 UTC, the funding rate calculation uses the premium index dataset from 01:00 UTC to 09:00 UTC (rather than from 08:00 UTC to 09:00 UTC).

<br>

### Step 3: Calculate Time-Weighted Average Premium Index


* Using the premium index series obtained in Step 2, calculate the time-weighted average.

* Calculation formula:

    ```
    Average Premium Index (P) = (1 * Premium Index_1 + 2 * Premium Index_2 + 3 * Premium Index_3 +···+ n * Premium Index_n) / (1 + 2 + 3 +···+ n) = (∑ (Weight * Premium Index)) / (∑ Weight)
    ```

    Where the weight is the weight of each time point, increasing in chronological order.

* In 8 hours periods, we have 960 Premium index

    ```
    Average Premium Index (P) = (1 * Premium Index_1 + 2 * Premium Index_2 + 3 * Premium Index_3 +···+ 960 * Premium Index_960) / (1 + 2 + 3 +···+ 960) 
    ```
  
<br>

### Step 4: Calculate Funding Rate (F)

* The funding rate is calculated based on the average premium index and the interest rate, with a buffer 0.05% added.

* Calculation formula:

    ```
    Funding Rate (F) = Current Average Premium Index (P) + Clamp(Interest Rate - Currenc Average Premium Index (P), 0.05%, -0.05%)
    ```
    
    note: Premium Index (P) here refers to the current average and interest rate is calculated by its funding period(0.03%/(24/n)

<br>
<br>

## Risk Management

<br>

Funding rate Cap will be manually maintained on management portal

### Funding rate Cap reference

|    | 100X leverage | 50X    | 20X   |
|  ----  |---------------|--------|-------|
| cap/floor | 0.3%          | 0.375% | 2%~3% |

<br>

### Management Portal:

| contract | daily interest rate  | impact size(USDT) | funding interval(Hour) | Funding rate cap/floor   | Mark Price | index price | premium index(%)   | Order book method(%) | Market neutral method(%) | Binance funding interval(Hour) | Binance funding rate |
|----------|---------------|-------------------|------------------------|-------|------------|-------------|-------|----------------------|--------------------------|--------------------------------|----------------------|
| BTCPERP  | 0.03         | 200               | 8                      | 0.3 | 70000      | 70010       | 0.02 | 0.01                 | 0.012%                   | 8                              | 0.01%                |

<br>

Definition:

* Order book Method: The above new formula, derived from order books.
* Market-neutral method : Old funding method from mm position

* Parameter: `daily interest rate / impact size(USDT) / funding interval(Hour) / Funding rate cap(%)`



* Monitor: `Mark Price / index price / premium index(%) / Order book method(%) / Market neutral method(%) / Binance funding interval(Hour)`

* if there is no Market neutral mothod, show “-“

* The default method is Market neutral method in the beginning and add a swtich that can choose different mothods

* Due to low depths and wide BBO spread on altcoin, we suggest following Impact size setting as default

| Symbol    | 100X leverage | 50X | 20X |
|-----------|---------------|----|-----|
| Impact size | 200           | 20 | 20  |

<br>
<br>


---

<br>
<br>

Reference: https://www.binance.com/zh-TC/support/faq/detail/360033525031?ref=VU3NZ3AJ