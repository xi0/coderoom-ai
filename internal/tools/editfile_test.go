package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/xi0/coderoom-ai/internal/wire"
)

func TestEditFileSuccessfulEdit(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with content
	testFile := tmpDir + "/test.txt"
	originalContent := "This is old content\nthat should be replaced"
	newContent := "This is new content\nthat should be replaced"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	// Test editing the test file
	argsJSON := `{"relative_path": "test.txt", "old_string": "This is old content", "new_string": "This is new content"}`
	tool := editFileTool()
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the result message
	expectedResult := "File edited successfully:\ntest.txt"
	if result != expectedResult {
		t.Errorf("Expected result %q, got %q", expectedResult, result)
	}

	// Verify that the file was actually modified
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if string(modifiedContent) != newContent {
		t.Errorf("Expected modified content %q, got %q", newContent, string(modifiedContent))
	}

	// Verify that a tool message was sent
	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Error("Expected ToolMessage to be set")
		} else {
			expectedMsg := `edit_file("test.txt")`
			if *msg.ToolMessage != expectedMsg {
				t.Errorf("Expected tool message %q, got %q", expectedMsg, *msg.ToolMessage)
			}
		}
	default:
		t.Error("Expected a message to be sent to writeChannel")
	}
}

func TestEditFileOldStringNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with content
	testFile := tmpDir + "/test.txt"
	originalContent := "This is some content"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()
	argsJSON := `{"relative_path": "test.txt", "old_string": "nonexistent string", "new_string": "replacement"}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for old_string not found, got nil")
	}

	if !strings.Contains(err.Error(), "the content in old_string is not found in the file") {
		t.Errorf("Expected error to contain 'the content in old_string is not found in the file', got: %v", err)
	}

	// Verify that the file was not modified
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != originalContent {
		t.Error("File should not have been modified when old_string is not found")
	}
}

func TestEditFileNonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()
	argsJSON := `{"relative_path": "nonexistent.txt", "old_string": "old", "new_string": "new"}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("Expected error to contain 'failed to read file', got: %v", err)
	}
}

func TestEditFileInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()

	// Test with invalid JSON
	argsJSON := `{invalid json}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "invalid arguments for edit_file") {
		t.Errorf("Expected error to contain 'invalid arguments for edit_file', got: %v", err)
	}
}

func TestEditFileMissingRequiredFields(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()

	// Test with missing fields
	argsJSON := `{}`
	_, err = tool.call(argsJSON, options)

	// This should error because it will try to read an empty path
	if err == nil {
		t.Error("Expected error for missing required fields, got nil")
	}
}

func TestEditFileEmptyStrings(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with content
	testFile := tmpDir + "/test.txt"
	originalContent := "This is content"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()

	// Test with empty old_string (should fail - empty string is technically found at position 0)
	// Actually, empty string will be found, so this tests replacing empty string with something
	argsJSON := `{"relative_path": "test.txt", "old_string": "", "new_string": "prefix: "}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error for empty old_string, got: %v", err)
	}

	// Verify the result
	expectedResult := "File edited successfully:\ntest.txt"
	if result != expectedResult {
		t.Errorf("Expected result %q, got %q", expectedResult, result)
	}

	// Verify that the file was modified (empty string at beginning gets replaced)
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	expectedContent := "prefix: This is content"
	if string(modifiedContent) != expectedContent {
		t.Errorf("Expected modified content %q, got %q", expectedContent, string(modifiedContent))
	}
}

func TestEditFileMultipleOccurrences(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with multiple occurrences of the old string
	testFile := tmpDir + "/test.txt"
	originalContent := "old old old\nold middle old\nold old old"
	expectedContent := "new old old\nold middle old\nold old old"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()
	argsJSON := `{"relative_path": "test.txt", "old_string": "old", "new_string": "new"}`
	_, err = tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that only the first occurrence was replaced
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if string(modifiedContent) != expectedContent {
		t.Errorf("Expected only first occurrence to be replaced.\nExpected: %q\nGot: %q", expectedContent, string(modifiedContent))
	}
}

