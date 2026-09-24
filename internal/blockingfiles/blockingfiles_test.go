package blockingfiles

import (
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
