package tools

import (
	"os"
	"testing"
)

// newTestFileEdits creates a temporary directory, opens it as a root and
// returns a FileEdits instance bound to that root together with the temporary
// directory path. The root and directory are cleaned up when the test ends.
func newTestFileEdits(t *testing.T) (*FileEdits, string) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "fileedits_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	root, err := os.OpenRoot(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open root: %v", err)
	}

	t.Cleanup(func() {
		root.Close()
		os.RemoveAll(tmpDir)
	})

	return NewFileEdits(root), tmpDir
}

func TestNewFileEdits(t *testing.T) {
	fe, _ := newTestFileEdits(t)

	if fe.root == nil {
		t.Error("Expected root to be set")
	}
	if fe.edits == nil {
		t.Fatal("Expected edits map to be initialised")
	}
	if len(fe.edits) != 0 {
		t.Errorf("Expected an empty edits map, got %d entries", len(fe.edits))
	}
}

func TestFileEditsWriteFileNewFile(t *testing.T) {
	fe, _ := newTestFileEdits(t)

	content := []byte("brand new content")
	fe.WriteFile("new.txt", content, true)

	edit, ok := fe.edits["new.txt"]
	if !ok {
		t.Fatal("Expected an entry for new.txt")
	}
	if !edit.OrigNonExistent {
		t.Error("Expected OrigNonExistent to be true for a new file")
	}
	if edit.OrigInfo != nil {
		t.Error("Expected OrigInfo to be nil for a new file")
	}
	if edit.OrigContent != nil {
		t.Errorf("Expected OrigContent to be nil for a new file, got %q", edit.OrigContent)
	}
	if len(edit.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(edit.Changes))
	}

	change := edit.Changes[0]
	if !change.Created {
		t.Error("Expected Created to be true")
	}
	if change.Deleted {
		t.Error("Expected Deleted to be false")
	}
	if string(change.New) != string(content) {
		t.Errorf("Expected New %q, got %q", content, change.New)
	}
	if change.Old != nil {
		t.Errorf("Expected Old to be nil, got %q", change.Old)
	}
}

func TestFileEditsWriteFileExistingFile(t *testing.T) {
	fe, tmpDir := newTestFileEdits(t)

	original := []byte("original content")
	if err := os.WriteFile(tmpDir+"/existing.txt", original, 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	newContent := []byte("new content")
	fe.WriteFile("existing.txt", newContent, false)

	edit, ok := fe.edits["existing.txt"]
	if !ok {
		t.Fatal("Expected an entry for existing.txt")
	}
	if edit.OrigNonExistent {
		t.Error("Expected OrigNonExistent to be false for an existing file")
	}
	if edit.OrigInfo == nil {
		t.Fatal("Expected OrigInfo to be recorded for an existing file")
	}
	if edit.OrigInfo.Size() != int64(len(original)) {
		t.Errorf("Expected OrigInfo.Size() %d, got %d", len(original), edit.OrigInfo.Size())
	}
	if string(edit.OrigContent) != string(original) {
		t.Errorf("Expected OrigContent %q, got %q", original, edit.OrigContent)
	}
	if len(edit.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(edit.Changes))
	}

	change := edit.Changes[0]
	if change.Created {
		t.Error("Expected Created to be false for an existing file")
	}
	if string(change.New) != string(newContent) {
		t.Errorf("Expected New %q, got %q", newContent, change.New)
	}
}

func TestFileEditsEditFile(t *testing.T) {
	fe, tmpDir := newTestFileEdits(t)

	original := []byte("hello world")
	if err := os.WriteFile(tmpDir+"/edit.txt", original, 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	oldString := []byte("hello")
	newString := []byte("goodbye")
	fe.EditFile("edit.txt", oldString, newString)

	edit, ok := fe.edits["edit.txt"]
	if !ok {
		t.Fatal("Expected an entry for edit.txt")
	}
	if edit.OrigNonExistent {
		t.Error("Expected OrigNonExistent to be false for an existing file")
	}
	if string(edit.OrigContent) != string(original) {
		t.Errorf("Expected OrigContent %q, got %q", original, edit.OrigContent)
	}
	if len(edit.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(edit.Changes))
	}

	change := edit.Changes[0]
	if string(change.Old) != string(oldString) {
		t.Errorf("Expected Old %q, got %q", oldString, change.Old)
	}
	if string(change.New) != string(newString) {
		t.Errorf("Expected New %q, got %q", newString, change.New)
	}
	if change.Deleted || change.Created {
		t.Error("Expected Deleted and Created to be false for an edit")
	}
}

