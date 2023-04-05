package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type Currency struct {
	ID           string `json:"id"`
	FullName     string `json:"fullName"`
	Ask          string `json:"ask"`
	Bid          string `json:"bid"`
	Last         string `json:"last"`
	Open         string `json:"open"`
	Low          string `json:"low"`
	High         string `json:"high"`
	FeeCurrency  string `json:"feeCurrency"`
	Volume       string `json:"volume"`
	QuoteVolume  string `json:"quoteVolume"`
	Change       string `json:"change"`
	PercentChage string `json:"percentChange"`
}

type CurrencyData struct {
	Currencies []Currency `json:"currencies"`
}

type MarketTicker struct {
	Ask    string `json:"ask"`
	Bid    string `json:"bid"`
	Last   string `json:"last"`
	Open   string `json:"open"`
	Low    string `json:"low"`
	High   string `json:"high"`
	Volume string `json:"volume"`
}

type Markets struct {
	Markets map[string]MarketTicker `json:"markets"`
}

type Config struct {
	Symbols []string `json:"symbols"`
}

func CurrencyHandler(markets *Markets) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "No symbol specified", http.StatusBadRequest)
			return
		}
		if _, ok := markets.Markets[symbol]; !ok {
			http.Error(w, "Invalid symbol specified", http.StatusBadRequest)
			return
		}

		marketTicker := markets.Markets[symbol]

		currency := Currency{
			ID:          symbol,
			FullName:    symbol,
			Ask:         marketTicker.Ask,
			Bid:         marketTicker.Bid,
			Last:        marketTicker.Last,
			Open:        marketTicker.Open,
			Low:         marketTicker.Low,
			High:        marketTicker.High,
			FeeCurrency: "BTC",
		}

		js, err := json.Marshal(currency)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

func CurrenciesHandler(markets *Markets) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var currencies []Currency
		for symbol, marketTicker := range markets.Markets {
			currency := Currency{
				ID:          symbol,
				FullName:    symbol,
				Ask:         marketTicker.Ask,
				Bid:         marketTicker.Bid,
				Last:        marketTicker.Last,
				Open:        marketTicker.Open,
				Low:         marketTicker.Low,
				High:        marketTicker.High,
				FeeCurrency: "BTC",
			}
			currencies = append(currencies, currency)
		}

		currencyData := CurrencyData{
			Currencies: currencies,
		}

		js, err := json.Marshal(currencyData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

func getConfig() (*Config, error) {
	configFile := "config.json"

	bytes, err := ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func getMarkets(symbols []string) (*Markets, error) {
	endpoint := "https://api.hitbtc.com/api/2/public/ticker"

	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var marketTickers map[string]MarketTicker
	err = json.Unmarshal(body, &marketTickers)
	if err != nil {
		return nil, err
	}

	markets := Markets{
		Markets: marketTickers,
	}

	return &markets, nil
}

func updateMarkets(markets *Markets, symbols []string) {
	updateInterval := 10 * time.Second

	ticker := time.NewTicker(updateInterval)

	var wg sync.WaitGroup

	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				log.Println("Updating markets...")
				newMarkets, err := getMarkets(symbols)
				if err != nil {
					log.Printf("Error updating markets: %v\n", err)
				} else {
					*markets = *newMarkets
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		<-done
	}()

	wg.Wait()
}

func main() {
	config, err := getConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v\n", err)
	}
	symbols := config.Symbols

	markets, err := getMarkets(symbols)
	if err != nil {
		log.Fatalf("Error getting initial markets: %v\n", err)
	}

	go updateMarkets(markets, symbols)

	http.HandleFunc("/currency", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/currency/all", http.StatusMovedPermanently)
	})
	http.HandleFunc("/currency/", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Path[len("/currency/"):]
		if symbol != "all" {
			r.URL.Query().Set("symbol", symbol)
			CurrencyHandler(markets)(w, r)
		} else {
			CurrenciesHandler(markets)(w, r)
		}
	})

	log.Println("Starting server...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func ReadFile(filename string) ([]byte, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
