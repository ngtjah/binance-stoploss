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

type Cache struct {
	Sell []Sell `yaml:"sell"`
}

type Sell struct {
	Pair       string    `yaml:"pair"`
	LastStop   float64   `yaml:"lastStop"`
	Exchange   string    `yaml:"exchange"`
	Percentage float64   `yaml:"percentage"`
	Amount     string    `yaml:"amount"`
	Updated    time.Time `yaml:"updated"`
	BuyPrice   float64   `yaml:"buy_price"`
	BuyPercent float64   `yaml:"buy_percent"`
}

func (tlg *Trailing) loadCache(cachePath string, fileMutex *sync.Mutex) {
	tlg.cachePath = cachePath

	fileMutex.Lock()
	defer fileMutex.Unlock()

	// check if cache file exists
	_, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		return
	}

	yamlFile, err := ioutil.ReadFile(cachePath)
	if err != nil {
		tlg.logger.Printf("Error reading YAML file: %s\n", err)
	}

	cache := Cache{}
	err = yaml.Unmarshal(yamlFile, &cache)
	if err != nil {
		tlg.logger.Printf("Error parsing YAML file: %s\n", err)
	}

	//tlg.logger.Printf("Loading the cache: %+v\n", cache)

	for _, order := range cache.Sell {
		market := fmt.Sprintf("%s/%s", tlg.baseCoin, tlg.countCoin)
		if (order.Pair == market) && (order.Exchange == tlg.exchange.Name()) {
			// make sure these other values haven't changed and an hour hasn't past since the last update
			if tlg.ValidateCache(order) {
				tlg.lastStopCache = order.LastStop
				//tlg.logger.Printf("Setting %s LastStop in tlg.lastStopCache: %f\n", v.Pair, v.LastStop)
			} else {
				// expired
				//tlg.lastStop = 0
				//tlg.lastStopCache = 0
			}
		}
	}
}

func (tlg *Trailing) ValidateCache(order Sell) bool {
	return (order.Amount == tlg.quantity && order.Percentage == tlg.stopFactor*100 &&
		order.BuyPrice == tlg.buyPrice && order.BuyPercent == tlg.buyStopFactor*100) &&
		(time.Since(order.Updated).Minutes() < 60)
}

func (tlg *Trailing) SaveCache(fileMutex *sync.Mutex) {
	cachePath := ".cache.yaml"
	fileCache := Cache{}

	//tlg.logger.Printf("Setting Cache: %+v\n", tlg.cache)

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
			tlg.logger.Printf("Error reading YAML file: %s\n", err)
		}

		// unmarshal YAML
		err = yaml.Unmarshal(yamlFile, &fileCache)
		if err != nil {
			tlg.logger.Printf("Error parsing YAML file: %s\n", err)
		}
	}

	// see if we need to update the existing value
	status := "not found"
	for i, order := range fileCache.Sell {
		market := fmt.Sprintf("%s/%s", tlg.baseCoin, tlg.countCoin)
		if (order.Pair == market) && (order.Exchange == tlg.exchange.Name()) {
			// make sure these other values haven't changed and an hour hasn't past since the last update
			if tlg.ValidateCache(order) {
				status = "found"
				fileCache.Sell[i].LastStop = tlg.lastStop
				fileCache.Sell[i].Percentage = tlg.stopFactor * 100
				fileCache.Sell[i].Amount = tlg.quantity
				fileCache.Sell[i].Updated = time.Now()
				fileCache.Sell[i].BuyPrice = tlg.buyPrice
				fileCache.Sell[i].BuyPercent = tlg.buyStopFactor * 100
				//tlg.logger.Printf("Setting %s LastStop in tlg.lastStopCache: %f\n", v.Pair, v.LastStop)
			} else {
				status = "expired" // move expired to load time
				//tlg.lastStop = 0
				//tlg.lastStopCache = 0
				// remove the expired or inconsistent cache item and let it rebuild
				fileCache.Sell = append(fileCache.Sell[:i], fileCache.Sell[i+1:]...)
				tlg.logger.Printf("Removing expired or inconsistent %s cache item: %s\n", order.Exchange, order.Pair)
			}
		}
	}

	// add it if it wasn't found
	if status != "found" {
		market := fmt.Sprintf("%s/%s", tlg.baseCoin, tlg.countCoin)

		fileCache.Sell = append(fileCache.Sell,
			Sell{
				Pair:       market,
				LastStop:   tlg.lastStop,
				Exchange:   tlg.exchange.Name(),
				Percentage: tlg.stopFactor * 100,
				Amount:     tlg.quantity,
				BuyPrice:   tlg.buyPrice,
				BuyPercent: tlg.buyStopFactor * 100,
				Updated:    time.Now(),
			},
		)
	}

	// re-encode YAML
	var cacheBuffer bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&cacheBuffer)
	yamlEncoder.SetIndent(2)

	err = yamlEncoder.Encode(&fileCache)
	if err != nil {
		log.Fatalf("Error encoding YAML: %v", err)
	}

	// write file
	err = ioutil.WriteFile(cachePath, cacheBuffer.Bytes(), 0644)
	if err != nil {
		log.Fatalf("Error writing file: %v", err)
	}
}
