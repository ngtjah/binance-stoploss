package stoploss

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"strings"
	"sync"
)

// Trailing stop-loss runner
type Trailing struct {
	exchange      Exchange
	notify        Notify
	orderType     string
	market        string
	baseCoin      string
	countCoin     string
	lastStop      float64
	lastStopCache float64
	price         float64
	quantity      string
	stopFactor    float64
	buyStopFactor float64
	buyPrice      float64
	cachePath     string
	logger        *log.Logger
	fileMutex     *sync.Mutex
	chatNotify    string
}

// NewTrailing new trailing instance
func NewTrailing(exchange Exchange, notify Notify, orderType string, market string, stopFactor float64, buyStopFactor float64, buyPrice float64, quantity string, price float64, logger *log.Logger, fileMutex *sync.Mutex) *Trailing {
	pair := strings.Split(strings.ToUpper(market), "/")

	tlg := &Trailing{
		exchange:      exchange,
		notify:        notify,
		orderType:     strings.ToUpper(orderType),
		market:        pair[0] + pair[1],
		baseCoin:      pair[0],
		countCoin:     pair[1],
		price:         price,
		quantity:      quantity,
		stopFactor:    stopFactor,
		buyStopFactor: buyStopFactor,
		buyPrice:      buyPrice,
		logger:        logger,
		fileMutex:     fileMutex,
		chatNotify:    "<@U0CAD3TLP>",
	}

	if tlg.orderType == "BUY" {
		tlg.lastStop = math.MaxFloat64
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
	marketPrice, err := tlg.exchange.GetMarketPrice(tlg.baseCoin, tlg.countCoin)
	if err != nil {
		tlg.notify.Send("Cannot get market price, error:" + err.Error())
		return true
	}
	// TODO: Set an upper threshold before this kicks in Sell Threshold - lookup what it was bought for and add x percentage
	// need ether the start number or the threshold number manually or automatically

	// I need it to tell me it is getting close to selling
	// I need it to @ me when it sells
	// I need it to set a limit sell on the high end
	// check my available funds to make sure it will really sell
	// print the stop-loss line when restored from the cache
	// print the percentage loss/gain based on the start_price
	// prevent sell from bad buy_price on first run

	// Grab the lastStop from the cache
	if tlg.lastStop == 0 {
		if tlg.lastStopCache > 0 {
			tlg.lastStop = tlg.lastStopCache
		}

		stop := math.Max(tlg.getSellStop(marketPrice), tlg.getBuySellStop(tlg.buyPrice))

		if marketPrice <= stop {
			tlg.notify.Send("Preventing sell on first run, clearing the last recorded stop price!")
			tlg.lastStop = 0
		}
	}

	stop := math.Max(tlg.getSellStop(marketPrice), tlg.getBuySellStop(tlg.buyPrice))

	//if marketPrice > stop && marketPrice > buyStop {
	//	if buyStop > stop && buyStop > 0 {
	//		tlg.notifyStopLossChange(tlg.lastStop, buyStop, marketPrice)
	//		tlg.lastStop = buyStop
	//	} else {
	if marketPrice > stop {
		tlg.notifyStopLossChange(tlg.lastStop, stop, marketPrice)
		tlg.lastStop = stop
		//}
		return false
	}

	quantity := tlg.quantity
	if quantity == "" {
		quantity, err = tlg.exchange.GetBalance(tlg.baseCoin)
		if err != nil {
			tlg.notify.Send("Cannot get balance, error:" + err.Error())
			return true
		}
	}

	order, err := tlg.exchange.Sell(tlg.baseCoin, tlg.countCoin, quantity)
	if err != nil {
		tlg.notify.Send("market: " + tlg.market + " quantity: " + quantity)
		tlg.notify.Send("Cannot create sell order, error:" + err.Error())
	} else {
		tlg.notify.Send(fmt.Sprintf("Sell: %s %s - Market Price (%s): %.8f - Order ID: %s - %s", quantity, tlg.baseCoin, tlg.market, marketPrice, order, tlg.chatNotify))
	}

	return true
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

		tlg.lastStop = stop
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
	if tlg.stopFactor > 0 {
		return math.Min(tlg.lastStop, price*(1+tlg.stopFactor))
	}

	return tlg.price
}

func (tlg *Trailing) getSellStop(price float64) float64 {
	if tlg.stopFactor > 0 {
		return math.Max(tlg.lastStop, price*(1-tlg.stopFactor))
	}

	return tlg.price
}

func (tlg *Trailing) getBuySellStop(price float64) float64 {
	if tlg.buyStopFactor > 0 {
		return math.Max(tlg.lastStop, price*(1-tlg.buyStopFactor))
	}

	return tlg.price
}

func (tlg *Trailing) notifyStopLossChange(prev float64, next float64, price float64) {
	result := big.NewFloat(prev).Cmp(big.NewFloat(next))

	if result == 0 {
		return
	}

	tlg.notify.Send(fmt.Sprintf("Stop-loss %s/%s (%s -%.2f%% -%.2f%%): %.8f - Market Price: %.8f", tlg.baseCoin, tlg.countCoin, tlg.orderType, tlg.stopFactor*100, tlg.buyStopFactor*100, next, price))
}
