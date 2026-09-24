package tools

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/xi0/coderoom-ai/internal/wire"
)

func TestWriteFileCreateNewFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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

	// Test creating a new file
	testContent := "This is new file content"
	argsJSON := `{"relative_path": "newfile.txt", "content": "This is new file content"}`
	tool := writeFileTool()
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the success message
	expectedMsg := "Successfully wrote file: newfile.txt"
	if result != expectedMsg {
		t.Errorf("Expected result %q, got %q", expectedMsg, result)
	}

	// Verify that a tool message was sent
	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Error("Expected ToolMessage to be set")
		} else {
			expectedToolMsg := `write_file("newfile.txt")`
			if *msg.ToolMessage != expectedToolMsg {
				t.Errorf("Expected tool message %q, got %q", expectedToolMsg, *msg.ToolMessage)
			}
		}
	default:
		t.Error("Expected a message to be sent to writeChannel")
	}

	// Verify the file was actually created with correct content
	content, err := os.ReadFile(tmpDir + "/newfile.txt")
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content %q, got %q", testContent, string(content))
	}
}

func TestWriteFileOverwriteExistingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an existing file
	existingFile := tmpDir + "/existing.txt"
	existingContent := "original content"
	if err := os.WriteFile(existingFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
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

	tool := writeFileTool()

	// Overwrite the existing file
	newContent := "overwritten content"
	argsJSON := `{"relative_path": "existing.txt", "content": "overwritten content"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedMsg := "Successfully wrote file: existing.txt"
	if result != expectedMsg {
		t.Errorf("Expected result %q, got %q", expectedMsg, result)
	}

	// Verify the file was overwritten
	content, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("Failed to read overwritten file: %v", err)
	}

	if string(content) != newContent {
		t.Errorf("Expected file content %q, got %q", newContent, string(content))
	}
}

func TestWriteFileCreateInSubdirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Create a file in a non-existent subdirectory (should create the directory)
	testContent := "content in nested directory"
	argsJSON := `{"relative_path": "subdir/nested/file.txt", "content": "content in nested directory"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedMsg := "Successfully wrote file: subdir/nested/file.txt"
	if result != expectedMsg {
		t.Errorf("Expected result %q, got %q", expectedMsg, result)
	}

	// Verify the file was created in the subdirectory
	fullPath := tmpDir + "/subdir/nested/file.txt"
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content %q, got %q", testContent, string(content))
	}

	// Verify the directory was created
	if _, err := os.Stat(tmpDir + "/subdir/nested"); os.IsNotExist(err) {
		t.Error("Expected subdirectory to be created")
	}
}

func TestWriteFileInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Test with invalid JSON
	argsJSON := `{invalid json}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "invalid arguments for write_file") {
		t.Errorf("Expected error to contain 'invalid arguments for write_file', got: %v", err)
	}
}

func TestWriteFileMissingRequiredFields(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Test with missing relative_path field
	argsJSON := `{"content": "some content"}`
	_, err = tool.call(argsJSON, options)

	// This should error because it will try to write to an empty path
	if err == nil {
		t.Error("Expected error for missing relative_path, got nil")
	}

	// Test with missing content field
	argsJSON = `{"relative_path": "test.txt"}`
	_, err = tool.call(argsJSON, options)

	// This should still work as content can be empty string (default)
	// But let's verify it creates an empty file
	if err != nil {
		t.Fatalf("Expected no error for missing content (defaults to empty), got: %v", err)
	}

	// Verify empty file was created
	content, readErr := os.ReadFile(tmpDir + "/test.txt")
	if readErr != nil {
		t.Fatalf("Failed to read created file: %v", readErr)
	}

	if string(content) != "" {
		t.Errorf("Expected empty file content, got %q", string(content))
	}
}

func TestWriteFileEmptyContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Test with empty content
	argsJSON := `{"relative_path": "empty.txt", "content": ""}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedMsg := "Successfully wrote file: empty.txt"
	if result != expectedMsg {
		t.Errorf("Expected result %q, got %q", expectedMsg, result)
	}

	// Verify the empty file was created
	content, err := os.ReadFile(tmpDir + "/empty.txt")
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("Expected empty file, got content with length %d", len(content))
	}
}

