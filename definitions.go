package tradingview

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type TradingView struct {
	conn           *websocket.Conn
	dialer         *websocket.Dialer
	url            string
	requestHeader  http.Header
	OnConnected    func(tv TradingView)
	OnConnectError func(err error, tv TradingView)
	OnDisconnected func(err error, tv TradingView)
	Watching       map[string]Quote
	notifications  []*notifications
	IsConnected    bool
	sessionID      string
	sendMutex      *sync.Mutex
	recvMutex      *sync.Mutex
}

type Quote struct {
	Symbol               string  `json:"short_name"`
	FullName             string  `json:"description"`
	CurrencyCode         string  `json:"currency_code"`
	IsTradable           bool    `json:"is_tradable"`
	Exchange             string  `json:"listed_exchange"`
	OriginalName         string  `json:"original_name"`
	ProName              string  `json:"pro_name"`
	CurrentSession       string  `json:"current_session"`
	LastPrice            float64 `json:"lp"`
	Change               float64 `json:"ch"`
	ChangePercentage     float64 `json:"chp"`
	LivePrice            float64 `json:"rtc"`
	LiveChange           float64 `json:"rch"`
	LiveChangePercentage float64 `json:"rchp"`
}

type notifications struct {
	Symbol string
	Action func(Quote) bool
}

type tvrequest struct {
	Method string        `json:"m"`
	Params []interface{} `json:"p"`
}

type tvevent struct {
	Type      string            `json:"m"`
	RawData   []json.RawMessage `json:"p"`
	SessionID string
	Data      interface{}
}

type tvqsdenvelope struct {
	Symbol string          `json:"n"`
	Status string          `json:"s"`
	Data   json.RawMessage `json:"v"`
}
