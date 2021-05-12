package stoploss

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
)

// Trailing stop-loss runner
type Trailing struct {
	exchange               Exchange
	notify                 Notify
	orderType              string
	market                 string
	baseCoin               string
	countCoin              string
	lastStop               float64
	lastStopCache          float64
	lastStopBuyPriceFactor float64
	price                  float64
	quantity               string
	startStopFactor        float64
	endStopFactor          float64
	maxLossStopFactor      float64
	limitSellFactor        float64
	buyPrice               float64
	observedHighPrice      float64
	cachePath              string
	logger                 *log.Logger
	fileMutex              *sync.Mutex
	chatNotify             string
	exchangeBalance        string
}

// NewTrailing new trailing instance
func NewTrailing(
	exchange Exchange,
	notify Notify,
	orderType string,
	market string,
	minStopFactor float64,
	maxStopFactor float64,
	maxLossStopFactor float64,
	limitSellFactor float64,
	buyPrice float64,
	quantity string,
	price float64,
	logger *log.Logger,
	fileMutex *sync.Mutex,
	chatNotify string,
) *Trailing {
	pair := strings.Split(strings.ToUpper(market), "/")

	tlg := &Trailing{
		exchange:          exchange,
		notify:            notify,
		orderType:         strings.ToUpper(orderType),
		market:            pair[0] + pair[1],
		baseCoin:          pair[0],
		countCoin:         pair[1],
		price:             price,
		quantity:          quantity,
		startStopFactor:   minStopFactor,
		endStopFactor:     maxStopFactor,
		maxLossStopFactor: maxLossStopFactor,
		limitSellFactor:   limitSellFactor,
		buyPrice:          buyPrice,
		logger:            logger,
		fileMutex:         fileMutex,
		chatNotify:        chatNotify,
	}

	if tlg.orderType == "BUY" {
		tlg.setLastStop(math.MaxFloat64)
	}

	tlg.loadCache(".cache.yaml", tlg.fileMutex)

	return tlg
}

// RunStop check stop loss apply
func (tlg *Trailing) RunStop() bool {
	run := *new(bool)

	if tlg.orderType == "BUY" {
		run = tlg.runBuy()
	} else {

		run = tlg.runSell()
	}

	tlg.SaveCache(tlg.fileMutex)

	return run
}

func (tlg *Trailing) runSell() bool {
	quantity, checkFailed := tlg.getQuantity()
	if checkFailed == true {
		return true
	} else {
		tlg.notify.Send(fmt.Sprintf("Ready to sell %s %s coins on %s exchange", quantity, tlg.baseCoin, tlg.exchange.Name()))
	}

	if setFailed := tlg.setHighSellLimitOrder(quantity); setFailed == true {
		return true
	}

	marketPrice, err := tlg.exchange.GetMarketPrice(tlg.baseCoin, tlg.countCoin)
	if err != nil {
		tlg.notify.Send("Cannot get market price, error:" + err.Error())
		return true
	}

	tlg.setObservedHighPrice(marketPrice)

	// TODO: Set an upper threshold before this kicks in Sell Threshold - lookup what it was bought for and add x percentage
	// need either the start number or the threshold number manually or automatically

	// I need it to set a limit sell on the high end
	// output quantity of coins on startup

	// check my available funds to make sure it will really sell
	// start watching trade when buy is detected - but need to deal with what happens with price dropping below the buy price immediately

	// display totals every 30 mins or hour in chat
	// give more summary numbers on the SELL log
	// I need it to tell me it is getting close to selling
	// print the stop-loss line when restored from the cache
	// --print the percentage loss/gain based on the start_price
	// --prevent sell from bad buy_price on first run

	// Grab the lastStop from the cache
	if tlg.lastStop == 0 {
		if tlg.lastStopCache > 0 {
			tlg.setLastStop(tlg.lastStopCache)
		}

		stop := math.Max(tlg.getSellStop(marketPrice), tlg.getBuyPriceSellStop(tlg.buyPrice))

		if marketPrice <= stop {
			tlg.notify.Send("Preventing sell on first run, clearing the last recorded stop price!")
			tlg.setLastStop(0)
		}
	}

	stop := math.Max(tlg.getSellStop(marketPrice), tlg.getBuyPriceSellStop(tlg.buyPrice))
	//fmt.Printf("Stop: %f\n", stop)

	if marketPrice > stop {
		tlg.notifyStopLossChange(tlg.lastStop, stop, marketPrice)
		tlg.setLastStop(stop)
		return false
	}

	//quantity := tlg.quantity
	//if quantity == "" {
	//	quantity, err = tlg.exchange.GetBalance(tlg.baseCoin)
	//	if err != nil {
	//		tlg.notify.Send("Cannot get balance, error:" + err.Error())
	//		return true
	//	}
	//}

	order, err := tlg.exchange.Sell(tlg.baseCoin, tlg.countCoin, quantity)
	if err != nil {
		tlg.notify.Send("market: " + tlg.market + " quantity: " + quantity)
		tlg.notify.Send("Cannot create sell order, error:" + err.Error())
	} else {
		tlg.notify.Send(fmt.Sprintf("Sell: %s %s - Market Price (%s): %.8f - Order ID: %s - %s", quantity, tlg.baseCoin, tlg.market, marketPrice, order, tlg.chatNotify))
	}

	return true
}

