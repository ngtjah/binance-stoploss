package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/preichenberger/go-coinbasepro/v2"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	binance "github.com/adshao/go-binance/v2"
	"github.com/giansalex/binance-stoploss/stoploss"
	"github.com/hashicorp/go-retryablehttp"
)

var (
	typePtr             = flag.String("type", "SELL", "order type: SELL or BUY")
	pairPtr             = flag.String("pair", "", "market pair, example: BNB/USDT")
	startPercentPtr     = flag.Float64("start.percent", 0.00, "min percent (for trailing stop loss), example: 1.0 (1%)")
	endPercentPtr       = flag.Float64("end.percent", 0.00, "max percent (for trailing stop loss), example: 3.0 (3%)")
	maxLossPercentPtr   = flag.Float64("max.loss.percent", 0.00, "max percent (for total stop loss), example: 1.0 (1%)")
	buyPricePtr         = flag.Float64("buy.price", 0.00, "max percent (for total stop loss), example: 1.0 (1%)")
	limitSellPercentPtr = flag.Float64("sell.limit.percent", 0.00, "max percent gain for a limit sell order, example: 1.0 (1%)")
	pricePtr            = flag.Float64("price", 0.00, "price (for static stop loss), example: 9200.00 (BTC price)")
	intervalPtr         = flag.Int("interval", 10, "interval in seconds to update price, example: 10 (10 sec.)")
	amountPtr           = flag.String("amount", "", "(optional) amount to order (sell or buy) on stoploss")
	chatPtr             = flag.Int64("telegram.chat", 0, "(optional) telegram User ID for notify")
	cfgPtr              = flag.String("cfg", "", "Enable config file mode")
	binanceUsPtr        = flag.Bool("binance.us", false, "Switch to the binance.us api")
)

type Config struct {
	Coins struct {
		Sell []Sell `yaml:"sell"`
	} `yaml:"coins"`
}

type Sell struct {
	Pair             string  `yaml:"pair"`
	StartPercent     float64 `yaml:"start_percent"`
	EndPercent       float64 `yaml:"end_percent"`
	Amount           string  `yaml:"amount"`
	Price            float64 `yaml:"price"`
	Exchange         string  `yaml:"exchange"`
	BuyPrice         float64 `yaml:"buy_price"`
	MaxLossPercent   float64 `yaml:"max_loss_percent"`
	LimitSellPercent float64 `yaml:"limit_sell_percent"`
}

func main() {
	flag.Parse()

	binanceApiKey := os.Getenv("BINANCE_APIKEY")
	binanceSecret := os.Getenv("BINANCE_SECRET")

	coinbaseKey := os.Getenv("COINBASEPRO_KEY")
	coinbaseSecret := os.Getenv("COINBASEPRO_SECRET")
	coinbasePassphrase := os.Getenv("COINBASEPRO_PASSPHRASE")

	chatNotify := os.Getenv("CHAT_NOTIFY")

	if binanceApiKey == "" || binanceSecret == "" {
		log.Fatal("BINANCE_APIKEY, BINANCE_SECRET are required")
	}

	if coinbaseKey == "" || coinbaseSecret == "" || coinbasePassphrase == "" {
		log.Fatal("COINBASEPRO_KEY, COINBASEPRO_SECRET, COINBASEPRO_PASSPHRASE are required")
	}

	if cfgPtr == nil || *cfgPtr == "" {
		if pairPtr == nil || *pairPtr == "" {
			log.Fatal("pair market is required")
		}

		if (startPercentPtr == nil || *startPercentPtr <= 0) && (endPercentPtr == nil || *endPercentPtr <= 0) && (pricePtr == nil || *pricePtr <= 0) {
			log.Fatal("a price or percent parameter is required")
		}
	}

	logger := log.New(os.Stdout, "", 0)

	binanceHttpClient := retryablehttp.NewClient()
	binanceHttpClient.Logger = nil
	binanceHttpClient.RetryMax = 10

	binanceApi := binance.NewClient(binanceApiKey, binanceSecret)
	binanceApi.HTTPClient = binanceHttpClient.StandardClient()

	if *binanceUsPtr {
		logger.Println("Configuring for the binance.us api")
		binanceApi.BaseURL = "https://api.binance.us"
	}

	coinbaseHttpClient := retryablehttp.NewClient()
	coinbaseHttpClient.Logger = nil
	coinbaseHttpClient.RetryMax = 10

	coinbaseApi := coinbasepro.NewClient()
	coinbaseApi.HTTPClient = coinbaseHttpClient.StandardClient()

	coinbaseApi.UpdateConfig(&coinbasepro.ClientConfig{
		BaseURL:    "https://api.pro.coinbase.com",
		Key:        coinbaseKey,
		Secret:     coinbaseSecret,
		Passphrase: coinbasePassphrase,
	})

	var notify stoploss.Notify

	telegramToken, telegramTokenExists := os.LookupEnv("TELEGRAM_TOKEN")
	if telegramTokenExists {
		notify = stoploss.NewTelegram(telegramToken, *chatPtr, logger)
	} else {
		notify = stoploss.NewSlack(os.Getenv("SLACK_WEBHOOK"), logger)
	}

	var fileMutex sync.Mutex

	if cfgPtr == nil || *cfgPtr == "" {
		trailing := stoploss.NewTrailing(
			stoploss.NewBinanceExchange(context.Background(), binanceApi),
			notify,
			*typePtr,
			*pairPtr,
			*startPercentPtr/100,
			*endPercentPtr/100,
			*maxLossPercentPtr/100,
			*limitSellPercentPtr/100,
			*buyPricePtr,
			*amountPtr,
			*pricePtr,
			logger,
			&fileMutex,
			chatNotify,
		)

		for {
			if trailing.RunStop() {
				break
			}

			time.Sleep(time.Duration(*intervalPtr) * time.Second)
		}

		// Loop over the config
	} else {
		yamlFile, err := ioutil.ReadFile(*cfgPtr)
		if err != nil {
			fmt.Printf("Error reading YAML file: %s\n", err)
		}

		var config Config
		err = yaml.Unmarshal(yamlFile, &config)
		if err != nil {
			fmt.Printf("Error parsing YAML file: %s\n", err)
		}

		fmt.Printf("Config: %+v\n", config)

		var wg sync.WaitGroup

		for _, v := range config.Coins.Sell {
			wg.Add(1)

			go func(wg *sync.WaitGroup, v Sell) {
				defer wg.Done()

				var exchange stoploss.Exchange
				if v.Exchange == "binance" {
					exchange = stoploss.NewBinanceExchange(context.Background(), binanceApi)
				} else if v.Exchange == "coinbase" {
					//exchange = stoploss.NewCoinbaseExchange(context.Background(), coinbaseApi)
				} else {
					logger.Fatal("Unrecognized exchange in config")
				}

				trailing := stoploss.NewTrailing(
					exchange,
					notify,
					"SELL",
					v.Pair,
					v.StartPercent/100,
					v.EndPercent/100,
					v.MaxLossPercent/100,
					v.LimitSellPercent/100,
					v.BuyPrice,
					v.Amount,
					v.Price,
					logger,
					&fileMutex,
					chatNotify,
				)

				//fileMutex.Lock()
				//trailing.Chart(v.Exchange, v.Pair)
				//fileMutex.Unlock()

				for {
					if trailing.RunStop() {
						break
					}

					time.Sleep(time.Duration(*intervalPtr) * time.Second)
				}
			}(&wg, v)
		}
		wg.Wait()
	}

}
