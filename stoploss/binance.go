package stoploss

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	binance "github.com/adshao/go-binance/v2"
)

type Binance struct {
	Market
	api *binance.Client
	ctx context.Context
}

func NewBinanceExchange(ctx context.Context, api *binance.Client) *Binance {
	return &Binance{api: api, ctx: ctx}
}

func (exchange *Binance) Name() string {
	return "binance"
}

// GetBalance get balance for coin
func (exchange *Binance) GetBalance(coin string) (string, error) {
	coin = strings.ToUpper(coin)
	account, err := exchange.api.NewGetAccountService().Do(exchange.ctx)
	if err != nil {
		return "0", err
	}

	for _, balance := range account.Balances {
		if strings.ToUpper(balance.Asset) == coin {
			return balance.Free, nil
		}
	}

	return "0", nil
}

// GetMarketPrice get last price for market pair
func (exchange *Binance) GetMarketPrice(baseCoin string, countCount string) (float64, error) {
	prices, err := exchange.api.NewListPricesService().Symbol(exchange.market(baseCoin, countCount)).Do(exchange.ctx)
	if err != nil {
		return 0, err
	}

	for _, p := range prices {
		if p.Symbol == exchange.market(baseCoin, countCount) {
			return strconv.ParseFloat(p.Price, 64)
		}
	}

	return 0, nil
}

// Sell create a sell order to market price
func (exchange *Binance) Sell(baseCoin string, countCount string, quantity string) (string, error) {
	order, err := exchange.api.NewCreateOrderService().Symbol(exchange.market(baseCoin, countCount)).
		Side(binance.SideTypeSell).Type(binance.OrderTypeMarket).
		Quantity(quantity).Do(exchange.ctx)

	if err != nil {
		return "", err
	}

	return strconv.FormatInt(order.OrderID, 10), nil
}

// Buy create a buy order to market price
func (exchange *Binance) Buy(baseCoin string, countCoin string, quantity string) (string, error) {
	order, err := exchange.api.NewCreateOrderService().Symbol(exchange.market(baseCoin, countCoin)).
		Side(binance.SideTypeBuy).Type(binance.OrderTypeMarket).
		Quantity(quantity).Do(exchange.ctx)

	if err != nil {
		return "", err
	}

	return strconv.FormatInt(order.OrderID, 10), nil
}

func (exchange *Binance) market(baseCoin string, countCount string) string {
	return fmt.Sprintf("%s%s", baseCoin, countCount)
}

func (exchange *Binance) GetLimitOrder(orderId string, baseCoin string, countCoin string, quantity string, price float64) (string, error) {
	id, err := strconv.ParseInt(orderId, 10, 64)
	if err != nil {
		return "", err
	}

	order, err := exchange.api.NewGetOrderService().Symbol(exchange.market(baseCoin, countCoin)).
		OrderID(id).Do(exchange.ctx)

	if err != nil {
		return "", err
	}

	return strconv.FormatInt(order.OrderID, 10), nil
}

func (exchange *Binance) CancelLimitOrder(orderId string, baseCoin string, countCoin string) error {
	id, err := strconv.ParseInt(orderId, 10, 64)
	if err != nil {
		return err
	}

	_, err = exchange.api.NewCancelOrderService().Symbol(exchange.market(baseCoin, countCoin)).
		OrderID(id).Do(exchange.ctx)

	if err != nil {
		return err
	}

	return nil
}

func (exchange *Binance) SetLimitOrder(orderType string, baseCoin string, countCoin string, quantity string, price float64) (string, error) {
	priceStr := fmt.Sprintf("%f", price)
	fmt.Println("limit price:", priceStr)
	if orderType == "SELL" {
		order, err := exchange.api.NewCreateOrderService().Symbol(exchange.market(baseCoin, countCoin)).
			Side(binance.SideTypeSell).Type(binance.OrderTypeLimit).
			TimeInForce(binance.TimeInForceTypeGTC).Quantity(quantity).
			Price(priceStr).Do(exchange.ctx)

		if err != nil {
			return "", err
		}

		return strconv.FormatInt(order.OrderID, 10), nil

		// Use Test() instead of Do() for testing.
	}
	return "", nil
}
