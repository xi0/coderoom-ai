package backend

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestMatchingFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "blockingfiles_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	files := []string{
		"Makefile",
		"main.go",
		"main_test.go",
		"internal/ui/ui.go",
		"internal/ui/ui_test.go",
		"internal/backend/settings.go",
		"docs/readme.md",
		".git/config",
	}

	for _, f := range files {
		p := filepath.Join(tmpDir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", f, err)
		}
		if err := os.WriteFile(p, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", f, err)
		}
	}

	settings := &Settings{ProjectDir: tmpDir}

	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{
			name:    "matches anywhere",
			pattern: "*_test.go",
			want: []string{
				"internal/ui/ui_test.go",
				"main_test.go",
			},
		},
		{
			name:    "root anchored",
			pattern: "/Makefile",
			want:    []string{"Makefile"},
		},
		{
			name:    "relative path",
			pattern: "internal/ui/*.go",
			want: []string{
				"internal/ui/ui.go",
				"internal/ui/ui_test.go",
			},
		},
		{
			name:    "no matches",
			pattern: "*.rs",
			want:    []string{},
		},
		{
			name:    "git directory skipped",
			pattern: "config",
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := settings.matchingFiles(tt.pattern)
			if err != nil {
				t.Fatalf("matchingFiles(%q): %v", tt.pattern, err)
			}
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) != len(tt.want) {
				t.Fatalf("matchingFiles(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("matchingFiles(%q) = %v, want %v", tt.pattern, got, tt.want)
					break
				}
			}
		})
	}
}