func (tlg *Trailing) setLastStop(stop float64) {
	tlg.lastStop = stop
	tlg.lastStopBuyPriceFactor = tlg.buyPrice / stop
}

func (tlg *Trailing) setObservedHighPrice(price float64) {
	if price > tlg.observedHighPrice {
		tlg.observedHighPrice = price
	}
}

func (tlg *Trailing) getSellStop(price float64) float64 {
	if tlg.startStopFactor > 0 && tlg.endStopFactor > 0 {
		currentPriceStop := tlg.computeSellStop(price)
		//tlg.logger.Printf("Current Price Sell: %.6f", currentPriceStop)

		observedHighPriceStop := tlg.computeSellStop(tlg.observedHighPrice)
		//tlg.logger.Printf("Observed High Sell: %.6f", observedHighPriceStop)

		return math.Max(math.Max(tlg.lastStop, currentPriceStop), observedHighPriceStop)
	}

	return tlg.price
}

func (tlg *Trailing) computeSellStop(price float64) float64 {
	min := tlg.startStopFactor
	max := tlg.endStopFactor

	minSellRange := tlg.buyPrice + (tlg.buyPrice * tlg.endStopFactor)
	maxSellRange := tlg.buyPrice + (tlg.buyPrice * (tlg.limitSellFactor))
	//fmt.Printf("range: %f - %f\n", minSellRange, maxSellRange)

	percent := (price - minSellRange) / (maxSellRange - minSellRange)
	rangeStopFactor := percent*(max-min) + min

	var sellStop float64
	if price <= minSellRange {
		sellStop = price * (1 - tlg.startStopFactor)
	} else if price > minSellRange && price < maxSellRange {
		//fmt.Printf("range stop factor: %f\n", rangeStopFactor)
		//fmt.Printf("range result: %f\n", price * (1-rangeStopFactor))
		sellStop = price * (1 - rangeStopFactor)
	} else {
		sellStop = price * (1 - tlg.endStopFactor)
	}
	return sellStop
}

func (tlg *Trailing) getQuantity() (string, bool) {
	if tlg.lastStop == 0 {
		var err error
		tlg.exchangeBalance, err = tlg.exchange.GetBalance(tlg.baseCoin)
		if err != nil {
			tlg.notify.Send("Cannot get balance, error:" + err.Error())
			return "", true
		}

		exchangeBalanceFlt := float64(0)
		exchangeBalanceFlt, err = strconv.ParseFloat(tlg.exchangeBalance, 64)
		if err != nil {
			tlg.notify.Send("Cannot parse the exchangeBalance to a float, error:" + err.Error())
			return "", true
		}

		quantityFlt := float64(0)
		quantityFlt, err = strconv.ParseFloat(tlg.quantity, 64)
		if err != nil {
			tlg.notify.Send("Cannot parse the quantity to a float, error:" + err.Error())
			return "", true
		}

		if exchangeBalanceFlt > 0.0 {
			if tlg.quantity != "" && exchangeBalanceFlt < quantityFlt {
				tlg.notify.Send(fmt.Sprintf("Amount is set to sell %.3f coins, but only found %.3f %s coins available on the %s exchange, exiting",
					quantityFlt, exchangeBalanceFlt, tlg.baseCoin, tlg.exchange.Name()))
				return "", true
			}

			tlg.notify.Send(fmt.Sprintf("Found %.3f %s coins available to sell on the %s exchange", exchangeBalanceFlt, tlg.baseCoin, tlg.exchange.Name()))

			if tlg.quantity == "" {
				return tlg.exchangeBalance, false
			} else {
				return tlg.quantity, false
			}
		} else {
			tlg.notify.Send(fmt.Sprintf("Cannot find any %s coins on the %s exchange", tlg.baseCoin, tlg.exchange.Name()))
			return "", true
		}
	}
}

