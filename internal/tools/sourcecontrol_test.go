package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/xi0/coderoom-ai/internal/wire"
)

func TestIsSourceControlPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".git", true},
		{".git/config", true},
		{"sub/.git/config", true},
		{"sub/../.git/config", true},
		{`sub\.git\config`, true},
		{".svn/entries", true},
		{".hg/hgrc", true},
		{".bzr/branch", true},
		{"_darcs/prefs", true},
		{"CVS/Root", true},
		{"sub/cvs/Root", true},
		{".GIT/config", true},
		{"git/config", false},
		{".gitignore", false},
		{"src/.github/workflows/build.yml", false},
		{"src/main.go", false},
		{".", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := isSourceControlPath(tt.path); got != tt.want {
			t.Errorf("isSourceControlPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func newSourceControlTestOptions(t *testing.T) (*os.Root, string, *ToolOptions) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "sourcecontrol_test_*")
	if err != nil {
		t.Fatalf("os.MkdirTemp(): %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("os.OpenRoot(): %v", err)
	}
	t.Cleanup(func() { root.Close() })

	writeChannel := make(chan wire.BackendMessage, 10)
	t.Cleanup(func() { close(writeChannel) })

	return root, tmpDir, &ToolOptions{
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}
}

func TestFileToolsRejectSourceControlPaths(t *testing.T) {
	root, tmpDir, options := newSourceControlTestOptions(t)

	if err := os.MkdirAll(tmpDir+"/.git", 0755); err != nil {
		t.Fatalf("os.MkdirAll(): %v", err)
	}
	if err := os.WriteFile(tmpDir+"/.git/config", []byte("original"), 0644); err != nil {
		t.Fatalf("os.WriteFile(): %v", err)
	}

	tests := []struct {
		name string
		tool *Tool
		args string
	}{
		{
			name: "read_file",
			tool: readFileTool(),
			args: `{"relative_path": ".git/config"}`,
		},
		{
			name: "write_file",
			tool: writeFileTool(),
			args: `{"relative_path": ".git/config", "content": "overwritten"}`,
		},
		{
			name: "edit_file",
			tool: editFileTool(),
			args: `{"relative_path": ".git/config", "old_string": "original", "new_string": "edited"}`,
		},
		{
			name: "delete_file",
			tool: deleteFileTool(),
			args: `{"relative_path": ".git/config"}`,
		},
		{
			name: "list_dir",
			tool: listDirTool(),
			args: `{"relative_path": ".git"}`,
		},
		{
			name: "nested write_file",
			tool: writeFileTool(),
			args: `{"relative_path": "sub/../.git/config", "content": "overwritten"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.tool.call(tt.args, options)
			if err == nil {
				t.Fatal("expected source control path to be rejected")
			}
			if !strings.Contains(strings.ToLower(err.Error()), "source control") {
				t.Errorf("expected error to mention source control, got: %v", err)
			}
		})
	}

	content, err := root.ReadFile(".git/config")
	if err != nil {
		t.Fatalf("root.ReadFile(): %v", err)
	}
	if string(content) != "original" {
		t.Errorf("source control file was modified: %q", content)
	}
}

func TestListDirOmitsSourceControlEntries(t *testing.T) {
	_, tmpDir, options := newSourceControlTestOptions(t)

	if err := os.MkdirAll(tmpDir+"/.git", 0755); err != nil {
		t.Fatalf("os.MkdirAll(): %v", err)
	}
	if err := os.MkdirAll(tmpDir+"/.svn", 0755); err != nil {
		t.Fatalf("os.MkdirAll(): %v", err)
	}
	if err := os.WriteFile(tmpDir+"/main.go", []byte("package main"), 0644); err != nil {
		t.Fatalf("os.WriteFile(): %v", err)
	}

	result, err := listDirTool().call(`{"relative_path": "."}`, options)
	if err != nil {
		t.Fatalf("list_dir returned error: %v", err)
	}

	if strings.Contains(result, ".git") {
		t.Errorf("expected .git to be omitted, got: %s", result)
	}
	if strings.Contains(result, ".svn") {
		t.Errorf("expected .svn to be omitted, got: %s", result)
	}
	if !strings.Contains(result, "main.go") {
		t.Errorf("expected main.go to be listed, got: %s", result)
	}
}
