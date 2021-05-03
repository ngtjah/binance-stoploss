package stoploss

import (
	"bytes"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"sync"
)

func (tlg *Trailing) loadCache(cachePath string, fileMutex *sync.Mutex) {
	tlg.cachePath = cachePath

	fileMutex.Lock()

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

	tlg.cache = cache

	defer fileMutex.Unlock()
}

func (tlg *Trailing) SaveCache(fileMutex *sync.Mutex) {
	cachePath := ".cache.yaml"

	//tlg.logger.Printf("Setting Cache: %+v\n", tlg.cache)

	var cache bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&cache)
	yamlEncoder.SetIndent(2)

	err := yamlEncoder.Encode(&tlg.cache)
	if err != nil {
		log.Fatalf("Error encoding YAML: %v", err)
	}

	fileMutex.Lock()

	err = ioutil.WriteFile(cachePath, cache.Bytes(), 0644)
	if err != nil {
		log.Fatalf("Error writing file: %v", err)
	}

	defer fileMutex.Unlock()
}
