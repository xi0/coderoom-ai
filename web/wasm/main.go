package main

import (
	"fmt"

	_ "github.com/xi0/coderoom-ai/internal/ui"
)

func main() {
	fmt.Println("Hello from Go WebAssembly!")

	// Keep the Go program running so event listeners stay active
	select {}
}
