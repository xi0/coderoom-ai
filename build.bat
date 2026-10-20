@echo off

set GOOS=js
set GOARCH=wasm

go build -o web/dist/main.wasm web/wasm/main.go
if %errorlevel% neq 0 (
    exit /b 1
)

set GOOS=
set GOARCH=

go run copy_wasm_exec.go
if %errorlevel% neq 0 (
    exit /b 1
)

go build -o bin/coderoom-ai.exe cmd/coderoom-ai/main.go
if %errorlevel% neq 0 (
    exit /b 1
)
