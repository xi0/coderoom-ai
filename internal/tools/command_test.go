package tools

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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

// blockingCommandOptions builds on commandTestOptions by recording the supplied
// paths as modified files in a FileEdits instance and wiring up a confirmation
// channel with the requested buffer size. The confirmation channel is returned
// so tests can pre-load a decision or drive it from a separate goroutine.
func blockingCommandOptions(t *testing.T, modifiedFiles []string, confirmationBuffer int) (*ToolOptions, chan wire.BackendMessage, chan bool, string) {
	t.Helper()

	options, writeChannel, tmpDir := commandTestOptions(t)

	edits := NewFileEdits(options.Root)
	for _, f := range modifiedFiles {
		// The content is irrelevant to blocking; WriteFile only needs to
		// register the path so it shows up in Filenames.
		edits.WriteFile(f, []byte("modified"), false)
	}

	confirmationChannel := make(chan bool, confirmationBuffer)
	options.Edits = edits
	options.ConfirmationChannel = confirmationChannel

	return options, writeChannel, confirmationChannel, tmpDir
}

// requireToolMessage consumes the initial tool message emitted by commandTool
// and asserts it matches want. It fails the test if the message is missing,
// malformed or unexpected.
func requireToolMessage(t *testing.T, writeChannel chan wire.BackendMessage, want string) {
	t.Helper()

	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Fatalf("Expected ToolMessage to be set, got %+v", msg)
		}
		if *msg.ToolMessage != want {
			t.Errorf("Expected tool message %q, got %q", want, *msg.ToolMessage)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for the tool message")
	}
}

// requireBlockedMessage consumes the blocked message emitted by commandTool and
// asserts its action, filenames (in order) and the WorkDone flag.
func requireBlockedMessage(t *testing.T, writeChannel chan wire.BackendMessage, wantAction string, wantFilenames []string) {
	t.Helper()

	select {
	case msg := <-writeChannel:
		if msg.BlockedMessage == nil {
			t.Fatalf("Expected BlockedMessage to be set, got %+v", msg)
		}
		if msg.BlockedMessage.Action != wantAction {
			t.Errorf("Expected blocked action %q, got %q", wantAction, msg.BlockedMessage.Action)
		}
		if !reflect.DeepEqual(msg.BlockedMessage.Filenames, wantFilenames) {
			t.Errorf("Expected blocked filenames %v, got %v", wantFilenames, msg.BlockedMessage.Filenames)
		}
		if !msg.WorkDone {
			t.Error("Expected WorkDone to be true on the blocked message")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for the blocked message")
	}
}

// requireNoBlockedMessage asserts that no blocked message has been queued.
func requireNoBlockedMessage(t *testing.T, writeChannel chan wire.BackendMessage) {
	t.Helper()

	select {
	case msg := <-writeChannel:
		t.Fatalf("Expected no further messages, got %+v", msg)
	default:
	}
}

// TestCommandToolSendsBlockedMessageForModifiedBlockingFile verifies that when a
// modified file matches a blocking pattern the tool forwards a BlockedMessage
// and runs the command once the user confirms.
func TestCommandToolSendsBlockedMessageForModifiedBlockingFile(t *testing.T) {
	requireBash(t)
	options, writeChannel, confirmationChannel, tmpDir := blockingCommandOptions(t, []string{"Makefile"}, 1)

	// Pre-approve the block so the handler can proceed without a goroutine.
	confirmationChannel <- true

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{
		Command:       "echo built > built.txt",
		BlockingFiles: []string{"Makefile"},
	})

	result, err := tool.call("{}", options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty output, got %q", result)
	}

	requireToolMessage(t, writeChannel, "build_project()")
	requireBlockedMessage(t, writeChannel, "build_project", []string{"Makefile"})

	// The command runs only after the block has been confirmed.
	if _, err := os.Stat(filepath.Join(tmpDir, "built.txt")); err != nil {
		t.Errorf("Expected command to run after confirmation: %v", err)
	}
}

