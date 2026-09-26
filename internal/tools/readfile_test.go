package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/xi0/coderoom-ai/internal/wire"
)

func TestReadFileValidFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "readfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with content
	testFile := tmpDir + "/test.txt"
	testContent := "This is test content\nwith multiple lines"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
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

	// Test reading the test file
	argsJSON := `{"relative_path": "test.txt"}`
	tool := readFileTool()
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the content matches
	if result != testContent {
		t.Errorf("Expected content %q, got %q", testContent, result)
	}

	// Verify that a tool message was sent
	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Error("Expected ToolMessage to be set")
		} else {
			expectedMsg := `read_file("test.txt")`
			if msg.ToolMessage.Tool != expectedMsg {
				t.Errorf("Expected tool message %q, got %q", expectedMsg, msg.ToolMessage.Tool)
			}
		}
	default:
		t.Error("Expected a message to be sent to writeChannel")
	}
}

func TestReadFileNonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readfile_test_*")
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

	tool := readFileTool()
	argsJSON := `{"relative_path": "nonexistent.txt"}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("Expected error to contain 'failed to read file', got: %v", err)
	}
}

func TestReadFileRootDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file in the temp directory
	testFile := tmpDir + "/rootfile.txt"
	testContent := "root file content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
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

	tool := readFileTool()

	// Test with explicit filename
	argsJSON := `{"relative_path": "rootfile.txt"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != testContent {
		t.Errorf("Expected content %q, got %q", testContent, result)
	}

	// Clear the channel
	for len(writeChannel) > 0 {
		<-writeChannel
	}
}

func TestReadFileInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readfile_test_*")
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

	tool := readFileTool()

	// Test with invalid JSON
	argsJSON := `{invalid json}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "invalid arguments for read_file") {
		t.Errorf("Expected error to contain 'invalid arguments for read_file', got: %v", err)
	}
}

func TestReadFileMissingRequiredField(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readfile_test_*")
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

	tool := readFileTool()

	// Test with missing relative_path field
	argsJSON := `{}`
	_, err = tool.call(argsJSON, options)

	// This should error because it will try to read an empty path
	if err == nil {
		t.Error("Expected error for missing relative_path, got nil")
	}
}

func TestReadFileEmptyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an empty test file
	testFile := tmpDir + "/empty.txt"
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty test file: %v", err)
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

	tool := readFileTool()
	argsJSON := `{"relative_path": "empty.txt"}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for empty file, got nil")
	}

}

func TestReadFileWithSpecialCharacters(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with special characters
	testFile := tmpDir + "/special.txt"
	testContent := "Special chars: \n\t\r\"'\\<>&\nUnicode: \u00e9\u00e8\u00ea"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
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

	tool := readFileTool()
	argsJSON := `{"relative_path": "special.txt"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the content matches exactly
	if result != testContent {
		t.Errorf("Expected content %q, got %q", testContent, result)
	}
}

func TestReadFileSubdirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory
	testSubdir := tmpDir + "/subdir"
	if err := os.Mkdir(testSubdir, 0755); err != nil {
		t.Fatalf("Failed to create test subdirectory: %v", err)
	}

	// Create a test file in the subdirectory
	testFile := testSubdir + "/nested.txt"
	testContent := "nested file content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
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

	tool := readFileTool()
	argsJSON := `{"relative_path": "subdir/nested.txt"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the content matches
	if result != testContent {
		t.Errorf("Expected content %q, got %q", testContent, result)
	}

	// Verify that a tool message was sent with the correct path
	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Error("Expected ToolMessage to be set")
		} else {
			expectedMsg := `read_file("subdir/nested.txt")`
			if msg.ToolMessage.Tool != expectedMsg {
				t.Errorf("Expected tool message %q, got %q", expectedMsg, msg.ToolMessage.Tool)
			}
		}
	default:
		t.Error("Expected a message to be sent to writeChannel")
	}
}

func TestReadFileLargeContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with large content
	testFile := tmpDir + "/large.txt"
	// Create content with 1000 lines
	var testContent strings.Builder
	for i := 0; i < 1000; i++ {
		testContent.WriteString("Line ")
		testContent.WriteString(string(rune(i)))
		testContent.WriteString("\n")
	}

	if err := os.WriteFile(testFile, []byte(testContent.String()), 0644); err != nil {
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

	tool := readFileTool()
	argsJSON := `{"relative_path": "large.txt"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the content matches
	if result != testContent.String() {
		t.Errorf("Expected content length %d, got %d", len(testContent.String()), len(result))
	}
}

func TestReadFileBinaryContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "readfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with binary content
	testFile := tmpDir + "/binary.bin"
	testContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
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

	tool := readFileTool()
	argsJSON := `{"relative_path": "binary.bin"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the content matches
	if result != string(testContent) {
		t.Errorf("Expected binary content to match")
	}
}
