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
	exchange   Exchange
	notify     Notify
	orderType  string
	market     string
	baseCoin   string
	countCoin  string
	lastStop   float64
	price      float64
	quantity   string
	stopFactor float64
	cache      Cache
	cachePath  string
	logger     *log.Logger
	fileMutex  *sync.Mutex
}

type Cache struct {
	Sell []Sell `yaml:"sell"`
}

type Sell struct {
	Pair     string  `yaml:"pair"`
	LastStop float64 `yaml:"lastStop"`
}

// NewTrailing new trailing instance
func NewTrailing(exchange Exchange, notify Notify, orderType string, market string, factor float64, quantity string, price float64, logger *log.Logger, fileMutex *sync.Mutex) *Trailing {
	pair := strings.Split(strings.ToUpper(market), "/")

	tlg := &Trailing{
		exchange:   exchange,
		notify:     notify,
		orderType:  strings.ToUpper(orderType),
		market:     pair[0] + pair[1],
		baseCoin:   pair[0],
		countCoin:  pair[1],
		price:      price,
		quantity:   quantity,
		stopFactor: factor,
		logger:     logger,
		fileMutex:  fileMutex,
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

	stop := tlg.getSellStop(marketPrice)

	// remove me!!!!!!!!!!!!!!!!!!!!!!
	quantity := tlg.quantity
	if quantity == "" {
		quantity, err = tlg.exchange.GetBalance(tlg.baseCoin)
		if err != nil {
			tlg.notify.Send("Cannot get balance, error:" + err.Error())
			return true
		}
	}
	// remove me!!!!!!!!!!!!!!!!!!!!!!

	// Grab the lastStop from the cache
	if tlg.lastStop == 0 {
		exists := false
		for _, v := range tlg.cache.Sell {
			if v.Pair == fmt.Sprintf("%s/%s", tlg.baseCoin, tlg.countCoin) {
				exists = true
				tlg.lastStop = v.LastStop
				//tlg.logger.Printf("Loading %s LastStop: %f\n", v.Pair, v.LastStop)
			}
		}

		if !exists {
			tlg.cache.Sell = append(tlg.cache.Sell, Sell{Pair: fmt.Sprintf("%s/%s", tlg.baseCoin, tlg.countCoin), LastStop: tlg.lastStop})
		}

		stop := tlg.getSellStop(marketPrice)

		if marketPrice <= stop {
			tlg.notify.Send("Preventing sell on first run, clearing the last recorded stop price!")
			tlg.lastStop = 0
		}
	}

	//stop := tlg.getSellStop(marketPrice)

	if marketPrice > stop {
		tlg.notifyStopLossChange(tlg.lastStop, stop, marketPrice)

		tlg.setLastStop(stop)
		return false
	}

	//quantity := tlg.quantity
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
		tlg.notify.Send(fmt.Sprintf("Sell: %s %s - Market Price (%s): %.6f - Order ID: %s", quantity, tlg.baseCoin, tlg.market, marketPrice, order))
	}

	return true
}

func (tlg *Trailing) setLastStop(stop float64) {
	tlg.lastStop = stop

	for i, v := range tlg.cache.Sell {
		if v.Pair == fmt.Sprintf("%s/%s", tlg.baseCoin, tlg.countCoin) {
			tlg.cache.Sell[i].LastStop = tlg.lastStop
			//tlg.logger.Printf("Setting %s LastStop in Cache: %f\n", v.Pair, tlg.lastStop)
		}
	}
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
		tlg.notify.Send(fmt.Sprintf("Buy: %s %s - Market Price (%s): %.6f - Order ID: %s", quantity, tlg.baseCoin, tlg.market, marketPrice, order))
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

func (tlg *Trailing) notifyStopLossChange(prev float64, next float64, price float64) {
	result := big.NewFloat(prev).Cmp(big.NewFloat(next))

	if result == 0 {
		return
	}

	tlg.notify.Send(fmt.Sprintf("Stop-loss %s/%s (%s -%.2f%%): %.6f - Market Price: %.6f", tlg.baseCoin, tlg.countCoin, tlg.orderType, tlg.stopFactor*100, next, price))
}
