package stoploss

import (
	"log"
	"sync"
)

// Exchange wrapper to connect to exchange
type Exchange interface {
	Name() string
	GetBalance(coin string) (string, error)
	GetMarketPrice(baseCoin string, countCount string) (float64, error)
	Sell(baseCoin string, countCount string, quantity string) (string, error)
	Buy(baseCoin string, countCount string, quantity string) (string, error)
	GetCacheLimitOrderId(logger *log.Logger, cachePath string, fileMutex *sync.Mutex, exchange string, orderType string, baseCoin string, countCoin string, quantity string, price float64) (string, error)
	GetLimitOrder(orderId string, baseCoin string, countCoin string, quantity string, price float64) (string, error)
	CancelLimitOrder(orderId string, baseCoin string, countCoin string) error
	SetLimitOrder(orderType string, baseCoin string, countCoin string, quantity string, price float64) (string, error)
	SetCacheLimitOrder(logger *log.Logger, cachePath string, fileMutex *sync.Mutex, exchange string, orderType string, baseCoin string, countCoin string, quantity string, price float64, id string) error
}
