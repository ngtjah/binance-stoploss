package stoploss

// Exchange wrapper to connect to exchange
type Exchange interface {
	GetBalance(coin string) (string, error)
	GetMarketPrice(baseCoin string, countCount string) (float64, error)
	Sell(baseCoin string, countCount string, quantity string) (string, error)
	Buy(baseCoin string, countCount string, quantity string) (string, error)
	Name() string
}