func TestFileEditsDeleteFile(t *testing.T) {
	fe, tmpDir := newTestFileEdits(t)

	original := []byte("content to delete")
	if err := os.WriteFile(tmpDir+"/delete.txt", original, 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	fe.DeleteFile("delete.txt")

	edit, ok := fe.edits["delete.txt"]
	if !ok {
		t.Fatal("Expected an entry for delete.txt")
	}
	if edit.OrigNonExistent {
		t.Error("Expected OrigNonExistent to be false for an existing file")
	}
	if string(edit.OrigContent) != string(original) {
		t.Errorf("Expected OrigContent %q, got %q", original, edit.OrigContent)
	}
	if len(edit.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(edit.Changes))
	}

	change := edit.Changes[0]
	if !change.Deleted {
		t.Error("Expected Deleted to be true")
	}
	if change.Created {
		t.Error("Expected Created to be false")
	}
}

func TestFileEditsDeleteNonExistentFile(t *testing.T) {
	fe, _ := newTestFileEdits(t)

	fe.DeleteFile("missing.txt")

	edit, ok := fe.edits["missing.txt"]
	if !ok {
		t.Fatal("Expected an entry for missing.txt")
	}
	if !edit.OrigNonExistent {
		t.Error("Expected OrigNonExistent to be true for a missing file")
	}
	if edit.OrigInfo != nil {
		t.Error("Expected OrigInfo to be nil for a missing file")
	}
	if len(edit.Changes) != 1 || !edit.Changes[0].Deleted {
		t.Error("Expected a single delete change")
	}
}

func TestFileEditsRegistersOriginalStateOnlyOnce(t *testing.T) {
	fe, _ := newTestFileEdits(t)

	fe.WriteFile("once.txt", []byte("first"), true)
	fe.WriteFile("once.txt", []byte("second"), false)

	edit, ok := fe.edits["once.txt"]
	if !ok {
		t.Fatal("Expected an entry for once.txt")
	}
	// The file did not exist when it was first registered, so that remains the
	// recorded original state even after subsequent writes.
	if !edit.OrigNonExistent {
		t.Error("Expected the original state to be recorded only once")
	}
	if edit.OrigContent != nil {
		t.Errorf("Expected OrigContent to stay nil, got %q", edit.OrigContent)
	}
	if len(edit.Changes) != 2 {
		t.Fatalf("Expected 2 changes, got %d", len(edit.Changes))
	}
	if string(edit.Changes[0].New) != "first" || string(edit.Changes[1].New) != "second" {
		t.Error("Expected changes to be recorded in order")
	}
}

func TestFileEditsTracksMultipleFilesIndependently(t *testing.T) {
	fe, tmpDir := newTestFileEdits(t)

	if err := os.WriteFile(tmpDir+"/a.txt", []byte("a content"), 0644); err != nil {
		t.Fatalf("Failed to create a.txt: %v", err)
	}

	fe.WriteFile("a.txt", []byte("a updated"), false)
	fe.WriteFile("b.txt", []byte("b content"), true)

	if len(fe.edits) != 2 {
		t.Fatalf("Expected 2 tracked files, got %d", len(fe.edits))
	}

	a := fe.edits["a.txt"]
	if a.OrigNonExistent {
		t.Error("Expected a.txt to be recorded as pre-existing")
	}
	if string(a.OrigContent) != "a content" {
		t.Errorf("Expected a.txt OrigContent %q, got %q", "a content", a.OrigContent)
	}

	b := fe.edits["b.txt"]
	if !b.OrigNonExistent {
		t.Error("Expected b.txt to be recorded as non-existent")
	}
	if len(b.Changes) != 1 || !b.Changes[0].Created {
		t.Error("Expected b.txt to have a single created change")
	}
}

func TestFileEditsDirectoryHasNoOriginalContent(t *testing.T) {
	fe, tmpDir := newTestFileEdits(t)

	if err := os.Mkdir(tmpDir+"/adir", 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// A directory exists so OrigInfo is recorded, but reading it as a file
	// fails, so OrigContent stays nil.
	fe.DeleteFile("adir")

	edit, ok := fe.edits["adir"]
	if !ok {
		t.Fatal("Expected an entry for adir")
	}
	if edit.OrigNonExistent {
		t.Error("Expected OrigNonExistent to be false for an existing directory")
	}
	if edit.OrigInfo == nil {
		t.Error("Expected OrigInfo to be recorded for an existing directory")
	}
	if !edit.OrigInfo.IsDir() {
		t.Error("Expected the recorded OrigInfo to describe a directory")
	}
	if edit.OrigContent != nil {
		t.Errorf("Expected OrigContent to be nil for a directory, got %q", edit.OrigContent)
	}
}
