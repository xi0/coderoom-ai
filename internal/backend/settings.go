package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/xi0/coderoom-ai/internal/wire"
)

type Settings struct {
	mu          sync.Mutex
	SettingsDir string
	ProjectDir  string
	global      *wire.GlobalSettings
	project     *wire.ProjectSettings
}

func NewSettings(settingsDir, projectDir string) (*Settings, error) {
	s := &Settings{
		SettingsDir: settingsDir,
		ProjectDir:  projectDir,
	}

	if err := s.loadGlobal(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Settings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/settings/global":
		s.serveGlobal(w, r)
	case "/settings/global/theme":
		s.serveTheme(w, r)
	case "/settings/project":
		s.serveProject(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Settings) serveGlobal(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		data, err := json.Marshal(s.global)
		defer s.mu.Unlock()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Printf("json.Marshal(): %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	case http.MethodPost:
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		s.mu.Lock()
		err = json.Unmarshal(data, s.global)
		s.mu.Unlock()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Printf("json.Unmarshal(): %v", err)
			return
		}

		s.saveGlobal()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"ok\": true}"))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Settings) serveTheme(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	darkTheme := r.FormValue("dark_theme")

	if darkTheme == "1" {
		s.SetDarkTheme(true)
	} else {
		s.SetDarkTheme(false)
	}

	s.saveGlobal()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{\"ok\": true}"))
}

func (s *Settings) loadGlobal() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.SettingsDir); os.IsNotExist(err) {
		err := os.MkdirAll(s.SettingsDir, 0755)
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	fileName := filepath.Join(s.SettingsDir, "global.json")
	data, err := os.ReadFile(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			s.global = &wire.GlobalSettings{
				DefaultModifications: true,
				DarkTheme:            true,
			}
			return nil
		} else {
			return err
		}
	}

	s.global = &wire.GlobalSettings{}
	if err := json.Unmarshal(data, s.global); err != nil {
		return fmt.Errorf("json.Unmarshal(): %v", err)
	}

	return nil
}

func (s *Settings) saveGlobal() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(s.global)
	if err != nil {
		return fmt.Errorf("json.Marshal(): %v", err)
	}

	fileName := filepath.Join(s.SettingsDir, "global.json")
	if err := os.WriteFile(fileName, data, 0600); err != nil {
		return err
	}

	return nil
}

func (s *Settings) GetDefaultModifications() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.global == nil {
		return true
	}

	return s.global.DefaultModifications
}

func (s *Settings) GetDarkTheme() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.global == nil {
		return true
	}

	return s.global.DarkTheme
}

func (s *Settings) SetDarkTheme(darkTheme bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.global == nil {
		return
	}

	s.global.DarkTheme = darkTheme
}

func (s *Settings) serveProject(w http.ResponseWriter, r *http.Request) {
}