// TestCommandToolWaitsForConfirmation proves that the handler does not run the
// command until the user confirms the block. It uses an unbuffered confirmation
// channel and a goroutine so the ordering is observable.
func TestCommandToolWaitsForConfirmation(t *testing.T) {
	requireBash(t)
	options, writeChannel, confirmationChannel, tmpDir := blockingCommandOptions(t, []string{"Makefile"}, 0)

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{
		Command:       "echo done > confirmed.txt",
		BlockingFiles: []string{"Makefile"},
	})

	type callResult struct {
		out string
		err error
	}
	resultChannel := make(chan callResult, 1)

	go func() {
		out, err := tool.call("{}", options)
		resultChannel <- callResult{out: out, err: err}
	}()

	// The tool message and blocked message are emitted before the tool waits.
	requireToolMessage(t, writeChannel, "build_project()")
	requireBlockedMessage(t, writeChannel, "build_project", []string{"Makefile"})

	// Without confirmation the call must not complete, and the command must
	// not have created its output file yet.
	select {
	case res := <-resultChannel:
		t.Fatalf("Expected the call to block on confirmation, but it returned: out=%q err=%v", res.out, res.err)
	case <-time.After(100 * time.Millisecond):
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "confirmed.txt")); !os.IsNotExist(err) {
		t.Fatalf("Expected command not to run before confirmation, stat err = %v", err)
	}

	// Confirming lets the handler proceed and run the command.
	confirmationChannel <- true

	select {
	case res := <-resultChannel:
		if res.err != nil {
			t.Fatalf("Expected no error, got: %v", res.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for the command to complete")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "confirmed.txt")); err != nil {
		t.Errorf("Expected command to run after confirmation: %v", err)
	}
}

// TestCommandToolBlockedByUser verifies that rejecting the block returns an
// error and prevents the command from running.
func TestCommandToolBlockedByUser(t *testing.T) {
	options, writeChannel, confirmationChannel, tmpDir := blockingCommandOptions(t, []string{"Makefile"}, 1)

	confirmationChannel <- false

	tool := commandTool("run_tests", "Runs all the tests for the project.", &wire.ToolSettings{
		Command:       "echo ran > should-not-exist.txt",
		BlockingFiles: []string{"Makefile"},
	})

	_, err := tool.call("{}", options)
	if err == nil {
		t.Fatal("Expected an error when the user blocks the action, got nil")
	}
	if !strings.Contains(err.Error(), `"run_tests" action blocked by user`) {
		t.Errorf(`Expected error to contain %q, got %q`, `"run_tests" action blocked by user`, err.Error())
	}

	requireToolMessage(t, writeChannel, "run_tests()")
	requireBlockedMessage(t, writeChannel, "run_tests", []string{"Makefile"})

	// The command must not have run when the user rejected the block.
	if _, err := os.Stat(filepath.Join(tmpDir, "should-not-exist.txt")); !os.IsNotExist(err) {
		t.Errorf("Expected command not to run when blocked, stat err = %v", err)
	}
}

// TestCommandToolBlockedActionUsesToolName ensures the blocked message reports
// the command tool's own name rather than a hard-coded action.
func TestCommandToolBlockedActionUsesToolName(t *testing.T) {
	options, writeChannel, confirmationChannel, _ := blockingCommandOptions(t, []string{"Makefile"}, 1)

	confirmationChannel <- false

	tool := commandTool("custom_action", "Does something.", &wire.ToolSettings{
		Command:       "true",
		BlockingFiles: []string{"Makefile"},
	})

	if _, err := tool.call("{}", options); err == nil {
		t.Fatal("Expected an error when the user blocks the action, got nil")
	}

	requireToolMessage(t, writeChannel, "custom_action()")
	requireBlockedMessage(t, writeChannel, "custom_action", []string{"Makefile"})
}

// TestCommandToolBlockedMessageListsOnlyMatchingFiles verifies that only the
// modified files matching a blocking pattern are reported, in the sorted order
// produced by FileEdits.Filenames.
func TestCommandToolBlockedMessageListsOnlyMatchingFiles(t *testing.T) {
	// Filenames() sorts lexicographically: "Makefile", "go.mod", "sub/Makefile".
	options, writeChannel, confirmationChannel, _ := blockingCommandOptions(t, []string{"go.mod", "sub/Makefile", "Makefile"}, 1)

	confirmationChannel <- false

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{
		Command:       "true",
		BlockingFiles: []string{"Makefile"},
	})

	if _, err := tool.call("{}", options); err == nil {
		t.Fatal("Expected an error when the user blocks the action, got nil")
	}

	requireToolMessage(t, writeChannel, "build_project()")
	requireBlockedMessage(t, writeChannel, "build_project", []string{"Makefile", "sub/Makefile"})
}

