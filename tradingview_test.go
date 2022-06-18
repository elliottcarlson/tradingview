package tradingview_test

import (
	"testing"

	"github.com/elliottcarlson/tradingview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCallback struct {
	mock.Mock
	tradingview.TradingView
}

func (m *MockCallback) GetQuoteCallback(quote tradingview.Quote) {
	m.Called(quote)
}

func (m *MockCallback) OnUpdateCallbackReturnTrue(quote tradingview.Quote) (shouldDelete bool) {
	m.Called(quote)
	return true
}

func (m *MockCallback) OnUpdateCallbackReturnFalse(quote tradingview.Quote) (shouldDelete bool) {
	m.Called(quote)
	return false
}

func TestNewTradingView(t *testing.T) {
	client := tradingview.NewClient()

	assert.IsType(t, client, tradingview.TradingView{})
}

func TestTradingView_Watch(t *testing.T) {
	client := tradingview.NewClient()

	client.Watch("AAPL")

	assert.Contains(t, client.Watching, "AAPL")
}

func TestTradingView_GetQuote(t *testing.T) {
	client := tradingview.NewClient()

	callback := &MockCallback{}
	callback.On("GetQuoteCallback", mock.AnythingOfType("tradingview.Quote")).Return(nil)

	client.GetQuote("AAPL", callback.GetQuoteCallback)
	client.Update("AAPL", tradingview.Quote{
		Symbol: "AAPL",
	})

	callback.AssertExpectations(t)
	callback.AssertCalled(t, "GetQuoteCallback", tradingview.Quote{
		Symbol: "AAPL",
	})
}

func TestTradingView_GetLastQuote(t *testing.T) {
	client := tradingview.NewClient()

	client.Update("MSFT", tradingview.Quote{
		Symbol:    "MSFT",
		LastPrice: 123.45,
	})

	quote, ok := client.GetLastQuote("MSFT")
	assert.Equal(t, 123.45, quote.LastPrice)
	assert.True(t, ok)

	quote, ok = client.GetLastQuote("AAPL")
	assert.Equal(t, quote, tradingview.Quote{})
	assert.False(t, ok)
}

func TestTradingView_OnUpdate(t *testing.T) {
	client := tradingview.NewClient()

	callback_true := &MockCallback{}
	callback_true.On("OnUpdateCallbackReturnTrue", mock.AnythingOfType("tradingview.Quote")).Return(nil)

	callback_false := &MockCallback{}
	callback_false.On("OnUpdateCallbackReturnFalse", mock.AnythingOfType("tradingview.Quote")).Return(nil)

	client.OnUpdate("AAPL", callback_true.OnUpdateCallbackReturnTrue)
	client.OnUpdate("AAPL", callback_false.OnUpdateCallbackReturnFalse)

	client.Update("AAPL", tradingview.Quote{
		Symbol:    "AAPL",
		LastPrice: 123.45,
	})
	client.Update("AAPL", tradingview.Quote{
		Symbol:    "AAPL",
		LastPrice: 124.56,
	})
	client.Update("AAPL", tradingview.Quote{
		Symbol:    "AAPL",
		LastPrice: 125.67,
	})

	callback_true.AssertExpectations(t)
	callback_true.AssertNumberOfCalls(t, "OnUpdateCallbackReturnTrue", 1)
	callback_true.AssertCalled(t, "OnUpdateCallbackReturnTrue", tradingview.Quote{
		Symbol:    "AAPL",
		LastPrice: 123.45,
	})
	callback_true.AssertNotCalled(t, "OnUpdateCallbackReturnTrue", tradingview.Quote{
		Symbol:    "AAPL",
		LastPrice: 124.56,
	})
	callback_true.AssertNotCalled(t, "OnUpdateCallbackReturnTrue", tradingview.Quote{
		Symbol:    "AAPL",
		LastPrice: 125.67,
	})

	callback_false.AssertExpectations(t)
	callback_false.AssertNumberOfCalls(t, "OnUpdateCallbackReturnFalse", 3)
	callback_false.AssertCalled(t, "OnUpdateCallbackReturnFalse", tradingview.Quote{
		Symbol:    "AAPL",
		LastPrice: 123.45,
	})
	callback_false.AssertCalled(t, "OnUpdateCallbackReturnFalse", tradingview.Quote{
		Symbol:    "AAPL",
		LastPrice: 124.56,
	})
	callback_false.AssertCalled(t, "OnUpdateCallbackReturnFalse", tradingview.Quote{
		Symbol:    "AAPL",
		LastPrice: 125.67,
	})
}
