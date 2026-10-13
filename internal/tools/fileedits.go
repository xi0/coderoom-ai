package tools

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/xi0/coderoom-ai/internal/wire"

	"znkr.io/diff/textdiff"
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

// Diffs constructs a FilesMessage describing every file that was changed during
// the session together with the action that was performed on it, as well as a
// map from each filename to the EditResponse that should be served for it. The
// URI of every file points to the session scoped endpoint that serves the
// corresponding EditResponse.
//
// The content of an EditResponse holds the entire resulting file for files that
// were newly created during the session and a unified diff otherwise.
func (e *FileEdits) Diffs(sessionID string) (*wire.FilesMessage, map[string]*wire.EditResponse) {
	filenames := e.Filenames()

	files := make([]wire.File, 0, len(filenames))
	responses := make(map[string]*wire.EditResponse, len(filenames))

	for _, filename := range filenames {
		edit := e.edits[filename]

		original := edit.OrigContent
		current := edit.currentContent()

		action := actionFor(edit)

		var content string
		if action == wire.FileActionEntireFile {
			content = string(current)
		} else {
			content = textdiff.Unified(string(original), string(current))
		}

		files = append(files, wire.File{
			Filename: filename,
			Action:   action,
			URI:      fmt.Sprintf("/edits/%s/%s", sessionID, filename),
		})
		responses[filename] = &wire.EditResponse{
			Action:  action,
			Content: content,
		}
	}

	var filesMessage *wire.FilesMessage
	if len(files) > 0 {
		filesMessage = &wire.FilesMessage{
			Files: files,
		}
	}

	return filesMessage, responses
}

// actionFor reports the File action that describes the changes recorded for the
// file. A file whose last change was a deletion is reported as deleted, a file
// that did not exist before the session is reported as an entire file and any
// other change is reported as a modification.
func actionFor(edit *FileEdit) string {
	if n := len(edit.Changes); n > 0 && edit.Changes[n-1].Deleted {
		return wire.FileActionDeletedFile
	}

	if edit.OrigNonExistent {
		return wire.FileActionEntireFile
	}

	return wire.FileActionModifiedFile
}

// currentContent reconstructs the content of the file after all the recorded
// changes have been replayed on top of its original content.
func (edit *FileEdit) currentContent() []byte {
	content := edit.OrigContent

	for _, change := range edit.Changes {
		switch {
		case change.Deleted:
			content = nil
		case change.Old == nil:
			// A whole-file write replaces the entire content.
			content = change.New
		default:
			content = bytes.Replace(content, change.Old, change.New, 1)
		}
	}

	return content
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
