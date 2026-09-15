bin/coderoom-ai: cmd/coderoom-ai/main.go web/web.go web/assets/index.html web/assets/logo.png web/assets/styles.css web/dist/main.wasm web/dist/wasm_exec.js internal/backend/*.go internal/common/*.go internal/wire/*.go internal/tools/*.go
	go build -o bin/coderoom-ai cmd/coderoom-ai/main.go

web/dist/main.wasm: web/wasm/main.go internal/browser/*.go internal/ui/*.go internal/common/*.go internal/wire/*.go
	GOOS=js GOARCH=wasm go build -o web/dist/main.wasm web/wasm/main.go

web/dist/wasm_exec.js:
	go run copy_wasm_exec.go

test:
	go test -v internal/tools/*.go
