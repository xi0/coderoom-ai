@echo off

go test -v github.com/xi0/coderoom-ai/internal/backend
if %errorlevel% neq 0 (
    exit /b 1
)

go test -v github.com/xi0/coderoom-ai/internal/blockingfiles
if %errorlevel% neq 0 (
    exit /b 1
)

go test -v github.com/xi0/coderoom-ai/internal/tools
if %errorlevel% neq 0 (
    exit /b 1
)
