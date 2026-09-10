package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type GrepArgs struct {
	Pattern         string `json:"pattern"`
	RelativePath    string `json:"relative_path,omitempty"`
	CaseInsensitive bool   `json:"case_insensitive,omitempty"`
}

func grepTool() *Tool {
	return &Tool{
		mutating: true,
		definition: &openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "grep",
				Description: "Searches files in the project directory for occurrences of a regexp pattern. Returns matching lines with file paths and line numbers.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"pattern": {
							"type": "string",
							"description": "Regular expression pattern to search for"
						},
						"relative_path": {
							"type": "string",
							"description": "Optional relative path to search within (defaults to project root)"
						},
						"case_insensitive": {
							"type": "boolean",
							"description": "Optional flag to perform case-insensitive search (defaults to false)"
						}
					},
					"required": ["pattern"]
				}`),
			},
		},
		handler: func(argsJSON string, options *ToolOptions) (string, error) {
			var args GrepArgs
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("invalid arguments for grep: %w", err)
			}

			toolString := fmt.Sprintf("grep(%q)", args.Pattern)
			options.writeChannel <- wire.BackendMessage{
				ToolMessage: &toolString,
			}

			if args.Pattern == "" {
				return "", fmt.Errorf("pattern cannot be empty")
			}

			if args.RelativePath == "" {
				args.RelativePath = "."
			}

			// Compile the regexp
			var re *regexp.Regexp
			var err error
			if args.CaseInsensitive {
				re, err = regexp.Compile("(?i)" + args.Pattern)
			} else {
				re, err = regexp.Compile(args.Pattern)
			}
			if err != nil {
				return "", fmt.Errorf("invalid regexp pattern: %w", err)
			}

			var results []string
			err = fs.WalkDir(options.root.FS(), args.RelativePath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Skip directories
				if d.IsDir() {
					// Skip hidden directories and common non-source directories
					name := d.Name()
					if name == "." {
						return nil
					}
					if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == ".git" || name == "CVS" {
						return fs.SkipDir
					}
					return nil
				}

				// Skip hidden files
				name := d.Name()
				if strings.HasPrefix(name, ".") {
					return nil
				}

				// Read file content
				content, err := options.root.ReadFile(path)
				if err != nil {
					// Skip files that can't be read
					return nil
				}

				// Skip binary files (simple heuristic: check for null bytes)
				if bytes.IndexByte(content, 0) != -1 {
					return nil
				}

				// Search for matches line by line
				lines := strings.Split(string(content), "\n")
				for lineNum, line := range lines {
					if re.MatchString(line) {
						results = append(results, fmt.Sprintf("%s:%d:%s", path, lineNum+1, line))
					}
				}

				return nil
			})

			if err != nil {
				return "", fmt.Errorf("error walking directory: %w", err)
			}

			if len(results) == 0 {
				return "No matches found.", nil
			}

			return fmt.Sprintf("Found %d match(es):\n%s", len(results), strings.Join(results, "\n")), nil
		},
	}
}
