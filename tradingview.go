/*
Package go-tradingview provides an interface to watch and action on updates of Stock symbols
via the TradingView WebSocket interface.
*/
package tradingview

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// Return a new instance of a TradingView.
func NewClient() TradingView {
	return TradingView{
		url: "wss://data.tradingview.com/socket.io/websocket",
		requestHeader: http.Header{
			"Origin": []string{
				"https://data.tradingview.com/",
			},
		},
		dialer:    &websocket.Dialer{},
		sendMutex: &sync.Mutex{},
		recvMutex: &sync.Mutex{},
		Watching:  make(map[string]Quote),
	}
}

// Connect the current instance of TradingView to the TradingView WebSocket.
func (tv *TradingView) Connect() {
	var err error
	var resp *http.Response

	tv.dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	tv.conn, resp, err = tv.dialer.Dial(tv.url, tv.requestHeader)

	if err != nil {
		log.Errorf("Error while connecting to TradingView: %v", err)
		if resp != nil {
			log.Errorf("HTTP Response %d status: %s", resp.StatusCode, resp.Status)
		}
		tv.IsConnected = false
		if tv.OnConnectError != nil {
			tv.OnConnectError(err, *tv)
		}
		return
	}

	log.Info("Connected to TradingView")

	tv.sessionID = createSessionID("qs_")

	tv.send("set_data_quality", []interface{}{"low"})
	tv.send("set_auth_token", []interface{}{"unauthorized_user_token"})
	tv.send("quote_create_session", []interface{}{tv.sessionID})
	tv.send("quote_set_fields", []interface{}{tv.sessionID, "listed_exchange",
		"ch", "chp", "rtc", "rch", "rchp", "lp", "is_tradable",
		"short_name", "description", "currency_code", "current_session",
		"status", "type", "update_mode", "fundamentals", "pro_name",
		"original_name",
	})

	tv.IsConnected = true
	if tv.OnConnected != nil {
		tv.OnConnected(*tv)
	}

	defaultCloseHandler := tv.conn.CloseHandler()
	tv.conn.SetCloseHandler(func(code int, text string) error {
		result := defaultCloseHandler(code, text)
		log.Info("Disconnected from server", result)
		if tv.OnDisconnected != nil {
			tv.OnDisconnected(result, *tv)
		}
		return result
	})

	go tv.loop()
}

func (tv *TradingView) loop() {
	for {
		tv.recvMutex.Lock()
		_, message, err := tv.conn.ReadMessage()
		tv.recvMutex.Unlock()

		if err != nil {
			if tv.OnDisconnected != nil {
				tv.OnDisconnected(err, *tv)
			} else {
				panic("Lost connection; panic to restart")
			}
		}

		tv.messageHandler(string(message))
	}
}

func (tv *TradingView) messageHandler(message string) {
	re := regexp.MustCompile("~m~[0-9]+~m~")
	lines := re.Split(message, -1)

	for i := range lines {
		if lines[i] != "" {
			if matched := re.MatchString(lines[i]); matched {
				tv.sendSigned(lines[i])
				continue
			}

			if err := tv.parseTradingViewEvent(lines[i]); err != nil {
				log.Errorf("Error parsing incoming message: %v", err)
			}
		}
	}
}

func (tv *TradingView) parseTradingViewEvent(message string) (err error) {
	event := &tvevent{}
	err = json.Unmarshal([]byte(message), event)
	if err != nil {
		return err
	}

	if event.Type == "" {
		return nil
	}

	event.SessionID = string(event.RawData[0])

	switch event.Type {
	case "qsd":
		if len(event.RawData) != 2 {
			return fmt.Errorf("unrecognized QSD event message format")
		}

		envelope := &tvqsdenvelope{}
		err = json.Unmarshal([]byte(event.RawData[1]), envelope)
		if err != nil {
			return err
		}

		symbol := envelope.Symbol
		re := regexp.MustCompile(`([A-Z]+)$`)
		parsed := re.FindStringSubmatch(symbol)
		if len(parsed) == 2 {
			symbol = parsed[0]
		}

		var qsd Quote
		if quote, ok := tv.GetLastQuote(symbol); ok {
			qsd = quote
		}

		err = json.Unmarshal([]byte(envelope.Data), &qsd)
		if err != nil {
			return fmt.Errorf("error parsing quote data: %v", err)
		}

		log.Debugf("QSD line %v", message)

		if qsd.OriginalName != "" {
			if _, ok := tv.Watching[qsd.OriginalName]; !ok {
				tv.Watch(qsd.OriginalName)
			}
		}

		if qsd.ProName != "" {
			if _, ok := tv.Watching[qsd.ProName]; !ok {
				tv.Watch(qsd.ProName)
			}
		}

		tv.Update(symbol, qsd)
	default:
		log.Infof("Unknown TV payload: %v", message)
		return nil
	}

	return nil
}

