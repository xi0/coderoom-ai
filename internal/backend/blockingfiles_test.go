package backend

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestMatchBlockingFile(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		relPath string
		want    bool
	}{
		// Patterns without "/" match the base name in any directory.
		{"base name root", "Makefile", "Makefile", true},
		{"base name nested", "Makefile", "sub/dir/Makefile", true},
		{"base name no match", "Makefile", "sub/dir/Makefile.am", false},
		{"wildcard base root", "*.go", "main.go", true},
		{"wildcard base nested", "*.go", "internal/ui/ui.go", true},
		{"wildcard base no match", "*.go", "internal/ui/ui.ts", false},
		{"question mark base", "a?c.go", "sub/abc.go", true},
		{"question mark base no match", "a?c.go", "sub/abbc.go", false},
		{"plain name matches any directory", "ui.go", "internal/ui/ui.go", true},
		{"star matches nested base", "*", "a/b/c.go", true},

		// Patterns with "/" match the path relative to the project root.
		{"relative path match", "internal/ui/*.go", "internal/ui/ui.go", true},
		{"relative path base mismatch", "internal/ui/*.go", "internal/backend/ui.go", false},
		{"relative path wildcard segment", "internal/*/ui.go", "internal/ui/ui.go", true},
		{"relative path anchored no deep match", "ui/ui.go", "internal/ui/ui.go", false},
		{"leading slash anchored", "/Makefile", "Makefile", true},
		{"leading slash not nested", "/Makefile", "sub/Makefile", false},
		{"leading slash relative path", "/internal/ui/ui.go", "internal/ui/ui.go", true},

		// Invalid patterns must not match.
		{"invalid pattern", "[", "main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchBlockingFile(tt.pattern, tt.relPath)
			if got != tt.want {
				t.Errorf("MatchBlockingFile(%q, %q) = %v, want %v", tt.pattern, tt.relPath, got, tt.want)
			}
		})
	}
}

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
