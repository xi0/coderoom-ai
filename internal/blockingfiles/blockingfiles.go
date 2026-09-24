package blockingfiles

import (
	"path"
	"path/filepath"
	"strings"
)

// MatchBlockingFile reports whether relPath matches a blocking file pattern.
//
// relPath is a path relative to the project root (slash or OS separated).
//
// Matching rules:
//   - If pattern contains no "/" characters it is matched against the base name
//     of the file, so it matches files in any directory under the project root.
//   - If pattern contains one or more "/" characters it is matched against the
//     path relative to the project root. A leading "/" is accepted and treated
//     as being anchored at the project root.
//   - "*" matches any sequence of non-"/" characters and "?" matches a single
//     non-"/" character.
func MatchBlockingFile(pattern, relPath string) bool {
	relPath = filepath.ToSlash(relPath)

	if !strings.Contains(pattern, "/") {
		matched, err := path.Match(pattern, path.Base(relPath))
		return err == nil && matched
	}

	pattern = strings.TrimPrefix(pattern, "/")

	matched, err := path.Match(pattern, relPath)
	return err == nil && matched
}

func BlockingFilesInList(patterns, filenames []string) []string {
	var result []string

	for _, filename := range filenames {
		for _, pattern := range patterns {
			if MatchBlockingFile(pattern, filename) {
				result = append(result, filename)
				break
			}
		}
	}

	return result
}
