package bitfinex

import (
	"log"
	"os"
	"sync"

	api "github.com/santacruz123/bitfinex-api-go"
	"github.com/santacruz123/go-trader/platform"
	"github.com/santacruz123/go-trader/trades"
)

var once sync.Once

var bitfinexKey string
var bitfinexSecret string

var bfxPlatform *bitfinex

type bitfinex struct {
	mu      sync.Mutex
	symbols []*trades.Symbol
	client  *api.Client
}

// Get platform
func Get() platform.Platformer {
	once.Do(func() {

		bfxPlatform = &bitfinex{}

		bfxPlatform.client = api.NewClient().Auth(bitfinexKey, bitfinexSecret)

		err := bfxPlatform.client.WebSocket.Connect()

		bfxPlatform.Symbol("BTCUSD")
		bfxPlatform.Symbol("LTCUSD")

		go bfxPlatform.client.WebSocket.Subscribe()

		if err != nil {
			log.Fatal("Error connecting to bitfinex socket")
		}
	})

	return bfxPlatform
}

func (platform *bitfinex) ClosePlatform() {
	platform.client.WebSocket.Close()
}

func (platform *bitfinex) symbol(s string) *trades.Symbol {
	prices := make(chan trades.Quotes)
	apiPrices := make(chan []float64)

	platform.client.WebSocket.AddSubscribe(api.CHAN_TICKER, s, apiPrices)

	go func() {
		for pack := range apiPrices {
			select {
			case prices <- trades.Quotes{Bid: pack[0], Ask: pack[2]}:
			default:
			}
		}

		log.Println("Bitfinex", s, "channel died")
		close(prices)
	}()

	return trades.NewSymbol(s, trades.Fx, 0.01, prices)
}

func (platform *bitfinex) Symbol(s string) (symbol *trades.Symbol, err error) {
	platform.mu.Lock()
	defer platform.mu.Unlock()

	for _, one := range platform.symbols {
		if one.Symbol() == s {
			return one, nil
		}
	}

	symbol = platform.symbol(s)
	platform.symbols = append(platform.symbols, symbol)

	return
}

func (platform *bitfinex) Orders() (orders trades.Orders, err error) {
	return
}

func (platform *bitfinex) Positions() (positions trades.Positions, err error) {
	bfPositions, err := platform.client.Positions.All()

	for _, one := range bfPositions {

		symbol, err := platform.Symbol(one.Symbol)

		if err != nil {
			return nil, err
		}

		positions = append(positions, trades.Position{
			Symbol: symbol,
			Amount: one.Amount,
			Price:  one.Base,
		})
	}

	return
}

func (platform *bitfinex) Order(o trades.Order) (id uint, err error) {

	orderType := api.ORDER_TYPE_LIMIT

	if o.IsStop {
		orderType = api.ORDER_TYPE_STOP
	}

	var data *api.Order

	data, err = platform.client.Orders.Create(o.Symbol.Symbol(), o.Amount, o.Price, orderType)

	if err == nil {
		return uint(data.Id), nil
	}

	return
}

func (platform *bitfinex) Cancel(id uint) error {
	return platform.client.Orders.Cancel(int(id))
}

func (platform *bitfinex) CancelAll() error {
	return platform.client.Orders.CancelAll()
}

func (platform *bitfinex) Modify(id uint, order trades.Order) (err error) {
	return
}

func init() {

	bitfinexKey = os.Getenv("bitfinex_key")
	bitfinexSecret = os.Getenv("bitfinex_secret")

	if bitfinexKey == "" {
		log.Fatalf("Missing bitfinex_key")
	}

	if bitfinexSecret == "" {
		log.Fatalf("Missing bitfinex_secret")
	}
}