//func (tlg *Trailing) setHighSellLimitOrder(quantity string) bool {
//	if tlg.lastStop == 0 && tlg.limitSellFactor > 0.0 {
//		limitSellOrder := tlg.exchange.GetLimitOrder("SELL")
//
//		if limitSellOrder > 0 {
//			tlg.exchange.CancelLimitOrder("SELL", orderId)
//		}
//
//		sellPrice := tlg.buyPrice + (tlg.buyPrice * tlg.limitSellFactor)
//		tlg.exchange.SetLimitOrder("SELL", tlg.baseCoin, tlg.countCoin, quantity, sellPrice)
//	}
//	return false
//}
//
//func (tlg *Trailing) getBuyPriceSellStop(buyPrice float64) float64 {
//	if tlg.maxLossStopFactor > 0 {
//		//tlg.logger.Printf("BuyPrice Sell: %.6f\n\n", buyPrice * (1-tlg.maxLossStopFactor))
//		return buyPrice * (1 - tlg.maxLossStopFactor)
//	}
//
//	return tlg.price
//}

func (tlg *Trailing) notifyStopLossChange(prev float64, next float64, price float64) {
	result := big.NewFloat(prev).Cmp(big.NewFloat(next))

	if result == 0 {
		return
	}

	marketPriceGain := ((price - tlg.buyPrice) / tlg.buyPrice) * 100
	stopPriceGain := ((next - tlg.buyPrice) / tlg.buyPrice) * 100
	stopToMarketDiff := ((next - price) / price) * 100
	tlg.notify.Send(fmt.Sprintf("%s/%s (%s Start:-%.1f%% End:-%.1f%% Max:-%.1f%%) Stop Price: %.8f (%.3f%%) Market Price: %.8f (%.3f%%) Stop/Market: %.3f%%", tlg.baseCoin, tlg.countCoin, tlg.orderType, tlg.startStopFactor*100, tlg.endStopFactor*100, tlg.maxLossStopFactor*100, next, stopPriceGain, price, marketPriceGain, stopToMarketDiff))
}

func (tlg *Trailing) runBuy() bool {
	marketPrice, err := tlg.exchange.GetMarketPrice(tlg.baseCoin, tlg.countCoin)
	if err != nil {
		tlg.notify.Send("Cannot get market price, error:" + err.Error())
		return true
	}

	stop := tlg.getBuyStop(marketPrice)

	if stop > marketPrice {
		tlg.notifyStopLossChange(tlg.lastStop, stop, marketPrice)

		tlg.setLastStop(stop)
		return false
	}

	quantity := tlg.quantity
	if quantity == "" {
		quantity, err = tlg.exchange.GetBalance(tlg.countCoin)
		if err != nil {
			tlg.notify.Send("Cannot get balance, error:" + err.Error())
			return true
		}
	}

	order, err := tlg.exchange.Buy(tlg.baseCoin, tlg.countCoin, quantity)
	if err != nil {
		tlg.notify.Send("Cannot create buy order, error:" + err.Error())
	} else {
		tlg.notify.Send(fmt.Sprintf("Buy: %s %s - Market Price (%s): %.8f - Order ID: %s", quantity, tlg.baseCoin, tlg.market, marketPrice, order))
	}

	return true
}

func (tlg *Trailing) getBuyStop(price float64) float64 {
	if tlg.startStopFactor > 0 && tlg.endStopFactor > 0 {
		return math.Min(tlg.lastStop, price*(1+tlg.startStopFactor))
	}

	return tlg.price
}
