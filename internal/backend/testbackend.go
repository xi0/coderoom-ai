package backend

import (
	"log"
	"time"

	"github.com/xi0/coderoom-ai/internal/wire"
)

type TestBackend struct{}

func (be *TestBackend) Run(writeChannel chan wire.BackendMessage, readChannel chan wire.FrontendMessage) {
	log.Println("Test backend")

	go func() {
		for message := range readChannel {
			log.Printf("Received: %v", message)
			if message.Ping != nil {
				writeChannel <- wire.BackendMessage{
					Pong: message.Ping,
				}
			}
		}
	}()

	for {
		//writeChannel <- []byte("Hello")

		time.Sleep(10 * time.Second)
	}
}
