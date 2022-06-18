package tradingview_test

import (
	"fmt"

	"github.com/elliottcarlson/tradingview"
)

func ExampleNewClient() {
	client := tradingview.NewClient()

	client.Connect()
}

func ExampleTradingView_Connect() {
	client := tradingview.NewClient()

	client.Connect()
}

func ExampleTradingView_Watch() {
	client := tradingview.NewClient()

	client.Connect()

	client.Watch("IBM")
	client.Watch("MSFT")
	client.Watch("AAPL")

	client.Start()
}

func ExampleTradingView_GetQuote() {
	client := tradingview.NewClient()

	client.Connect()

	client.GetQuote("AAPL", func(quote tradingview.Quote) {
		// .. Handle the received quote data.
		fmt.Printf("AAPL last price: %f\n", quote.LastPrice)
	})

	client.Start()
}

func ExampleTradingView_GetLastQuote() {
	client := tradingview.NewClient()

	client.Connect()

	client.Watch("AAPL")

	if quote, ok := client.GetLastQuote("AAPL"); ok {
		// ... Handle the received quote data.
		fmt.Println(quote)
	}

	client.Start()
}

func ExampleTradingView_OnUpdate() {
	client := tradingview.NewClient()

	client.Connect()

	client.OnUpdate("AAPL", func(quote tradingview.Quote) (shouldDelete bool) {
		// ... Handle the received quote data.
		if quote.LastPrice > 500.00 {
			fmt.Println("AAPL is now over $500.00!")
			return true // Stop calling this callback for each update.
		}

		return false // Continue listening for future updates.
	})

	client.Start()
}