func (tv *TradingView) send(method string, params []interface{}) {
	data := tvrequest{
		Method: method,
		Params: params,
	}

	message, err := json.Marshal(data)

	if err != nil {
		log.Errorf("Error creating Signed Message: %v", err)
		return
	}

	tv.sendSigned(string(message))
}

func (tv *TradingView) sendSigned(message string) {
	message = fmt.Sprintf("~m~%d~m~%s", len(message), message)

	err := tv.sendRaw(message)
	if err != nil {
		log.Errorf("Error sending message: %v", err)
		return
	}
}

func (tv *TradingView) sendRaw(message string) error {
	tv.sendMutex.Lock()
	err := tv.conn.WriteMessage(websocket.TextMessage, []byte(message))
	tv.sendMutex.Unlock()
	return err
}

// Add the specified symbol to the TradingView watch list.
func (tv *TradingView) Watch(symbol string) {
	if _, ok := tv.Watching[symbol]; ok {
		return
	} else {
		tv.Watching[symbol] = Quote{}
	}

	if tv.IsConnected {
		tv.send("quote_add_symbols", []interface{}{
			tv.sessionID,
			symbol,
			map[string][]string{
				"flags": {
					"force_permission",
				},
			},
		})
		tv.send("quote_fast_symbols", []interface{}{
			tv.sessionID,
			symbol,
		})
	}
}

// Send an updated quote for the specified symbol.
// Sends the provided TradingViewQuote to all callbacks listening for
// updates on the specified symbol. This method is useful for testing
// or replaying quotes, but does not provide functionality that
// changes the interaction with the TradingView WebSocket.
func (tv *TradingView) Update(symbol string, quote Quote) {
	tv.Watching[symbol] = quote

	temp := tv.notifications[:0]
	for i := range tv.notifications {
		notification := tv.notifications[i]

		if notification.Symbol == symbol {
			shouldDelete := notification.Action(quote)

			if shouldDelete {
				continue
			}
		}

		temp = append(temp, notification)
	}

	tv.notifications = temp
}

// Retrieve the specified symbols quote, with a callback for when the quote resolves.
// If the symbol is already being watched, it will call the provided callback with the
// last cached version of the symbol. If the symbol is not being watched yet, it will
// If the symbol is not in the existing watch list, it will add it, and queue up a
// response to the callback once it has a quote to provide.
// The callback should expect to be called once, with a TradingViewQuote containing the
// latest quote for the specified symbol.
func (tv *TradingView) GetQuote(symbol string, callback func(Quote)) {
	if _, ok := tv.Watching[symbol]; ok {
		callback(tv.Watching[symbol])
		return
	}

	tv.OnUpdate(symbol, func(quote Quote) (shouldDelete bool) {
		callback(quote)
		return true
	})
}

// Retrieve the most recent quote for the specified symbol.
// If the symbol is already being watched, it will return the latest quote available,
// and a true value to indicate this request was successful.
// If the symbol is not watched, it will return an empty TradingViewQuote struct, and
// a false value to indicate the quote is not being watched.
func (tv *TradingView) GetLastQuote(symbol string) (quote Quote, ok bool) {
	if _, ok := tv.Watching[symbol]; ok {
		return tv.Watching[symbol], true
	}

	return Quote{}, false
}

// Create a notification request for all updates to the specified symbol which will
// call the specified callback for each update event.
// The callback should expect a TradingViewQuote for each updated quote value it
// receives, until the callback returns true to indicate it should stop receiving
// updates. Returning false in the callback will indicate that it should continue
// to notify on updates for the specified symbol.
func (tv *TradingView) OnUpdate(symbol string, callback func(Quote) (shouldDelete bool)) {
	if _, ok := tv.Watching[symbol]; !ok {
		tv.Watch(symbol)
	}

	notification := &notifications{
		Symbol: symbol,
		Action: callback,
	}

	tv.notifications = append(tv.notifications, notification)
}

func createSessionID(prefix string) string {
	rand.Seed(time.Now().UnixNano())
	var runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, 12)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}

	return prefix + string(b)
}
