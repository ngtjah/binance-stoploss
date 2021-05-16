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
func (exchange *Binance) GetBalance(coin string) (string, int, error) {
	coin = strings.ToUpper(coin)
	account, err := exchange.api.NewGetAccountService().Do(exchange.ctx)
	if err != nil {
		return "0", 0, err
	}

	var precision int
	precision, err = exchange.GetExchangeInfo(coin)
	if err != nil {
		return "0", 0, err
	}

	for _, balance := range account.Balances {
		if strings.ToUpper(balance.Asset) == coin {
			return balance.Free, precision, nil
		}
	}

	return "0", 0, nil
}

func (exchange *Binance) GetExchangeInfo(coin string) (int, error) {
	exchangeInfo, err := exchange.api.NewExchangeInfoService().Do(exchange.ctx)
	if err != nil {
		return 0, err
	}

	//fmt.Printf("exchangeInfo: %v", exchangeInfo.Symbols)
	for _, symbol := range exchangeInfo.Symbols {
		if strings.ToUpper(symbol.BaseAsset) == coin {
			zero, dot := "0", "."
			tickSize := strings.TrimRight(strings.TrimRight(symbol.PriceFilter().TickSize, zero), dot)
			//fmt.Println("ticksize:", tickSize)
			s := tickSize
			i := strings.IndexByte(tickSize, '.')
			if i > -1 {
				return len(s) - i - 1, nil
			}
			return 0, nil
		}
	}
	return 0, nil
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

func (exchange *Binance) GetLimitOrderStatusFilled(orderId string, baseCoin string, countCoin string) (bool, error) {
	id, err := strconv.ParseInt(orderId, 10, 64)
	if err != nil {
		return false, err
	}

	order, err := exchange.api.NewGetOrderService().Symbol(exchange.market(baseCoin, countCoin)).
		OrderID(id).Do(exchange.ctx)

	if err != nil {
		return false, err
	}

	if order.Status == binance.OrderStatusTypeFilled {
		fmt.Println("Order Status:", order.Status)
		return true, nil
	} else {
		return false, nil
	}
}

func (exchange *Binance) GetLimitOrderRunning(orderId string, baseCoin string, countCoin string) (bool, error) {
	id, err := strconv.ParseInt(orderId, 10, 64)
	if err != nil {
		return false, err
	}

	order, err := exchange.api.NewGetOrderService().Symbol(exchange.market(baseCoin, countCoin)).
		OrderID(id).Do(exchange.ctx)

	if err != nil {
		return false, err
	}

	if order.Status == binance.OrderStatusTypeNew || order.Status == binance.OrderStatusTypePartiallyFilled {
		fmt.Println("Order IsRunning:", order.Status)
		return true, nil
	} else {
		return false, nil
	}
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

func (exchange *Binance) SetLimitOrder(orderType string, baseCoin string, countCoin string, quantity string, price string) (string, error) {
	fmt.Println("limit price:", price)
	if orderType == "SELL" {
		order, err := exchange.api.NewCreateOrderService().Symbol(exchange.market(baseCoin, countCoin)).
			Side(binance.SideTypeSell).Type(binance.OrderTypeLimit).
			TimeInForce(binance.TimeInForceTypeGTC).Quantity(quantity).
			Price(price).Do(exchange.ctx)

		if err != nil {
			return "", err
		}

		return strconv.FormatInt(order.OrderID, 10), nil

		// Use Test() instead of Do() for testing.
	}
	return "", nil
}