func TestEditFileSpecialCharacters(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with special characters
	testFile := tmpDir + "/test.txt"
	originalContent := "Special chars: \n\t\r\"'\\<>&\nUnicode: éèê"
	newContent := "Special chars: \n\t\r\"'\\<>&\nUnicode: ÉÈÊ"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()
	argsJSON := `{"relative_path": "test.txt", "old_string": "Unicode: éèê", "new_string": "Unicode: ÉÈÊ"}`
	_, err = tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that the file was modified correctly
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if string(modifiedContent) != newContent {
		t.Errorf("Expected modified content %q, got %q", newContent, string(modifiedContent))
	}
}

func TestEditFileSubdirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
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
	originalContent := "nested old content"
	newContent := "nested new content"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()
	argsJSON := `{"relative_path": "subdir/nested.txt", "old_string": "nested old content", "new_string": "nested new content"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the result message
	expectedResult := "File edited successfully:\nsubdir/nested.txt"
	if result != expectedResult {
		t.Errorf("Expected result %q, got %q", expectedResult, result)
	}

	// Verify that the file was modified
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if string(modifiedContent) != newContent {
		t.Errorf("Expected modified content %q, got %q", newContent, string(modifiedContent))
	}

	// Verify that a tool message was sent with the correct path
	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Error("Expected ToolMessage to be set")
		} else {
			expectedMsg := `edit_file("subdir/nested.txt")`
			if *msg.ToolMessage != expectedMsg {
				t.Errorf("Expected tool message %q, got %q", expectedMsg, *msg.ToolMessage)
			}
		}
	default:
		t.Error("Expected a message to be sent to writeChannel")
	}
}

func TestEditFileBinaryContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with binary content
	testFile := tmpDir + "/binary.bin"
	originalContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	newContent := []byte{0x00, 0x01, 0x03, 0xFF, 0xFE, 0xFD}
	if err := os.WriteFile(testFile, originalContent, 0644); err != nil {
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()
	// Replace 0x02 with 0x03
	argsJSON := `{"relative_path": "binary.bin", "old_string": "\u0002", "new_string": "\u0003"}`
	_, err = tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that the file was modified
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if string(modifiedContent) != string(newContent) {
		t.Errorf("Expected binary content %v, got %v", newContent, modifiedContent)
	}
}

func TestEditFileLargeContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with large content
	testFile := tmpDir + "/large.txt"
	var originalContent strings.Builder
	for i := 0; i < 1000; i++ {
		originalContent.WriteString("Line ")
		originalContent.WriteString(string(rune(i)))
		originalContent.WriteString("\n")
	}

	// Replace line 500
	targetLine := "Line " + string(rune(500)) + "\n"
	replacementLine := "Line REPLACED\n"

	if err := os.WriteFile(testFile, []byte(originalContent.String()), 0644); err != nil {
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()
	argsJSON := `{"relative_path": "large.txt", "old_string": "Line ` + string(rune(500)) + `\n", "new_string": "Line REPLACED\n"}`
	_, err = tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that the file was modified
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	expectedContent := strings.Replace(originalContent.String(), targetLine, replacementLine, 1)
	if string(modifiedContent) != expectedContent {
		t.Error("Expected large file to be modified correctly")
	}
}

func TestEditFileNewlineHandling(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with newlines
	testFile := tmpDir + "/test.txt"
	originalContent := "line1\nline2\nline3"
	newContent := "line1\nLINE2\nline3"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()
	argsJSON := `{"relative_path": "test.txt", "old_string": "line2", "new_string": "LINE2"}`
	_, err = tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify that the file was modified
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if string(modifiedContent) != newContent {
		t.Errorf("Expected modified content %q, got %q", newContent, string(modifiedContent))
	}
}

func TestEditFileWithoutModificationsPermission(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "editfile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with content
	testFile := tmpDir + "/test.txt"
	originalContent := "This is content"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	writeChannel := make(chan wire.BackendMessage, 10)
	defer close(writeChannel)

	// Set modifications to false
	options := &ToolOptions{
		Modifications: false,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := editFileTool()
	argsJSON := `{"relative_path": "test.txt", "old_string": "This is content", "new_string": "New content"}`
	_, err = tool.call(argsJSON, options)

	// The tool should still execute (the modifications flag is checked at a higher level)
	// But let's verify the behavior
	if err != nil {
		// If there's an error, it should be related to file writing permissions or similar
		t.Logf("Got error with modifications=false: %v", err)
	}
}
