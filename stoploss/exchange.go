package stoploss

import (
	"log"
	"sync"
)

// Exchange wrapper to connect to exchange
type Exchange interface {
	Name() string
	GetBalance(coin string) (string, int, error)
	GetMarketPrice(baseCoin string, countCount string) (float64, error)
	Sell(baseCoin string, countCount string, quantity string) (string, error)
	Buy(baseCoin string, countCount string, quantity string) (string, error)
	LoadCacheLimitOrders(logger *log.Logger, cachePath string, fileMutex *sync.Mutex)
	GetCacheLimitOrderId(logger *log.Logger, exchange string, orderType string, baseCoin string, countCoin string, quantity string, price float64) (string, string, error)
	GetLimitOrderRunning(orderId string, baseCoin string, countCoin string) (bool, error)
	GetLimitOrderStatusFilled(orderId string, baseCoin string, countCoin string) (bool, error)
	GetCacheLimitOrderQuantity(exchange string, orderType string, baseCoin string, countCoin string) (string, error)
	CancelLimitOrder(orderId string, baseCoin string, countCoin string) error
	SetLimitOrder(orderType string, baseCoin string, countCoin string, quantity string, price string) (string, error)
	SetCacheLimitOrder(logger *log.Logger, cachePath string, fileMutex *sync.Mutex, exchange string, orderType string, baseCoin string, countCoin string, quantity string, price float64, id string) error
}
