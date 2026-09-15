package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/xi0/coderoom-ai/internal/wire"
)

func TestGrepValidPatternWithMatches(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with content
	testFile1 := tmpDir + "/test1.txt"
	testContent1 := "This is line 1\nThis contains pattern\nThis is line 3"
	if err := os.WriteFile(testFile1, []byte(testContent1), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}

	testFile2 := tmpDir + "/test2.txt"
	testContent2 := "No match here\nAnother line with pattern\nFinal line"
	if err := os.WriteFile(testFile2, []byte(testContent2), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
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

	// Test grep with pattern that should match
	argsJSON := `{"pattern": "pattern"}`
	tool := grepTool()
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the result contains matches
	if !strings.Contains(result, "Found 2 match(es)") {
		t.Errorf("Expected result to contain 'Found 2 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "test1.txt:2:This contains pattern") {
		t.Errorf("Expected result to contain match from test1.txt, got: %s", result)
	}

	if !strings.Contains(result, "test2.txt:2:Another line with pattern") {
		t.Errorf("Expected result to contain match from test2.txt, got: %s", result)
	}

	// Verify that a tool message was sent
	select {
	case msg := <-writeChannel:
		if msg.ToolMessage == nil {
			t.Error("Expected ToolMessage to be set")
		} else {
			expectedMsg := `grep("pattern")`
			if *msg.ToolMessage != expectedMsg {
				t.Errorf("Expected tool message %q, got %q", expectedMsg, *msg.ToolMessage)
			}
		}
	default:
		t.Error("Expected a message to be sent to writeChannel")
	}
}

func TestGrepNoMatchesFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file with content
	testFile := tmpDir + "/test.txt"
	testContent := "This is line 1\nThis is line 2\nThis is line 3"
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

	tool := grepTool()
	argsJSON := `{"pattern": "nonexistent"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the result indicates no matches
	if result != "No matches found." {
		t.Errorf("Expected 'No matches found.', got: %s", result)
	}
}

func TestGrepCaseInsensitive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file with mixed case content
	testFile := tmpDir + "/test.txt"
	testContent := "This is PATTERN\nThis is pattern\nThis is Pattern"
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

	tool := grepTool()
	// Search for lowercase but should match all cases
	argsJSON := `{"pattern": "pattern", "case_insensitive": true}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should find all 3 matches
	if !strings.Contains(result, "Found 3 match(es)") {
		t.Errorf("Expected result to contain 'Found 3 match(es)', got: %s", result)
	}
}

func TestGrepCaseSensitive(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file with mixed case content
	testFile := tmpDir + "/test.txt"
	testContent := "This is PATTERN\nThis is pattern\nThis is Pattern"
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

	tool := grepTool()
	// Search for lowercase, should only match lowercase
	argsJSON := `{"pattern": "pattern"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should find only 1 match (case-sensitive by default)
	if !strings.Contains(result, "Found 1 match(es)") {
		t.Errorf("Expected result to contain 'Found 1 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "This is pattern") {
		t.Errorf("Expected result to contain exact case match, got: %s", result)
	}
}

func TestGrepInvalidRegex(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
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

	tool := grepTool()
	// Invalid regex pattern (unclosed bracket)
	argsJSON := `{"pattern": "[invalid"}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for invalid regex, got nil")
	}

	if !strings.Contains(err.Error(), "invalid regexp pattern") {
		t.Errorf("Expected error to contain 'invalid regexp pattern', got: %v", err)
	}
}

func TestGrepEmptyPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
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

	tool := grepTool()
	argsJSON := `{"pattern": ""}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for empty pattern, got nil")
	}

	if !strings.Contains(err.Error(), "pattern cannot be empty") {
		t.Errorf("Expected error to contain 'pattern cannot be empty', got: %v", err)
	}
}

func TestGrepInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
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

	tool := grepTool()

	// Test with invalid JSON
	argsJSON := `{invalid json}`
	_, err = tool.call(argsJSON, options)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "invalid arguments for grep") {
		t.Errorf("Expected error to contain 'invalid arguments for grep', got: %v", err)
	}
}

func TestGrepInSubdirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory
	testSubdir := tmpDir + "/subdir"
	if err := os.Mkdir(testSubdir, 0755); err != nil {
		t.Fatalf("Failed to create test subdirectory: %v", err)
	}

	// Create test files in both root and subdirectory
	testFileRoot := tmpDir + "/root.txt"
	testContentRoot := "pattern in root"
	if err := os.WriteFile(testFileRoot, []byte(testContentRoot), 0644); err != nil {
		t.Fatalf("Failed to create test file in root: %v", err)
	}

	testFileSubdir := testSubdir + "/nested.txt"
	testContentSubdir := "pattern in subdir"
	if err := os.WriteFile(testFileSubdir, []byte(testContentSubdir), 0644); err != nil {
		t.Fatalf("Failed to create test file in subdir: %v", err)
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

	tool := grepTool()
	// Search only in subdirectory
	argsJSON := `{"pattern": "pattern", "relative_path": "subdir"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only find match in subdir
	if !strings.Contains(result, "Found 1 match(es)") {
		t.Errorf("Expected result to contain 'Found 1 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "subdir/nested.txt") {
		t.Errorf("Expected result to contain path to subdir file, got: %s", result)
	}

	if strings.Contains(result, "root.txt") {
		t.Errorf("Expected result to not contain root.txt when searching only in subdir, got: %s", result)
	}
}

func TestGrepSkipHiddenFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create regular file and hidden file
	testFile := tmpDir + "/test.txt"
	testContent := "pattern in regular file"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hiddenFile := tmpDir + "/.hidden.txt"
	hiddenContent := "pattern in hidden file"
	if err := os.WriteFile(hiddenFile, []byte(hiddenContent), 0644); err != nil {
		t.Fatalf("Failed to create hidden test file: %v", err)
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

	tool := grepTool()
	argsJSON := `{"pattern": "pattern"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only find match in regular file
	if !strings.Contains(result, "Found 1 match(es)") {
		t.Errorf("Expected result to contain 'Found 1 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "test.txt") {
		t.Errorf("Expected result to contain test.txt, got: %s", result)
	}

	if strings.Contains(result, ".hidden.txt") {
		t.Errorf("Expected result to not contain hidden file, got: %s", result)
	}
}

func TestGrepSkipHiddenDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create regular directory and hidden directory
	testSubdir := tmpDir + "/subdir"
	if err := os.Mkdir(testSubdir, 0755); err != nil {
		t.Fatalf("Failed to create test subdirectory: %v", err)
	}

	hiddenSubdir := tmpDir + "/.hidden_dir"
	if err := os.Mkdir(hiddenSubdir, 0755); err != nil {
		t.Fatalf("Failed to create hidden test subdirectory: %v", err)
	}

	// Create test files in both directories
	testFile := testSubdir + "/test.txt"
	testContent := "pattern in regular dir"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hiddenFile := hiddenSubdir + "/hidden.txt"
	hiddenContent := "pattern in hidden dir"
	if err := os.WriteFile(hiddenFile, []byte(hiddenContent), 0644); err != nil {
		t.Fatalf("Failed to create hidden test file: %v", err)
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

	tool := grepTool()
	argsJSON := `{"pattern": "pattern"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only find match in regular directory
	if !strings.Contains(result, "Found 1 match(es)") {
		t.Errorf("Expected result to contain 'Found 1 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "subdir/test.txt") {
		t.Errorf("Expected result to contain subdir/test.txt, got: %s", result)
	}

	if strings.Contains(result, ".hidden_dir") {
		t.Errorf("Expected result to not contain hidden directory, got: %s", result)
	}
}

func TestGrepSkipBinaryFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create regular text file
	testFile := tmpDir + "/test.txt"
	testContent := "pattern in text file"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create binary file with null bytes
	binaryFile := tmpDir + "/binary.bin"
	binaryContent := []byte{0x00, 0x01, 0x02, 'p', 'a', 't', 't', 'e', 'r', 'n', 0x00}
	if err := os.WriteFile(binaryFile, binaryContent, 0644); err != nil {
		t.Fatalf("Failed to create binary test file: %v", err)
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

	tool := grepTool()
	argsJSON := `{"pattern": "pattern"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only find match in text file
	if !strings.Contains(result, "Found 1 match(es)") {
		t.Errorf("Expected result to contain 'Found 1 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "test.txt") {
		t.Errorf("Expected result to contain test.txt, got: %s", result)
	}

	if strings.Contains(result, "binary.bin") {
		t.Errorf("Expected result to not contain binary file, got: %s", result)
	}
}

func TestGrepSkipCommonDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create common directories that should be skipped
	nodeModules := tmpDir + "/node_modules"
	if err := os.Mkdir(nodeModules, 0755); err != nil {
		t.Fatalf("Failed to create node_modules: %v", err)
	}

	vendor := tmpDir + "/vendor"
	if err := os.Mkdir(vendor, 0755); err != nil {
		t.Fatalf("Failed to create vendor: %v", err)
	}

	gitDir := tmpDir + "/.git"
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git: %v", err)
	}

	// Create files with pattern in these directories
	nodeModulesFile := nodeModules + "/module.js"
	if err := os.WriteFile(nodeModulesFile, []byte("pattern in node_modules"), 0644); err != nil {
		t.Fatalf("Failed to create file in node_modules: %v", err)
	}

	vendorFile := vendor + "/vendor.go"
	if err := os.WriteFile(vendorFile, []byte("pattern in vendor"), 0644); err != nil {
		t.Fatalf("Failed to create file in vendor: %v", err)
	}

	gitFile := gitDir + "/config"
	if err := os.WriteFile(gitFile, []byte("pattern in .git"), 0644); err != nil {
		t.Fatalf("Failed to create file in .git: %v", err)
	}

	// Create regular directory with file
	regularDir := tmpDir + "/src"
	if err := os.Mkdir(regularDir, 0755); err != nil {
		t.Fatalf("Failed to create src: %v", err)
	}

	regularFile := regularDir + "/main.go"
	if err := os.WriteFile(regularFile, []byte("pattern in src"), 0644); err != nil {
		t.Fatalf("Failed to create file in src: %v", err)
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

	tool := grepTool()
	argsJSON := `{"pattern": "pattern"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only find match in regular directory
	if !strings.Contains(result, "Found 1 match(es)") {
		t.Errorf("Expected result to contain 'Found 1 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "src/main.go") {
		t.Errorf("Expected result to contain src/main.go, got: %s", result)
	}

	if strings.Contains(result, "node_modules") {
		t.Errorf("Expected result to not contain node_modules, got: %s", result)
	}

	if strings.Contains(result, "vendor") {
		t.Errorf("Expected result to not contain vendor, got: %s", result)
	}

	if strings.Contains(result, ".git") {
		t.Errorf("Expected result to not contain .git, got: %s", result)
	}
}

func TestGrepRegexPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file with content
	testFile := tmpDir + "/test.txt"
	testContent := "email1@example.com\nemail2@test.org\nnot an email"
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

	tool := grepTool()
	// Use regex pattern to match emails
	argsJSON := `{"pattern": "[a-z0-9]+@[a-z]+\\.[a-z]+"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should find 2 email matches
	if !strings.Contains(result, "Found 2 match(es)") {
		t.Errorf("Expected result to contain 'Found 2 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "email1@example.com") {
		t.Errorf("Expected result to contain first email, got: %s", result)
	}

	if !strings.Contains(result, "email2@test.org") {
		t.Errorf("Expected result to contain second email, got: %s", result)
	}
}

func TestGrepMultipleMatchesInSameFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file with multiple matches
	testFile := tmpDir + "/test.txt"
	testContent := "line with pattern\nline without\nanother pattern line\npattern again"
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

	tool := grepTool()
	argsJSON := `{"pattern": "pattern"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should find 3 matches
	if !strings.Contains(result, "Found 3 match(es)") {
		t.Errorf("Expected result to contain 'Found 3 match(es)', got: %s", result)
	}

	// Verify line numbers are correct (1-indexed)
	if !strings.Contains(result, "test.txt:1:line with pattern") {
		t.Errorf("Expected result to contain match on line 1, got: %s", result)
	}

	if !strings.Contains(result, "test.txt:3:another pattern line") {
		t.Errorf("Expected result to contain match on line 3, got: %s", result)
	}

	if !strings.Contains(result, "test.txt:4:pattern again") {
		t.Errorf("Expected result to contain match on line 4, got: %s", result)
	}
}

func TestGrepEmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
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

	tool := grepTool()
	argsJSON := `{"pattern": "anything"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Empty directory should return no matches
	if result != "No matches found." {
		t.Errorf("Expected 'No matches found.', got: %s", result)
	}
}

func TestGrepWithRelativePathDot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := tmpDir + "/test.txt"
	testContent := "pattern here"
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

	tool := grepTool()
	// Test with explicit "." as relative path
	argsJSON := `{"pattern": "pattern", "relative_path": "."}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.Contains(result, "Found 1 match(es)") {
		t.Errorf("Expected result to contain 'Found 1 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "test.txt") {
		t.Errorf("Expected result to contain test.txt, got: %s", result)
	}
}

func TestGrepWithEmptyRelativePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := tmpDir + "/test.txt"
	testContent := "pattern here"
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

	tool := grepTool()
	// Test with empty relative_path (should default to root)
	argsJSON := `{"pattern": "pattern", "relative_path": ""}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !strings.Contains(result, "Found 1 match(es)") {
		t.Errorf("Expected result to contain 'Found 1 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "test.txt") {
		t.Errorf("Expected result to contain test.txt, got: %s", result)
	}
}

func TestGrepSkipCVSDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create CVS directory that should be skipped
	cvsDir := tmpDir + "/CVS"
	if err := os.Mkdir(cvsDir, 0755); err != nil {
		t.Fatalf("Failed to create CVS: %v", err)
	}

	// Create file with pattern in CVS directory
	cvsFile := cvsDir + "/entries"
	if err := os.WriteFile(cvsFile, []byte("pattern in CVS"), 0644); err != nil {
		t.Fatalf("Failed to create file in CVS: %v", err)
	}

	// Create regular directory with file
	regularDir := tmpDir + "/src"
	if err := os.Mkdir(regularDir, 0755); err != nil {
		t.Fatalf("Failed to create src: %v", err)
	}

	regularFile := regularDir + "/main.go"
	if err := os.WriteFile(regularFile, []byte("pattern in src"), 0644); err != nil {
		t.Fatalf("Failed to create file in src: %v", err)
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

	tool := grepTool()
	argsJSON := `{"pattern": "pattern"}`
	result, err := tool.call(argsJSON, options)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only find match in regular directory
	if !strings.Contains(result, "Found 1 match(es)") {
		t.Errorf("Expected result to contain 'Found 1 match(es)', got: %s", result)
	}

	if !strings.Contains(result, "src/main.go") {
		t.Errorf("Expected result to contain src/main.go, got: %s", result)
	}

	if strings.Contains(result, "CVS") {
		t.Errorf("Expected result to not contain CVS, got: %s", result)
	}
}
