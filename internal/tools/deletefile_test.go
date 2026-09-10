package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/xi0/coderoom-ai/internal/wire"
)

func TestDeleteFileValidFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "deletefile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file to delete
	testFile := tmpDir + "/test.txt"
	testContent := "This is test content to be deleted"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify file exists before deletion
	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("Test file should exist before deletion: %v", err)
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
		modifications: true,
		root:          root,
		writeChannel:  writeChannel,
	}

	// Test deleting the test file
	argsJSON := `{"relative_path": "test.txt"}`
	tool := deleteFileTool()
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the result message
	expectedResult := "File deleted successfully:\ntest.txt"
	if result != expectedResult {
		t.Errorf("Expected result %q, got %q", expectedResult, result)
	}

	// Verify the file was actually deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("Expected file to be deleted, but it still exists")
	}

	// Verify that a tool message was sent
	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Error("Expected ToolMessage to be set")
		} else {
			expectedMsg := `delete_file("test.txt")`
			if *msg.ToolMessage != expectedMsg {
				t.Errorf("Expected tool message %q, got %q", expectedMsg, *msg.ToolMessage)
			}
		}
	default:
		t.Error("Expected a message to be sent to writeChannel")
	}
}

func TestDeleteFileNonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deletefile_test_*")
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
		modifications: true,
		root:          root,
		writeChannel:  writeChannel,
	}

	tool := deleteFileTool()
	argsJSON := `{"relative_path": "nonexistent.txt"}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "failed to delete file") {
		t.Errorf("Expected error to contain 'failed to delete file', got: %v", err)
	}

	if !strings.Contains(err.Error(), "nonexistent.txt") {
		t.Errorf("Expected error to contain filename 'nonexistent.txt', got: %v", err)
	}
}

func TestDeleteFileInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deletefile_test_*")
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
		modifications: true,
		root:          root,
		writeChannel:  writeChannel,
	}

	tool := deleteFileTool()

	// Test with invalid JSON
	argsJSON := `{invalid json}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "invalid arguments for delete_file") {
		t.Errorf("Expected error to contain 'invalid arguments for delete_file', got: %v", err)
	}
}

func TestDeleteFileMissingRequiredField(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deletefile_test_*")
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
		modifications: true,
		root:          root,
		writeChannel:  writeChannel,
	}

	tool := deleteFileTool()

	// Test with missing relative_path field
	argsJSON := `{}`
	_, err = tool.call(argsJSON, options)

	// This should error because it will try to delete an empty path
	if err == nil {
		t.Error("Expected error for missing relative_path, got nil")
	}
}

func TestDeleteFileSubdirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deletefile_test_*")
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

	// Verify file exists before deletion
	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("Test file should exist before deletion: %v", err)
	}

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	writeChannel := make(chan wire.BackendMessage, 10)
	defer close(writeChannel)

	options := &ToolOptions{
		modifications: true,
		root:          root,
		writeChannel:  writeChannel,
	}

	tool := deleteFileTool()
	argsJSON := `{"relative_path": "subdir/nested.txt"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the result message
	expectedResult := "File deleted successfully:\nsubdir/nested.txt"
	if result != expectedResult {
		t.Errorf("Expected result %q, got %q", expectedResult, result)
	}

	// Verify the file was actually deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("Expected file to be deleted, but it still exists")
	}

	// Verify that a tool message was sent with the correct path
	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Error("Expected ToolMessage to be set")
		} else {
			expectedMsg := `delete_file("subdir/nested.txt")`
			if *msg.ToolMessage != expectedMsg {
				t.Errorf("Expected tool message %q, got %q", expectedMsg, *msg.ToolMessage)
			}
		}
	default:
		t.Error("Expected a message to be sent to writeChannel")
	}
}

func TestDeleteFileEmptyPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deletefile_test_*")
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
		modifications: true,
		root:          root,
		writeChannel:  writeChannel,
	}

	tool := deleteFileTool()

	// Test with empty relative_path
	argsJSON := `{"relative_path": ""}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for empty path, got nil")
	}
}

func TestDeleteFileMultipleFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deletefile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple test files
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, fileName := range testFiles {
		testFile := tmpDir + "/" + fileName
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", fileName, err)
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
		modifications: true,
		root:          root,
		writeChannel:  writeChannel,
	}

	tool := deleteFileTool()

	// Delete each file
	for _, fileName := range testFiles {
		argsJSON := `{"relative_path": "` + fileName + `"}`
		result, err := tool.call(argsJSON, options)

		if err != nil {
			t.Fatalf("Expected no error for %s, got: %v", fileName, err)
		}

		expectedResult := "File deleted successfully:\n" + fileName
		if result != expectedResult {
			t.Errorf("Expected result %q for %s, got %q", expectedResult, fileName, result)
		}

		// Clear the channel for next iteration
		for len(writeChannel) > 0 {
			<-writeChannel
		}
	}

	// Verify all files were deleted
	for _, fileName := range testFiles {
		testFile := tmpDir + "/" + fileName
		if _, err := os.Stat(testFile); !os.IsNotExist(err) {
			t.Errorf("Expected file %s to be deleted, but it still exists", fileName)
		}
	}
}

func TestDeleteFileWithSpecialCharactersInName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deletefile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with special characters in name
	testFile := tmpDir + "/test-file_123.txt"
	testContent := "content with special chars in filename"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify file exists before deletion
	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("Test file should exist before deletion: %v", err)
	}

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	writeChannel := make(chan wire.BackendMessage, 10)
	defer close(writeChannel)

	options := &ToolOptions{
		modifications: true,
		root:          root,
		writeChannel:  writeChannel,
	}

	tool := deleteFileTool()
	argsJSON := `{"relative_path": "test-file_123.txt"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the result message
	expectedResult := "File deleted successfully:\ntest-file_123.txt"
	if result != expectedResult {
		t.Errorf("Expected result %q, got %q", expectedResult, result)
	}

	// Verify the file was actually deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("Expected file to be deleted, but it still exists")
	}
}
