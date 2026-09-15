package backend

import (
	"io/fs"
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

// matchingFiles returns the paths of all regular files under the project
// directory that match the given pattern. The returned paths are relative to
// the project root and use "/" as separator. The .git directory is skipped.
func (s *Settings) matchingFiles(pattern string) ([]string, error) {
	root := s.ProjectDir
	matches := []string{}

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}

		if MatchBlockingFile(pattern, rel) {
			matches = append(matches, filepath.ToSlash(rel))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return matches, nil
}
