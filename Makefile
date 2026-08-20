bin/coderoom-ai: cmd/coderoom-ai/main.go web/web.go web/assets/index.html web/assets/logo.png web/assets/styles.css web/dist/main.wasm web/dist/wasm_exec.js internal/backend/websocket.go internal/backend/testbackend.go internal/backend/openai.go internal/wire/messages.go
	go build -o bin/coderoom-ai cmd/coderoom-ai/main.go

web/dist/main.wasm: web/wasm/main.go internal/browser/object.go internal/browser/element.go internal/browser/websocket.go internal/ui/ui.go internal/ui/message.go internal/wire/messages.go
	GOOS=js GOARCH=wasm go build -o web/dist/main.wasm web/wasm/main.go

web/dist/wasm_exec.js:
	go run copy_wasm_exec.go