func TestWriteFileWithSpecialCharacters(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Test with special characters including newlines, tabs, quotes, and unicode
	testContent := "Special chars: \n\t\r\"'\\<>&\nUnicode: \u00e9\u00e8\u00ea\u4e2d\u6587"
	argsJSON := `{"relative_path": "special.txt", "content": "Special chars: \n\t\r\"'\\<>&\nUnicode: \u00e9\u00e8\u00ea\u4e2d\u6587"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedMsg := "Successfully wrote file: special.txt"
	if result != expectedMsg {
		t.Errorf("Expected result %q, got %q", expectedMsg, result)
	}

	// Verify the file was created with correct content
	content, err := os.ReadFile(tmpDir + "/special.txt")
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content %q, got %q", testContent, string(content))
	}
}

func TestWriteFileLargeContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Create large content (1000 lines)
	var testContent strings.Builder
	for i := 0; i < 1000; i++ {
		testContent.WriteString("Line ")
		testContent.WriteString(string(rune(i)))
		testContent.WriteString("\n")
	}

	args := WriteFileArgs{
		RelativePath: "large.txt",
		Content:      testContent.String(),
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("json.Encode(): %v", err)
	}

	result, err := tool.call(string(argsJSON), options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedMsg := "Successfully wrote file: large.txt"
	if result != expectedMsg {
		t.Errorf("Expected result %q, got %q", expectedMsg, result)
	}

	// Verify the file was created with correct content
	content, err := os.ReadFile(tmpDir + "/large.txt")
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent.String() {
		t.Errorf("Expected file content length %d, got %d", len(testContent.String()), len(string(content)))
	}
}

func TestWriteFileWithBackslashes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Test with backslashes (Windows-style paths in content)
	testContent := "C:\\Users\\test\\file.txt\nPath: D:\\data\\config"
	argsJSON := `{"relative_path": "paths.txt", "content": "C:\\Users\\test\\file.txt\nPath: D:\\data\\config"}`
	_, err = tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the file was created with correct content
	content, err := os.ReadFile(tmpDir + "/paths.txt")
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content %q, got %q", testContent, string(content))
	}
}

func TestWriteFileWithJSONContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Test with JSON content (which contains quotes and braces)
	testContent := `{"name": "test", "value": 123, "nested": {"key": "value"}}`
	// Need to escape the JSON properly for the argsJSON
	argsJSON := `{"relative_path": "config.json", "content": "{\"name\": \"test\", \"value\": 123, \"nested\": {\"key\": \"value\"}}"}`
	_, err = tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the file was created with correct content
	content, err := os.ReadFile(tmpDir + "/config.json")
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content %q, got %q", testContent, string(content))
	}
}

func TestWriteFileDeeplyNestedDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Create a file in deeply nested directories
	testContent := "deeply nested content"
	argsJSON := `{"relative_path": "a/b/c/d/e/f/g/file.txt", "content": "deeply nested content"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedMsg := "Successfully wrote file: a/b/c/d/e/f/g/file.txt"
	if result != expectedMsg {
		t.Errorf("Expected result %q, got %q", expectedMsg, result)
	}

	// Verify the file was created
	fullPath := tmpDir + "/a/b/c/d/e/f/g/file.txt"
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content %q, got %q", testContent, string(content))
	}
}

func TestWriteFileWithMultilineContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Test with multiline content
	testContent := "Line 1\nLine 2\nLine 3\nLine 4"
	argsJSON := "{\"relative_path\": \"multiline.txt\", \"content\": \"Line 1\\nLine 2\\nLine 3\\nLine 4\"}"
	_, err = tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the file was created with correct content
	content, err := os.ReadFile(tmpDir + "/multiline.txt")
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content %q, got %q", testContent, string(content))
	}
}

func TestWriteFileRootDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	// Test creating a file in the root directory
	testContent := "root level file"
	argsJSON := `{"relative_path": "rootfile.txt", "content": "root level file"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expectedMsg := "Successfully wrote file: rootfile.txt"
	if result != expectedMsg {
		t.Errorf("Expected result %q, got %q", expectedMsg, result)
	}

	// Verify the file was created
	content, err := os.ReadFile(tmpDir + "/rootfile.txt")
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected file content %q, got %q", testContent, string(content))
	}
}

func TestWriteFileRecordsCreationOfNewFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()
	argsJSON := `{"relative_path": "recorded.txt", "content": "recorded content"}`
	if _, err := tool.call(argsJSON, options); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	edit, ok := edits.edits["recorded.txt"]
	if !ok {
		t.Fatal("Expected write_file to record an edit for recorded.txt")
	}
	if !edit.OrigNonExistent {
		t.Error("Expected OrigNonExistent to be true for a newly created file")
	}
	if edit.OrigInfo != nil {
		t.Error("Expected OrigInfo to be nil for a newly created file")
	}
	if len(edit.OrigContent) != 0 {
		t.Errorf("Expected no original content, got %q", edit.OrigContent)
	}
	if len(edit.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(edit.Changes))
	}

	change := edit.Changes[0]
	if !change.Created {
		t.Error("Expected change.Created to be true")
	}
	if change.Deleted {
		t.Error("Expected change.Deleted to be false")
	}
	if string(change.New) != "recorded content" {
		t.Errorf("Expected change.New %q, got %q", "recorded content", change.New)
	}
}

func TestWriteFileRecordsOverwriteOfExistingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalContent := "original content"
	if err := os.WriteFile(tmpDir+"/existing.txt", []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
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

	tool := writeFileTool()
	argsJSON := `{"relative_path": "existing.txt", "content": "overwritten content"}`
	if _, err := tool.call(argsJSON, options); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	edit, ok := edits.edits["existing.txt"]
	if !ok {
		t.Fatal("Expected write_file to record an edit for existing.txt")
	}
	if edit.OrigNonExistent {
		t.Error("Expected OrigNonExistent to be false for an existing file")
	}
	if edit.OrigInfo == nil {
		t.Fatal("Expected OrigInfo to be recorded for an existing file")
	}
	if edit.OrigInfo.Size() != int64(len(originalContent)) {
		t.Errorf("Expected OrigInfo.Size() %d, got %d", len(originalContent), edit.OrigInfo.Size())
	}
	if string(edit.OrigContent) != originalContent {
		t.Errorf("Expected OrigContent %q, got %q", originalContent, edit.OrigContent)
	}
	if len(edit.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(edit.Changes))
	}

	change := edit.Changes[0]
	if change.Created {
		t.Error("Expected change.Created to be false for an overwrite")
	}
	if string(change.New) != "overwritten content" {
		t.Errorf("Expected change.New %q, got %q", "overwritten content", change.New)
	}
}

func TestWriteFileRecordsMultipleWrites(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "writefile_test_*")
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

	tool := writeFileTool()

	writes := []string{"first", "second", "third"}
	for _, content := range writes {
		argsJSON, err := json.Marshal(WriteFileArgs{
			RelativePath: "multi.txt",
			Content:      content,
		})
		if err != nil {
			t.Fatalf("json.Marshal(): %v", err)
		}
		if _, err := tool.call(string(argsJSON), options); err != nil {
			t.Fatalf("Expected no error writing %q, got: %v", content, err)
		}
	}

	edit, ok := edits.edits["multi.txt"]
	if !ok {
		t.Fatal("Expected write_file to record an edit for multi.txt")
	}
	// The original state is registered before the first write and must not be
	// overwritten by later writes.
	if !edit.OrigNonExistent {
		t.Error("Expected OrigNonExistent to be true for a file that did not exist")
	}
	if len(edit.Changes) != len(writes) {
		t.Fatalf("Expected %d changes, got %d", len(writes), len(edit.Changes))
	}
	if !edit.Changes[0].Created {
		t.Error("Expected the first write to be marked as created")
	}
	if edit.Changes[1].Created || edit.Changes[2].Created {
		t.Error("Expected subsequent writes not to be marked as created")
	}
	for i, want := range writes {
		if string(edit.Changes[i].New) != want {
			t.Errorf("Change %d: expected New %q, got %q", i, want, edit.Changes[i].New)
		}
	}
}
