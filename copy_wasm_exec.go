package main

import (
	"bytes"
	"go/build"
	"log"
	"os"
	"path/filepath"
)

func main() {
	srcFilename := filepath.Join(build.Default.GOROOT, "lib/wasm/wasm_exec.js")
	dstFilename := "web/dist/wasm_exec.js"

	srcContent, err := os.ReadFile(srcFilename)
	if err != nil {
		log.Fatalf("failed to read source file: %w", err)
	}

	dstContent, err := os.ReadFile(dstFilename)
	if err == nil && bytes.Equal(srcContent, dstContent) {
		return
	}

	// Preserve permissions if destination exists, otherwise default to 0644
	var mode os.FileMode = 0644
	if info, err := os.Stat(srcFilename); err == nil {
		mode = info.Mode()
	}

	if err := os.WriteFile(dstFilename, srcContent, mode); err != nil {
		log.Fatalf("failed to write destination file: %w", err)
	}
}
