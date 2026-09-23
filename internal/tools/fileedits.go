package tools

import (
	"io/fs"
	"os"
	"sort"
)

// FileEdits keeps track of changes to different files during a session.
type FileEdits struct {
	root  *os.Root
	edits map[string]*FileEdit
}

// NewFileEdits initializes a FileEdits struct and returns a pointer.
func NewFileEdits(root *os.Root) *FileEdits {
	return &FileEdits{
		root:  root,
		edits: make(map[string]*FileEdit),
	}
}

// WriteFile records a write event for a file where the entire content of the
// file is written.
func (e *FileEdits) WriteFile(relativePath string, content []byte, created bool) {
	e.registerFile(relativePath)

	edit := e.edits[relativePath]
	edit.Changes = append(edit.Changes, &FileChanges{
		New:     content,
		Created: created,
	})
}

// EditFile records an edit event for a file where the old content of the file
// is replaced with the new content.
func (e *FileEdits) EditFile(relativePath string, oldContent, newContent []byte) {
	e.registerFile(relativePath)

	edit := e.edits[relativePath]
	edit.Changes = append(edit.Changes, &FileChanges{
		Old: oldContent,
		New: newContent,
	})
}

// DeleteFile records a delete event for a file.
func (e *FileEdits) DeleteFile(relativePath string) {
	e.registerFile(relativePath)

	edit := e.edits[relativePath]
	edit.Changes = append(edit.Changes, &FileChanges{
		Deleted: true,
	})
}

// registerFile checks whether the file is present in the map and if not, it
// records the original state of the file.
func (e *FileEdits) registerFile(relativePath string) {
	if _, ok := e.edits[relativePath]; ok {
		return
	}

	edit := &FileEdit{}

	info, err := e.root.Stat(relativePath)
	if err != nil {
		edit.OrigNonExistent = true
	} else {
		edit.OrigInfo = info

		if content, err := e.root.ReadFile(relativePath); err == nil {
			edit.OrigContent = content
		}
	}

	e.edits[relativePath] = edit
}

func (e *FileEdits) Filenames() []string {
	if e == nil {
		return nil
	}

	result := make([]string, len(e.edits))

	i := 0
	for f, _ := range e.edits {
		result[i] = f
		i++
	}
	sort.Strings(result)

	return result
}

// FileEdit keeps track of all changes to one file. The Orig* fields holds the
// original state of the file before it was modified.
type FileEdit struct {
	OrigNonExistent bool
	OrigInfo        fs.FileInfo
	OrigContent     []byte
	Changes         []*FileChanges
}

// FileChanges holds the detail of one single change to a file.
type FileChanges struct {
	Old     []byte
	New     []byte
	Deleted bool
	Created bool
}
