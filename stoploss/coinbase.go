package stoploss

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/preichenberger/go-coinbasepro/v2"
)

type Coinbase struct {
	Market
	api *coinbasepro.Client
	ctx context.Context
}

func NewCoinbaseExchange(ctx context.Context, api *coinbasepro.Client) *Coinbase {
	return &Coinbase{api: api, ctx: ctx}
}

func (exchange *Coinbase) Name() string {
	return "coinbase"
}

// GetBalance get balance for coin
func (exchange *Coinbase) GetBalance(coin string) (string, int, error) {
	coin = strings.ToUpper(coin)

	accounts, err := exchange.api.GetAccounts()
	if err != nil {
		return "0", 0, err
	}

	//fmt.Printf("Accounts: %+v\n", accounts)

	for _, account := range accounts {
		if strings.ToUpper(account.Currency) == coin {
			return account.Available, 0, nil // TODO: FIX THIS ZERO!!
		}
	}

	return "0", 0, nil
}

// GetMarketPrice get last price for market pair
func (exchange *Coinbase) GetMarketPrice(baseCoin string, countCount string) (float64, error) {
	ticker, err := exchange.api.GetTicker(exchange.market(baseCoin, countCount))
	if err != nil {
		return 0, err
	}

	//fmt.Printf("Ticker: %+v\n", ticker)

	if ticker.Price != "" {
		return strconv.ParseFloat(ticker.Price, 64)
	}

	return 0, nil
}

// Sell create a sell order to market price
func (exchange *Coinbase) Sell(baseCoin string, countCount string, quantity string) (string, error) {
	order := coinbasepro.Order{
		Side:      "sell",
		Type:      "market",
		Size:      quantity,
		ProductID: exchange.market(baseCoin, countCount),
	}

	savedOrder, err := exchange.api.CreateOrder(&order)
	if err != nil {
		return "", err
	}

	return savedOrder.ID, nil
}

// Buy create a buy order to market price
func (exchange *Coinbase) Buy(baseCoin string, countCount string, quantity string) (string, error) {
	order := coinbasepro.Order{
		Side:      "buy",
		Type:      "market",
		Size:      quantity,
		ProductID: exchange.market(baseCoin, countCount),
	}

	savedOrder, err := exchange.api.CreateOrder(&order)
	if err != nil {
		return "", err
	}

	return savedOrder.ID, nil
}

func (exchange *Coinbase) market(baseCoin string, countCount string) string {
	return fmt.Sprintf("%s-%s", baseCoin, countCount)
}
