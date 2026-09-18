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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
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
		Modifications: true,
		Root:          root,
		WriteChannel:  writeChannel,
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

func TestDeleteFileRecordsDeletion(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deletefile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalContent := "content to be deleted"
	testFile := tmpDir + "/test.txt"
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

	edits := NewFileEdits(root)
	options := &ToolOptions{
		Modifications: true,
		Edits:         edits,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := deleteFileTool()
	argsJSON := `{"relative_path": "test.txt"}`
	if _, err := tool.call(argsJSON, options); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	edit, ok := edits.edits["test.txt"]
	if !ok {
		t.Fatal("Expected delete_file to record an edit for test.txt")
	}
	if edit.OrigNonExistent {
		t.Error("Expected OrigNonExistent to be false for an existing file")
	}
	if edit.OrigInfo == nil {
		t.Fatal("Expected OrigInfo to be recorded for an existing file")
	}
	// The original content must be preserved even though the file was deleted.
	if string(edit.OrigContent) != originalContent {
		t.Errorf("Expected OrigContent %q, got %q", originalContent, edit.OrigContent)
	}
	if len(edit.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(edit.Changes))
	}

	change := edit.Changes[0]
	if !change.Deleted {
		t.Error("Expected change.Deleted to be true")
	}
	if change.Created {
		t.Error("Expected change.Created to be false")
	}
}

func TestDeleteFileRecordsDeletionInSubdirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "deletefile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.Mkdir(tmpDir+"/subdir", 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	originalContent := "nested content"
	testFile := tmpDir + "/subdir/nested.txt"
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

	edits := NewFileEdits(root)
	options := &ToolOptions{
		Modifications: true,
		Edits:         edits,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := deleteFileTool()
	argsJSON := `{"relative_path": "subdir/nested.txt"}`
	if _, err := tool.call(argsJSON, options); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	edit, ok := edits.edits["subdir/nested.txt"]
	if !ok {
		t.Fatalf("Expected delete_file to record an edit for subdir/nested.txt, got entries: %v", edits.edits)
	}
	if string(edit.OrigContent) != originalContent {
		t.Errorf("Expected OrigContent %q, got %q", originalContent, edit.OrigContent)
	}
	if len(edit.Changes) != 1 || !edit.Changes[0].Deleted {
		t.Error("Expected a single delete change")
	}
}

func TestDeleteFileNotRecordedWhenRemovalFails(t *testing.T) {
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

	edits := NewFileEdits(root)
	options := &ToolOptions{
		Modifications: true,
		Edits:         edits,
		Root:          root,
		WriteChannel:  writeChannel,
	}

	tool := deleteFileTool()
	argsJSON := `{"relative_path": "missing.txt"}`
	if _, err := tool.call(argsJSON, options); err == nil {
		t.Fatal("Expected an error deleting a non-existent file")
	}

	// A failed deletion must not record a change, even though the original
	// (non-existent) state may have been registered before the attempt.
	if edit, ok := edits.edits["missing.txt"]; ok && len(edit.Changes) != 0 {
		t.Errorf("Expected no changes to be recorded on failure, got %d", len(edit.Changes))
	}
}
