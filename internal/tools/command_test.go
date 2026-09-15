package tools

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xi0/coderoom-ai/internal/wire"
)

// requireBash skips the calling test when bash is not available on PATH. The
// commandTool implementation shells out to bash, so every behavioural test
// depends on it being installed.
func requireBash(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not available: %v", err)
	}
}

// commandTestOptions builds a ToolOptions suitable for exercising commandTool.
// It creates a fresh temporary directory used as the tool root and a buffered
// write channel so the handler can forward the tool message without blocking.
// The temporary directory path is returned so tests can assert on files created
// by the executed command.
func commandTestOptions(t *testing.T) (*ToolOptions, chan wire.BackendMessage, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "command_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	t.Cleanup(func() { root.Close() })

	writeChannel := make(chan wire.BackendMessage, 10)

	options := &ToolOptions{
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	return options, writeChannel, tmpDir
}

func TestCommandToolSendsToolMessage(t *testing.T) {
	requireBash(t)
	options, writeChannel, _ := commandTestOptions(t)

	tool := commandTool("run_tests", "Runs all the tests for the project.", &wire.ToolSettings{Command: "true"})
	if _, err := tool.call("{}", options); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Fatal("Expected ToolMessage to be set")
		}
		if *msg.ToolMessage != "run_tests()" {
			t.Errorf("Expected tool message %q, got %q", "run_tests()", *msg.ToolMessage)
		}
	default:
		t.Error("Expected a message to be sent to writeChannel")
	}
}

func TestCommandToolExecutesCommand(t *testing.T) {
	requireBash(t)
	options, _, _ := commandTestOptions(t)

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{Command: "echo hello"})
	result, err := tool.call("{}", options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "hello\n" {
		t.Errorf("Expected output %q, got %q", "hello\n", result)
	}
}

func TestCommandToolRunsInRootDirectory(t *testing.T) {
	requireBash(t)
	options, _, tmpDir := commandTestOptions(t)

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{Command: "echo generated > created.txt"})
	if _, err := tool.call("{}", options); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "created.txt"))
	if err != nil {
		t.Fatalf("Expected command to create file in root directory: %v", err)
	}
	if string(content) != "generated\n" {
		t.Errorf("Expected file content %q, got %q", "generated\n", string(content))
	}
}

func TestCommandToolWorkingDirectoryIsRoot(t *testing.T) {
	requireBash(t)
	options, _, tmpDir := commandTestOptions(t)

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{Command: "pwd"})
	result, err := tool.call("{}", options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	got := strings.TrimSpace(result)
	resolvedWant := resolvePath(t, tmpDir)
	if resolvePath(t, got) != resolvedWant {
		t.Errorf("Expected command working directory %q, got %q", tmpDir, got)
	}
}

func TestCommandToolRootWithSpaces(t *testing.T) {
	requireBash(t)

	tmpDir, err := os.MkdirTemp("", "command test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	options := &ToolOptions{
		Modifications: true,
		Root:          root,
		WriteChannel:  make(chan wire.BackendMessage, 10),
	}

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{Command: "echo spaced > spaced.txt"})
	if _, err := tool.call("{}", options); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "spaced.txt")); err != nil {
		t.Errorf("Expected command to run in root directory containing spaces: %v", err)
	}
}

func TestCommandToolCapturesStdoutAndStderr(t *testing.T) {
	requireBash(t)
	options, _, _ := commandTestOptions(t)

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{Command: "echo to-stdout; echo to-stderr 1>&2"})
	result, err := tool.call("{}", options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !strings.Contains(result, "to-stdout") {
		t.Errorf("Expected combined output to contain stdout, got %q", result)
	}
	if !strings.Contains(result, "to-stderr") {
		t.Errorf("Expected combined output to contain stderr, got %q", result)
	}
}

func TestCommandToolMultilineOutput(t *testing.T) {
	requireBash(t)
	options, _, _ := commandTestOptions(t)

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{Command: "printf 'line1\\nline2\\nline3\\n'"})
	result, err := tool.call("{}", options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	want := "line1\nline2\nline3\n"
	if result != want {
		t.Errorf("Expected output %q, got %q", want, result)
	}
}

func TestCommandToolEmptyOutput(t *testing.T) {
	requireBash(t)
	options, _, _ := commandTestOptions(t)

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{Command: "true"})
	result, err := tool.call("{}", options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty output, got %q", result)
	}
}

func TestCommandToolIgnoresArguments(t *testing.T) {
	requireBash(t)
	options, _, _ := commandTestOptions(t)

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{Command: "echo ok"})
	// The handler does not parse its arguments, so even invalid JSON is fine.
	result, err := tool.call("{not valid json}", options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "ok\n" {
		t.Errorf("Expected output %q, got %q", "ok\n", result)
	}
}

func TestCommandToolCommandFailure(t *testing.T) {
	requireBash(t)
	options, _, _ := commandTestOptions(t)

	tool := commandTool("run_tests", "Runs all the tests for the project.", &wire.ToolSettings{Command: "exit 3"})
	_, err := tool.call("{}", options)
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 3 {
		t.Errorf("Expected exit code 3, got %d", exitErr.ExitCode())
	}
}

func TestCommandToolReturnsOutputOnFailure(t *testing.T) {
	requireBash(t)
	options, _, _ := commandTestOptions(t)

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{Command: "echo before-failure; exit 1"})
	result, err := tool.call("{}", options)
	if err == nil {
		t.Fatal("Expected error for failing command, got nil")
	}
	if !strings.Contains(result, "before-failure") {
		t.Errorf("Expected output to be returned on failure, got %q", result)
	}
}

func TestCommandToolNonExistentCommand(t *testing.T) {
	requireBash(t)
	options, _, _ := commandTestOptions(t)

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{Command: "definitely-not-a-real-command-xyz"})
	result, err := tool.call("{}", options)
	if err == nil {
		t.Fatal("Expected error for non-existent command, got nil")
	}
	if result == "" {
		t.Error("Expected bash error output to be returned")
	}
}

// resolvePath returns the symlink-resolved absolute form of path, falling back
// to the original string when resolution fails. Some platforms (e.g. macOS)
// expose temporary directories through symlinks, so comparisons of paths
// reported by child processes should be done on their resolved forms.
func resolvePath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolved
}
