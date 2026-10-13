package backend

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"sync"

	"github.com/xi0/coderoom-ai/internal/wire"
)

type Edits struct {
	mu sync.Mutex
	// edits[sessionID][filename]
	edits map[string]map[string]*wire.EditResponse
}

var (
	edits         Edits
	editsURLMatch = regexp.MustCompile("^/edits/([0-9a-f]{32})/(.*)$")
)

func init() {
	edits = Edits{
		edits: make(map[string]map[string]*wire.EditResponse),
	}
}

func addEdits(sessionID string, files map[string]*wire.EditResponse) {
	edits.mu.Lock()
	defer edits.mu.Unlock()

	edits.edits[sessionID] = files
}

func clearEdits(sessionID string) {
	edits.mu.Lock()
	defer edits.mu.Unlock()

	delete(edits.edits, sessionID)
}

func ServeEdits(w http.ResponseWriter, r *http.Request) {
	m := editsURLMatch.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}

	sessionID := m[1]
	filename := m[2]

	if _, ok := edits.edits[sessionID]; !ok {
		http.NotFound(w, r)
		return
	}

	if _, ok := edits.edits[sessionID][filename]; !ok {
		http.NotFound(w, r)
		return
	}

	json, err := json.Marshal(edits.edits[sessionID][filename])
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Printf("json.Marshal(): %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}
