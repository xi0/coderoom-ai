package browser

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"syscall/js"
	"time"

	"github.com/xi0/coderoom-ai/internal/wire"
)

type WebSocket struct {
	ws          js.Value
	Open        bool
	lastPingSeq *atomic.Int64
}

func NewWebSocket() *WebSocket {
	location := Document().Location()
	protocol := strings.Replace(location.Protocol, "http", "ws", 1)
	url := fmt.Sprintf("%s//%s/chat", protocol, location.Host)
	fmt.Printf("WebSocket URL: %s\n", url)

	ws := Global().value.Get("WebSocket").New(url)
	result := &WebSocket{
		ws: ws,
	}

	ws.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) any {
		result.onOpen()
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) any {
		data := args[0].Get("data").String()
		result.onMessage(data)
		return nil
	}))

	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) any {
		result.onClose()
		return nil
	}))

	ws.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) any {
		result.onError()
		return nil
	}))

	return result
}

func (ws *WebSocket) onOpen() {
	fmt.Println("onOpen")
	ws.Open = true
	go ws.keepAlive()
}

func (ws *WebSocket) onMessage(data string) {
	fmt.Println("onMessage")
	fmt.Printf("Received: %s\n", data)

	var message wire.BackendMessage
	if err := json.Unmarshal([]byte(data), &message); err != nil {
		fmt.Printf("json.Unmarshal(): %v\n", err)
	}

	if message.Pong != nil {
		fmt.Printf("Got pong: %d\n", message.Pong.Seq)
		ws.lastPingSeq.Store(message.Pong.Seq)
	}
}

func (ws *WebSocket) onClose() {
	fmt.Println("onClose")
	ws.Open = false
}

func (ws *WebSocket) onError() {
	fmt.Println("onError")
	ws.Open = false
}

func (ws *WebSocket) Send(message wire.FrontendMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	if ws.Open {
		ws.ws.Call("send", string(data))
	}

	return nil
}

func (ws *WebSocket) keepAlive() {
	var seq int64
	ws.lastPingSeq = &atomic.Int64{}

	for {
		if !ws.Open {
			return
		}

		if seq != ws.lastPingSeq.Load() {
			fmt.Printf("no response to last ping\n")
			ws.Open = false
			return
		}

		seq++

		ping := wire.FrontendMessage{
			Ping: &wire.PingMessage{
				Seq: seq,
				TS:  fmt.Sprintf("%d", time.Now().Unix()),
			},
		}
		err := ws.Send(ping)
		if err != nil {
			fmt.Printf("webSocket.Send(): %v", err)
		}

		time.Sleep(12 * time.Second)
	}
}
