package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/xi0/coderoom-ai/internal/backend"
	"github.com/xi0/coderoom-ai/web"
)

type webAsset struct {
	mimeType string
	payload  []byte
}

var (
	projectDir = flag.String("dir", "", "project directory")
	port       = flag.Int("port", 8037, "TCP port to listen on")
	testMode   = flag.Bool("testmode", false, "run the backend in test mode")

	staticAssets = map[string]webAsset{
		"/": webAsset{
			mimeType: "text/html",
			payload:  web.IndexHTML,
		},
		"/styles.css": webAsset{
			mimeType: "text/css",
			payload:  web.StylesCSS,
		},
		"/logo.png": webAsset{
			mimeType: "image/png",
			payload:  web.Logo,
		},
		"/main.wasm": webAsset{
			mimeType: "application/wasm",
			payload:  web.MainWASM,
		},
		"/wasm_exec.js": webAsset{
			mimeType: "text/javascript",
			payload:  web.WASMExec,
		},
	}
)

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if asset, ok := staticAssets[path]; ok {
		w.Header().Set("Content-Type", asset.mimeType)
		w.Write(asset.payload)
	} else {
		http.NotFound(w, r)
	}
}

func main() {
	flag.Parse()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("os.UserHomeDir(): %v", err)
	}

	settingsDir := filepath.Join(homeDir, ".config/coderoom-ai")
	fmt.Printf("Settings directory: %s\n", settingsDir)

	if *projectDir == "" {
		*projectDir, err = os.Getwd()
		if err != nil {
			log.Fatalf("os.Getwd(): %v", err)
		}
	}
	fmt.Printf("Project directory: %s\n", *projectDir)

	settings, err := backend.NewSettings(settingsDir, *projectDir)
	if err != nil {
		log.Fatalf("backend.NewSettings(): %v", err)
	}

	if *testMode {
		backend.SetBackend(&backend.TestBackend{Settings: settings})
	} else {
		backend.SetBackend(&backend.OpenAI{})
	}

	http.HandleFunc("/", serveHTTP)
	http.HandleFunc("/chat", backend.ServeHTTP)
	http.Handle("/settings/", settings)

	fmt.Printf("\nStarting a web server on http://localhost:%d/\n", *port)

	http.ListenAndServe(fmt.Sprintf("localhost:%d", *port), nil)
}
