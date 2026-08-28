package backend

import (
	"log"

	"github.com/xi0/coderoom-ai/internal/wire"
)

type OpenAI struct{}

func (be *OpenAI) Run(writeChannel chan wire.BackendMessage, readChannel chan wire.FrontendMessage) {
	log.Println("OpenAI backend")
}
