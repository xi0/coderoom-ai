package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/xi0/coderoom-ai/internal/blockingfiles"
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

	if err := s.loadProject(); err != nil {
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
	case "/settings/project/blocking-files":
		s.serveBlockingFiles(w, r)
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

func (s *Settings) GetAutoOpen() *string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.global == nil {
		return nil
	}

	return s.global.AutoOpen
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

func (s *Settings) GetProjectName() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.project == nil {
		return "-"
	}

	return s.project.Name
}

func (s *Settings) DirAllowed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.global == nil {
		return false
	}

	return slices.Contains(s.global.AllowedDirs, s.ProjectDir)
}

func (s *Settings) AllowDir() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.global == nil {
		return
	}

	s.global.AllowedDirs = append(s.global.AllowedDirs, s.ProjectDir)
}

func (s *Settings) GetDefaultProvider() *wire.ProviderSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.global == nil {
		return nil
	}

	for _, p := range s.global.Providers {
		if p.Default {
			return &p
		}
	}

	return nil
}

func (s *Settings) GetBuildProjectTool() *wire.ToolSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.project == nil {
		return nil
	}

	if s.project.BuildProjectTool == nil {
		return nil
	}

	// Make a copy of the tool object that can be used when the mutex is not held.
	tool := *s.project.BuildProjectTool
	return &tool
}

func (s *Settings) GetRunTestsTool() *wire.ToolSettings {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.project == nil {
		return nil
	}

	if s.project.RunTestsTool == nil {
		return nil
	}

	// Make a copy of the tool object that can be used when the mutex is not held.
	tool := *s.project.RunTestsTool
	return &tool
}

func (s *Settings) serveProject(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		data, err := json.Marshal(s.project)
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
		err = json.Unmarshal(data, s.project)
		s.mu.Unlock()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			log.Printf("json.Unmarshal(): %v", err)
			return
		}

		s.saveProject()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"ok\": true}"))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Settings) loadProject() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileName := filepath.Join(s.ProjectDir, "coderoom-ai.json")
	data, err := os.ReadFile(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			s.project = &wire.ProjectSettings{
				Name: filepath.Base(s.ProjectDir),
			}
			return nil
		} else {
			return err
		}
	}

	s.project = &wire.ProjectSettings{}
	if err := json.Unmarshal(data, s.project); err != nil {
		return fmt.Errorf("json.Unmarshal(): %v", err)
	}

	return nil
}

func (s *Settings) saveProject() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s.project, "", "  ")
	if err != nil {
		return fmt.Errorf("json.MarshalIndent(): %v", err)
	}

	fileName := filepath.Join(s.ProjectDir, "coderoom-ai.json")
	if err := os.WriteFile(fileName, data, 0644); err != nil {
		return err
	}

	return nil
}

func (s *Settings) serveBlockingFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var request wire.BlockingFilesRequest
	if err := json.Unmarshal(data, &request); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var response wire.BlockingFilesResponse

	files, err := s.matchingFiles(request.Pattern)
	if err != nil {
		response.Error = err.Error()
	} else {
		response.Files = files
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("json.NewEncoder().Encode(): %v", err)
	}
}

// matchingFiles returns the paths of all regular files under the project
// directory that match the given pattern. The returned paths are relative to
// the project root and use "/" as separator. The .git directory is skipped.
func (s *Settings) matchingFiles(pattern string) ([]string, error) {
	root := s.ProjectDir
	matches := []string{}

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}

		if blockingfiles.MatchBlockingFile(pattern, rel) {
			matches = append(matches, filepath.ToSlash(rel))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return matches, nil
}
