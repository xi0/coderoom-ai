package tools

import (
	"fmt"
	"path"
	"strings"
)

// sourceControlNames contains the directory (or file) names used by common
// source control systems. The agent must not touch these directly, as doing so
// could corrupt or expose the user's version history.
var sourceControlNames = map[string]struct{}{
	".git":   {},
	".svn":   {},
	".hg":    {},
	".bzr":   {},
	"_darcs": {},
	"cvs":    {},
}

// isSourceControlName reports whether name is the name of source control
// metadata. The comparison is case-insensitive and ignores the trailing dots
// and spaces that Windows strips from path components.
func isSourceControlName(name string) bool {
	name = strings.ToLower(strings.TrimRight(name, ". "))
	_, ok := sourceControlNames[name]
	return ok
}

// isSourceControlPath reports whether any component of relativePath is source
// control metadata. Both "/" and "\" are treated as separators so that a path
// cannot bypass the check on one platform by using the other platform's
// separator.
func isSourceControlPath(relativePath string) bool {
	normalized := strings.ReplaceAll(relativePath, "\\", "/")
	cleaned := path.Clean(normalized)

	if cleaned == "." || cleaned == "/" {
		return false
	}

	for _, component := range strings.Split(cleaned, "/") {
		if isSourceControlName(component) {
			return true
		}
	}

	return false
}

// sourceControlPathError returns the error used by the file tools when a path
// points into source control metadata.
func sourceControlPathError(relativePath string) error {
	return fmt.Errorf("refusing to access source control path: %s", relativePath)
}