// TestCommandToolBlockedFilesNotDuplicated verifies that a file matching several
// blocking patterns is only reported once.
func TestCommandToolBlockedFilesNotDuplicated(t *testing.T) {
	options, writeChannel, confirmationChannel, _ := blockingCommandOptions(t, []string{"Makefile"}, 1)

	confirmationChannel <- false

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{
		Command:       "true",
		BlockingFiles: []string{"Makefile", "Make*", "*file"},
	})

	if _, err := tool.call("{}", options); err == nil {
		t.Fatal("Expected an error when the user blocks the action, got nil")
	}

	requireToolMessage(t, writeChannel, "build_project()")
	requireBlockedMessage(t, writeChannel, "build_project", []string{"Makefile"})
}

// TestCommandToolBlockedWildcardPattern verifies wildcard blocking patterns are
// matched against the base name of modified files in nested directories.
func TestCommandToolBlockedWildcardPattern(t *testing.T) {
	options, writeChannel, confirmationChannel, _ := blockingCommandOptions(t, []string{"main.go", "internal/ui/ui.go", "README.md"}, 1)

	confirmationChannel <- false

	tool := commandTool("run_tests", "Runs all the tests for the project.", &wire.ToolSettings{
		Command:       "true",
		BlockingFiles: []string{"*.go"},
	})

	if _, err := tool.call("{}", options); err == nil {
		t.Fatal("Expected an error when the user blocks the action, got nil")
	}

	requireToolMessage(t, writeChannel, "run_tests()")
	// Filenames() sorts to: internal/ui/ui.go, main.go, README.md; only the
	// two .go files match "*.go".
	requireBlockedMessage(t, writeChannel, "run_tests", []string{"internal/ui/ui.go", "main.go"})
}

// TestCommandToolNoBlockWhenModifiedFileDoesNotMatch verifies that a modified
// file that does not match any blocking pattern does not trigger a block.
func TestCommandToolNoBlockWhenModifiedFileDoesNotMatch(t *testing.T) {
	requireBash(t)
	// Pre-load a confirmation so the test cannot hang if the handler
	// unexpectedly blocks.
	options, writeChannel, confirmationChannel, _ := blockingCommandOptions(t, []string{"main.go"}, 1)
	confirmationChannel <- true

	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{
		Command:       "echo ok",
		BlockingFiles: []string{"Makefile"},
	})

	result, err := tool.call("{}", options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "ok\n" {
		t.Errorf("Expected output %q, got %q", "ok\n", result)
	}

	requireToolMessage(t, writeChannel, "build_project()")
	requireNoBlockedMessage(t, writeChannel)
}

// TestCommandToolNoBlockWithoutEdits verifies that a nil FileEdits (no recorded
// modifications) never triggers a block, even when a blocking pattern is set.
func TestCommandToolNoBlockWithoutEdits(t *testing.T) {
	requireBash(t)
	options, writeChannel, _ := commandTestOptions(t)

	// Pre-load a confirmation so the test cannot hang if the handler
	// unexpectedly blocks.
	confirmationChannel := make(chan bool, 1)
	confirmationChannel <- true
	options.ConfirmationChannel = confirmationChannel

	// Edits is intentionally left nil.
	tool := commandTool("build_project", "Builds the project.", &wire.ToolSettings{
		Command:       "echo ok",
		BlockingFiles: []string{"Makefile"},
	})

	result, err := tool.call("{}", options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "ok\n" {
		t.Errorf("Expected output %q, got %q", "ok\n", result)
	}

	requireToolMessage(t, writeChannel, "build_project()")
	requireNoBlockedMessage(t, writeChannel)
}
