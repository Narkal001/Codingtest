package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

// Currency struct represents the data for a single currency
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

// CurrencyData struct represents the data for all supported currencies
type CurrencyData struct {
	Currencies []Currency `json:"currencies"`
}

// MarketTicker represents the data for a single market ticker
type MarketTicker struct {
	Ask    string `json:"ask"`
	Bid    string `json:"bid"`
	Last   string `json:"last"`
	Open   string `json:"open"`
	Low    string `json:"low"`
	High   string `json:"high"`
	Volume string `json:"volume"`
}

// Markets struct represents the data for all supported markets
type Markets struct {
	Markets map[string]MarketTicker `json:"markets"`
}

// Config struct represents the configuration data for the microservice
type Config struct {
	Symbols []string `json:"symbols"`
}

// CurrencyHandler handles the GET /currency/{symbol} endpoint
func CurrencyHandler(markets *Markets) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the requested symbol and validate it
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "No symbol specified", http.StatusBadRequest)
			return
		}
		if _, ok := markets.Markets[symbol]; !ok {
			http.Error(w, "Invalid symbol specified", http.StatusBadRequest)
			return
		}

		// Get the market ticker for the requested symbol
		marketTicker := markets.Markets[symbol]

		// Create a Currency object with the data from the market ticker
		currency := Currency{
			ID:          symbol,
			FullName:    symbol,
			Ask:         marketTicker.Ask,
			Bid:         marketTicker.Bid,
			Last:        marketTicker.Last,
			Open:        marketTicker.Open,
			Low:         marketTicker.Low,
			High:        marketTicker.High,
			FeeCurrency: "BTC", // Hard-coded for now
		}

		// Convert the Currency object to JSON and write it to the response
		js, err := json.Marshal(currency)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

// CurrenciesHandler handles the GET /currency/all endpoint
func CurrenciesHandler(markets *Markets) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create a slice of Currency objects from the market tickers
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
				FeeCurrency: "BTC", // Hard-coded for now
			}
			currencies = append(currencies, currency)
		}

		// Create a CurrencyData object with the slice of Currency objects
		currencyData := CurrencyData{
			Currencies: currencies,
		}

		// Convert the CurrencyData object to JSON and write it to the response
		js, err := json.Marshal(currencyData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

// getConfig loads the configuration data from the config file and returns a Config object
func getConfig() (*Config, error) {
	// Hard-coded config file for now
	configFile := "config.json"

	// Read the config file
	bytes, err := ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	// Parse the JSON config data into a Config object
	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// getMarkets gets the latest market ticker data for all supported symbols and returns
// a Markets object with the data
func getMarkets(symbols []string) (*Markets, error) {
	// Set the HitBTC API endpoint
	endpoint := "https://api.hitbtc.com/api/2/public/ticker"

	// Make an HTTP request to the API endpoint
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body into a byte slice
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON response into a map of MarketTicker objects
	var marketTickers map[string]MarketTicker
	err = json.Unmarshal(body, &marketTickers)
	if err != nil {
		return nil, err
	}

	// Create a Markets object with the MarketTicker map
	markets := Markets{
		Markets: marketTickers,
	}

	return &markets, nil
}

// updateMarkets periodically updates the market ticker data for all supported symbols and
// updates the Markets object with the new data
func updateMarkets(markets *Markets, symbols []string) {
	// Set the update interval to 10 seconds
	updateInterval := 10 * time.Second

	// Create a ticker to trigger the update at the specified interval
	ticker := time.NewTicker(updateInterval)

	// Use a WaitGroup to block until all goroutines are finished
	var wg sync.WaitGroup

	// Start a goroutine to update the markets on each tick of the ticker
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

	// Wait for the ticker goroutine to finish and mark the WaitGroup as done
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		<-done
	}()

	// Wait for all goroutines to finish
	wg.Wait()
}

func main() {
	// Load the configuration data and get the list of supported symbols
	config, err := getConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v\n", err)
	}
	symbols := config.Symbols

	// Get the initial market ticker data
	markets, err := getMarkets(symbols)
	if err != nil {
		log.Fatalf("Error getting initial markets: %v\n", err)
	}

	// Start the goroutine to update the market ticker data periodically
	go updateMarkets(markets, symbols)

	// Set up the HTTP server and handlers
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

	// Start the server on port 8080
	log.Println("Starting server...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ReadFile reads a file with the specified name and returns its contents as a byte slice
func ReadFile(filename string) ([]byte, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
