package stoploss

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

// Parent struct embedded in each exchange struct
type Market struct {
	marketCache MarketCache
}

type MarketCache struct {
	Order []Order `yaml:"order"`
}

type Order struct {
	OrderId   string    `yaml:"order_id"`
	OrderType string    `yaml:"order_type"`
	Exchange  string    `yaml:"exchange"`
	Pair      string    `yaml:"pair"`
	Quantity  string    `yaml:"quantity"`
	Price     float64   `yaml:"price"`
	Updated   time.Time `yaml:"updated"`
}

func (m *Market) LoadCacheLimitOrders(logger *log.Logger, cachePath string, fileMutex *sync.Mutex) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// check if cache file exists
	_, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		return
	}

	yamlFile, err := ioutil.ReadFile(cachePath)
	if err != nil {
		logger.Printf("Error reading YAML file: %s\n", err)
	}

	marketCache := MarketCache{}
	err = yaml.Unmarshal(yamlFile, &marketCache)
	if err != nil {
		logger.Printf("Error parsing YAML file: %s\n", err)
	}

	m.marketCache = marketCache

	logger.Printf("Loading the market cache: %+v\n", marketCache)
}

func (m *Market) GetCacheLimitOrderId(logger *log.Logger, exchange string, orderType string, baseCoin string, countCoin string, quantity string, price float64) (string, string, error) {
	marketCache := m.marketCache
	for _, order := range marketCache.Order {
		market := fmt.Sprintf("%s/%s", baseCoin, countCoin)
		if (order.Pair == market) && (order.Exchange == exchange) && (order.OrderType == orderType) {
			if order.Price == price && order.Quantity == quantity {
				//logger.Printf("Found an order in the market cache: %s\n", order.OrderId)
				return order.OrderId, order.OrderId, nil
			} else {
				logger.Printf("Replace an order from the market cache - the price and quantity didn't match the config: %s\n", order.OrderId)
				return order.OrderId, "replace", nil
			}
		}
	}
	return "", "", nil
}

func (m *Market) GetCacheLimitOrderQuantity(exchange string, orderType string, baseCoin string, countCoin string) (string, error) {
	marketCache := m.marketCache
	for _, order := range marketCache.Order {
		market := fmt.Sprintf("%s/%s", baseCoin, countCoin)
		if (order.Pair == market) && (order.Exchange == exchange) && (order.OrderType == orderType) {
			return order.Quantity, nil
		}
	}
	return "", nil
}

//func (m *Market) GetCacheLimitOrderId2(logger *log.Logger, exchange string, orderType string, baseCoin string, countCoin string, quantity string, price float64) (string, string, error) {
//	marketCache := m.marketCache
//	for _, order := range marketCache.Order {
//		market := fmt.Sprintf("%s/%s", baseCoin, countCoin)
//		if (order.Pair == market) && (order.Exchange == exchange) && (order.OrderType == orderType) {
//			if order.Price == price && order.Quantity == quantity {
//				//logger.Printf("Found an order in the market cache: %s\n", order.OrderId)
//				return order.OrderId, order.OrderId, nil
//			} else {
//				logger.Printf("Replace an order from the market cache - the price and quantity didn't match the config: %s\n", order.OrderId)
//				return order.OrderId, "replace", nil
//			}
//		}
//	}
//	return "", "", nil
//
//}

func (m *Market) SetCacheLimitOrder(logger *log.Logger, cachePath string, fileMutex *sync.Mutex, exchange string, orderType string, baseCoin string, countCoin string, quantity string, price float64, id string) error {
	fileCache := MarketCache{}

	logger.Printf("Setting Cache: %+s\n", id)

	fileMutex.Lock()
	defer fileMutex.Unlock()

	// check if cache file exists
	_, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		fmt.Println("Creating cache file:", cachePath)
	} else {
		// read file
		yamlFile, err := ioutil.ReadFile(cachePath)
		if err != nil {
			return err
		}

		// unmarshal YAML
		err = yaml.Unmarshal(yamlFile, &fileCache)
		if err != nil {
			return err
		}
	}

	status := "not found"
	for i, order := range fileCache.Order {
		market := fmt.Sprintf("%s/%s", baseCoin, countCoin)
		if (order.Pair == market) && (order.Exchange == exchange) && (order.OrderType == orderType) {
			status = "found"
			if order.Price == price && order.Quantity == quantity && order.OrderId == id {
				logger.Printf("Found an order match in the market cache: %s\n", order.OrderId)
				return nil
			} else {
				//logger.Printf("price: %f %f", order.Price, price)
				//logger.Printf("quantity: %f %f", order.Quantity, quantity)
				//logger.Printf("quantity: %f %f", order.OrderId, id)
				fileCache.Order[i].Price = price
				fileCache.Order[i].Quantity = quantity
				fileCache.Order[i].OrderId = id
				logger.Printf("Found an order in the market cache, but the price and quantity didn't match, updating: %s\n", order.OrderId)
			}
		}
	}

	// add it if it wasn't found
	if status != "found" {
		market := fmt.Sprintf("%s/%s", baseCoin, countCoin)
		logger.Printf("Adding new %s order to the %s market cache: %s\n", market, exchange, id)

		fileCache.Order = append(fileCache.Order,
			Order{
				OrderId:   id,
				OrderType: orderType,
				Exchange:  exchange,
				Pair:      market,
				Quantity:  quantity,
				Price:     price,
				Updated:   time.Now(),
			},
		)
	}

	// re-encode YAML
	var cacheBuffer bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&cacheBuffer)
	yamlEncoder.SetIndent(2)

	err = yamlEncoder.Encode(&fileCache)
	if err != nil {
		return err
	}

	// write file
	err = ioutil.WriteFile(cachePath, cacheBuffer.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}
