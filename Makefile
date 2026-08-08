.PHONY: bin/coderoom-ai

bin/coderoom-ai: cmd/coderoom-ai/main.go web/web.go web/assets/index.html web/assets/logo.png web/assets/styles.css
	go build -o bin/coderoom-ai cmd/coderoom-ai/main.go
