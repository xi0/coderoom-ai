package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/xi0/coderoom-ai/web"
)

type webAsset struct {
	mimeType string
	payload  []byte
}

var (
	port = flag.Int("port", 8037, "")

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
		http.NotFoundHandler().ServeHTTP(w, r)
	}
}

func main() {
	fmt.Printf("Starting a web server on http://localhost:%d/\n", *port)

	http.HandleFunc("/", serveHTTP)

	http.ListenAndServe(fmt.Sprintf("localhost:%d", *port), nil)
}
