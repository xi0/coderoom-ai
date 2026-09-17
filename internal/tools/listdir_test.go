package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/xi0/coderoom-ai/internal/wire"
)

func TestListDirValidDirectory(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "listdir_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files and subdirectories
	testFile := tmpDir + "/test.txt"
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testSubdir := tmpDir + "/subdir"
	if err := os.Mkdir(testSubdir, 0755); err != nil {
		t.Fatalf("Failed to create test subdirectory: %v", err)
	}

	testSubFile := testSubdir + "/subfile.txt"
	if err := os.WriteFile(testSubFile, []byte("subfile content"), 0644); err != nil {
		t.Fatalf("Failed to create test subfile: %v", err)
	}

	// Create root and open the temp directory
	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	// Create a channel to capture backend messages
	writeChannel := make(chan wire.BackendMessage, 10)
	defer close(writeChannel)

	options := &ToolOptions{
		Modifications: false,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	// Test listing the root of our temp directory
	argsJSON := `{"relative_path": "."}`
	tool := listDirTool()
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the result contains our test files and directories
	lines := strings.Split(result, "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 entries, got %d: %v", len(lines), lines)
	}

	// Check that both entries are present (order may vary)
	hasTestFile := false
	hasSubdir := false
	for _, line := range lines {
		if strings.Contains(line, "test.txt (file)") {
			hasTestFile = true
		}
		if strings.Contains(line, "subdir (dir)") {
			hasSubdir = true
		}
	}

	if !hasTestFile {
		t.Errorf("Expected result to contain 'test.txt (file)', got: %s", result)
	}
	if !hasSubdir {
		t.Errorf("Expected result to contain 'subdir (dir)', got: %s", result)
	}

	// Verify that a tool message was sent
	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Error("Expected ToolMessage to be set")
		}
	default:
		t.Error("Expected a message to be sent to writeChannel")
	}
}

func TestListDirNonExistentDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "listdir_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	writeChannel := make(chan wire.BackendMessage, 10)
	defer close(writeChannel)

	options := &ToolOptions{
		Modifications: false,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := listDirTool()
	argsJSON := `{"relative_path": "nonexistent"}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}

	if !strings.Contains(err.Error(), "failed to open directory") {
		t.Errorf("Expected error to contain 'failed to open directory', got: %v", err)
	}
}

func TestListDirRootDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "listdir_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file in the temp directory
	testFile := tmpDir + "/rootfile.txt"
	if err := os.WriteFile(testFile, []byte("root content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	writeChannel := make(chan wire.BackendMessage, 10)
	defer close(writeChannel)

	options := &ToolOptions{
		Modifications: false,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := listDirTool()

	// Test with "."
	argsJSON := `{"relative_path": "."}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error for '.' path, got: %v", err)
	}

	if !strings.Contains(result, "rootfile.txt (file)") {
		t.Errorf("Expected result to contain 'rootfile.txt (file)', got: %s", result)
	}

	// Clear the channel
	for len(writeChannel) > 0 {
		<-writeChannel
	}

	// Test with "" (empty string)
	argsJSON = `{"relative_path": ""}`
	result, err = tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error for '' path, got: %v", err)
	}

	if !strings.Contains(result, "rootfile.txt (file)") {
		t.Errorf("Expected result to contain 'rootfile.txt (file)', got: %s", result)
	}
}

func TestListDirInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "listdir_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	writeChannel := make(chan wire.BackendMessage, 10)
	defer close(writeChannel)

	options := &ToolOptions{
		Modifications: false,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := listDirTool()

	// Test with invalid JSON
	argsJSON := `{invalid json}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "invalid arguments for list_dir") {
		t.Errorf("Expected error to contain 'invalid arguments for list_dir', got: %v", err)
	}
}

func TestListDirMissingRequiredField(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "listdir_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	writeChannel := make(chan wire.BackendMessage, 10)
	defer close(writeChannel)

	options := &ToolOptions{
		Modifications: false,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := listDirTool()

	// Test with missing relative_path field. A missing field defaults to "",
	// which is treated as the root ("."). Since the root is empty, the tool
	// reports that the directory is empty.
	argsJSON := `{}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Fatal("Expected error for empty root directory, got nil")
	}

	if !strings.Contains(err.Error(), "directory is empty") {
		t.Errorf("Expected error to contain 'directory is empty', got: %v", err)
	}
}

func TestListDirEmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "listdir_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	writeChannel := make(chan wire.BackendMessage, 10)
	defer close(writeChannel)

	options := &ToolOptions{
		Modifications: false,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := listDirTool()
	argsJSON := `{"relative_path": "."}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for empty directory, got nil")
	}

	if !strings.Contains(err.Error(), "directory is empty") {
		t.Errorf("Expected error to contain 'directory is empty', got: %v", err)
	}
}

func TestListDirMultipleFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "listdir_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple test files
	files := []string{"file1.txt", "file2.txt", "file3.go"}
	for _, filename := range files {
		filepath := tmpDir + "/" + filename
		if err := os.WriteFile(filepath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create multiple subdirectories
	dirs := []string{"dir1", "dir2"}
	for _, dirname := range dirs {
		dirpath := tmpDir + "/" + dirname
		if err := os.Mkdir(dirpath, 0755); err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dirname, err)
		}
	}

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	writeChannel := make(chan wire.BackendMessage, 10)
	defer close(writeChannel)

	options := &ToolOptions{
		Modifications: false,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := listDirTool()
	argsJSON := `{"relative_path": "."}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Fatalf("Expected 5 entries, got %d: %v", len(lines), lines)
	}

	// Verify all files and directories are present
	expectedFiles := []string{
		"file1.txt (file)",
		"file2.txt (file)",
		"file3.go (file)",
		"dir1 (dir)",
		"dir2 (dir)",
	}

	for _, expected := range expectedFiles {
		found := false
		for _, line := range lines {
			if line == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %q in result, got: %s", expected, result)
		}
	}
}
