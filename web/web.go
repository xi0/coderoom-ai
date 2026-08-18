package web

import (
	_ "embed"
)

var (
	//go:embed assets/index.html
	IndexHTML []byte
	//go:embed assets/styles.css
	StylesCSS []byte
	//go:embed assets/logo.png
	Logo []byte
	//go:embed dist/main.wasm
	MainWASM []byte
	//go:embed dist/wasm_exec.js
	WASMExec []byte
)
