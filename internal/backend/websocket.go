package backend

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/gorilla/websocket"
)

type Backend interface {
	Run(chan wire.BackendMessage, chan wire.FrontendMessage)
}

var (
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}

	backend Backend
)

func SetBackend(b Backend) {
	backend = b
}

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	writeChannel := make(chan wire.BackendMessage)
	readChannel := make(chan wire.FrontendMessage)

	go wsWriter(ws, writeChannel)
	go wsReader(ws, readChannel)

	backend.Run(writeChannel, readChannel)
}

func wsReader(ws *websocket.Conn, channel chan wire.FrontendMessage) {
	defer ws.Close()
	for {
		mt, p, err := ws.ReadMessage()
		if err != nil || mt != websocket.TextMessage {
			break
		}

		var message wire.FrontendMessage
		if err := json.Unmarshal(p, &message); err != nil {
			log.Printf("json.Unmarshal(): %v", err)
			break
		}

		channel <- message
	}
	close(channel)
}

func wsWriter(ws *websocket.Conn, channel chan wire.BackendMessage) {
	done := false

	for message := range channel {
		data, err := json.Marshal(message)
		if err != nil {
			log.Printf("json.Marshal(): %v", err)
			done = true
		}

		if !done {
			ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("ws.WriteMessage(): %v", err)
				done = true
			}
		}
	}
}
