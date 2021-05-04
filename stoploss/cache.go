package stoploss

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

type Cache struct {
	Sell []Sell `yaml:"sell"`
}

type Sell struct {
	Pair     string  `yaml:"pair"`
	LastStop float64 `yaml:"lastStop"`
	Exchange string  `yaml:"exchange"`
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

	for _, v := range cache.Sell {
		if v.Pair == fmt.Sprintf("%s/%s", tlg.baseCoin, tlg.countCoin) {
			tlg.lastStopCache = v.LastStop
			//tlg.logger.Printf("Setting %s LastStop in tlg.lastStopCache: %f\n", v.Pair, v.LastStop)
		}
	}
}

func (tlg *Trailing) SaveCache(fileMutex *sync.Mutex) {
	cachePath := ".cache.yaml"

	//tlg.logger.Printf("Setting Cache: %+v\n", tlg.cache)

	fileCache := Cache{}

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
	found := false
	for i, v := range fileCache.Sell {
		market := fmt.Sprintf("%s/%s", tlg.baseCoin, tlg.countCoin)
		if (v.Pair == market) && (v.Exchange == tlg.exchange.Name()) {
			found = true
			fileCache.Sell[i].LastStop = tlg.lastStop
			fileCache.Sell[i].Exchange = tlg.exchange.Name()
			//tlg.logger.Printf("Setting %s LastStop in tlg.lastStopCache: %f\n", v.Pair, v.LastStop)
		}
	}

	// add it if it wasn't found
	if !found {
		market := fmt.Sprintf("%s/%s", tlg.baseCoin, tlg.countCoin)
		fileCache.Sell = append(fileCache.Sell, Sell{Pair: market, LastStop: tlg.lastStop, Exchange: tlg.exchange.Name()})
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
