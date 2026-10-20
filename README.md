# Coderoom AI
A simple but powerful AI code assistant with a web based UI.

## About Coderoom AI
Coderoom AI is a simple AI code assistant that provides intelligent code suggestions and assistance through a user-friendly web-based interface.

## Background
This project was created to help developers write code by leveraging AI-powered assistance directly in their browser.

### Key design considerations

One of the primary goals of this project is to provide a UI that is more friendly to work with than a CLI based agent.

Most of the other design considerations are centered around security.

* Don't ask the user lots of difficult questions (e.g. "Do you want to run `find . -name "*.log" -exec grep -l "ERR" {} + | xargs awk -F: '{print $1, $3}'`?) in order to prevent confirmation fatigue.
* Make relevant tools only run if a configured list of files have not been touched by the agent. (E.g. dont allow "build_project" if Makefile has been changed).
* Prevent the model from touching source control related files directly.
* Provide easy rollback mechanisms.
* Use the Go built-in os.Root type for restricting all file operations to within the project directory.

Additionally, we will strive to keep the number of external dependencies as low as possible.

The tool consists of a single binary containing everything needed to run and serve its web inteface in the local browser.

## Project Structure

```
coderoom-ai/
├── cmd/
│   └── coderoom-ai/          # Main application entry point (flags, static assets, HTTP routes)
├── internal/                 # Private application code
│   ├── backend/              # Backends (OpenAI, test), WebSocket server, settings, edit store
│   ├── blockingfiles/        # Blocking file pattern matching
│   ├── browser/              # Browser interaction layer (DOM objects, WebSocket) for WASM
│   ├── common/               # Static provider and model catalog
│   ├── tools/                # Tools executed by the agent loop
│   ├── ui/                   # UI logic and message handling (WASM)
│   └── wire/                 # Message and settings definitions shared by frontend and backend
├── web/
│   ├── assets/               # Static web assets (index.html, styles.css, logo.png)
│   ├── dist/                 # Compiled WASM and JavaScript files
│   ├── wasm/                 # WebAssembly source code
│   └── web.go                # Embedded web resources
├── bin/                      # Compiled binaries
├── .gitignore                # Ignored build output (bin/, web/dist/)
├── coderoom-ai.json          # Project settings for this repository
├── copy_wasm_exec.go         # Utility to copy the WASM exec file
├── Makefile                  # Build automation
├── build.bat                 # Build script for Windows
├── runtests.bat              # Test script for Windows
├── go.mod                    # Go module definition
├── go.sum                    # Go module checksums
├── LICENSE                   # License
└── logo.xcf                  # GIMP source file for logo.png
```

### Directory Descriptions

- **`cmd/`**: Contains the main application entry points. The `coderoom-ai` subdirectory holds the primary executable. It parses the command line flags, serves the embedded static assets, the WebSocket endpoint (`/chat`), the changed file endpoint (`/edits/`) and the settings endpoints (`/settings/`).
- **`internal/`**: Private application code not intended for external use.
  - **`backend/`**: Backend service implementations including OpenAI integration, test backend, WebSocket handling, settings persistence and the store that serves changed file contents.
  - **`blockingfiles/`**: Pattern matching used to decide whether a tool may run after the agent has modified files.
  - **`browser/`**: Browser interaction layer for DOM manipulation and WebSocket communication. Compiled to WebAssembly.
  - **`common/`**: Static catalog of the supported providers and their models.
  - **`tools/`**: The tools the agent can call, plus the bookkeeping of file edits used for the diff view and for blocking file checks.
  - **`ui/`**: User interface logic and message handling. Compiled to WebAssembly.
  - **`wire/`**: Message definitions and communication protocols between frontend and backend.
- **`web/`**: Web frontend components.
  - **`assets/`**: Static files served to the browser (HTML, CSS, images).
  - **`dist/`**: Compiled output including WebAssembly binaries and JavaScript support files.
  - **`wasm/`**: Source code compiled to WebAssembly.
  - **`web.go`**: Embeds `assets/` and `dist/` into the native binary.
- **`bin/`**: Compiled binary outputs.

## Available Tools

The agent can call the following tools:

| Tool | Description |
|------|-------------|
| `list_dir` | Lists files and directories in a given relative path within the project root. |
| `read_file` | Reads the content of a specified file relative to the project root. |
| `grep` | Searches files in the project directory for occurrences of a regexp pattern. Returns matching lines with file paths and line numbers. Hidden files and directories, binary files and common non-source directories (`node_modules`, `vendor`, `.git`, `CVS`) are skipped. |
| `write_file` | Creates a new file or overwrites an existing file with specified content. |
| `edit_file` | Replaces the first occurrence of a block of content with a new block. |
| `delete_file` | Deletes the specified file relative to the project root. |
| `propose_plan` | Proposes a plan to the user and returns whether the user wishes to proceed. |
| `present_options` | Presents a list of options to the user and returns the selected option. |
| `report_progress` | Reports the current work in progress with an optional completion percentage. |
| `build_project` | Runs the project's configured build command. Only available when configured (see [Project Configuration](#project-configuration-coderoom-aijson)). |
| `run_tests` | Runs the project's configured test command. Only available when configured. |

The *mutating* tools (`write_file`, `edit_file`, `delete_file`, `grep`) and the two command tools are only offered to the model when the UI is in modification mode; `list_dir`, `read_file`, `propose_plan`, `present_options` and `report_progress` are always available. In inspection mode the model can therefore still explore the project.

Tool calls that change files are recorded in a per-session change list, which is used both for the changed file diff view and for the blocking file checks described below.

## Supported Providers

Providers and models are selected per conversation in the UI and configured in the Global Settings dialog. Each provider entry holds its own API key.

| Provider | Provider ID | Models |
|----------|-------------|--------|
| DeepSeek | `deepseek` | DeepSeek Flash, DeepSeek V4 Pro |
| Depaza | `depaza` | Max, Advanced |
| GDPRchat | `gdprchat` | Code |
| Kimi (Moonshot AI) | `kimi` | Kimi K3 |
| OpenAI | `openai` | GPT-6 Luna |

The catalog lives in `internal/common/providers.go`; every provider is used through an OpenAI compatible API, so adding a provider only requires adding an entry there.

## How to Build

### Prerequisites
- Go 1.27.0 or later (see `go.mod`)
- Make (optional, for using Makefile on Linux/macOS; not needed on Windows, where `build.bat` and `runtests.bat` are provided)

### Build Steps

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd coderoom-ai
   ```

2. Build everything at once:
   ```bash
   make
   ```

   The default target builds, in order:

   * `web/dist/main.wasm` — the WebAssembly frontend (`GOOS=js GOARCH=wasm go build -o web/dist/main.wasm web/wasm/main.go`)
   * `web/dist/wasm_exec.js` — copied from the Go installation by `go run copy_wasm_exec.go`
   * `bin/coderoom-ai` — the native binary (`go build -o bin/coderoom-ai cmd/coderoom-ai/main.go`)

   The WASM build must run before the native build, because `web/web.go` embeds the files in `web/dist/` into the binary.

   **On Windows**, use the provided batch script instead of `make`:
   ```bat
   build.bat
   ```
   This compiles `web/dist/main.wasm`, copies `wasm_exec.js` into `web/dist/`, and builds `bin\coderoom-ai.exe`.

### Dependencies

The external dependencies are intentionally few:

- `github.com/sashabaranov/go-openai` — OpenAI compatible client (all providers)
- `github.com/gorilla/websocket` — frontend ↔ backend WebSocket connection
- `github.com/gomarkdown/markdown` — Markdown rendering in the WASM frontend
- `znkr.io/diff` — unified diffs for the changed file review dialog

## How to Use

1. **Start the application**
   ```bash
   cd /home/user/my/project
   coderoom-ai
   ```

2. **Open your web browser** and navigate to the application URL (typically `http://localhost:8037/`, or as specified in the application output)

3. **Answer the directory question.** On the first run in a project directory, Coderoom AI asks whether it may work in that directory. Allow answers are remembered in `~/.config/coderoom-ai/global.json` and can be removed again from Global Settings.

4. **Begin coding** with AI assistance through the web interface.

### The web interface

- **Modification mode toggle** — switch between *Modifications allowed* and *Inspection only*. In inspection mode the mutating tools are not offered to the model.
- **Provider / model selector** — choose which configured provider and model answers the current conversation.
- **Reset session** — reloads the page and starts a fresh conversation with a fresh change tracking session.
- **Theme toggle** — switches between the dark and light theme; the choice is persisted.
- **Settings menu** — opens the Global Settings and Project Settings dialogs.
- **Changed files list** — after each completed turn the files touched in the session are listed together with the action taken on them (entire file, modified, deleted). Clicking a modified file opens a dialog with a colorized unified diff; new files are shown in full.
- **Plans, options and blocked actions** — `propose_plan`, `present_options` and blocked command tools all render as messages with explicit buttons, so the agent can ask a question without a free form confirmation prompt.

### Test mode

With `-testmode` the backend makes no API calls. Instead, the prompt text is matched against keywords that each render one kind of UI message:

| Keyword | Message rendered |
|---------|------------------|
| `system` | Markdown system message |
| `reasoning` | Reasoning message (visible at high verbosity) |
| `tool` | Tool message |
| `proposal` | Plan proposal with Proceed / Decline buttons |
| `options` | Option list with a Proceed button |
| `blocked` | Blocked action with Allow / Deny buttons |

### Logging

After each completed turn the backend dumps the full conversation to `/tmp/messages-<unix-timestamp>.json` and logs the file name, which is helpful when debugging tool calls.

## Configuration

Coderoom AI supports several configuration options via command-line flags and settings files:

### Command-Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-dir` | Project directory to work on | Current working directory |
| `-port` | TCP port for the web server | `8037` |
| `-testmode` | Run backend in test mode (no API calls) | `false` |

### Example Usage

```bash
# Run with custom project directory and port
coderoom-ai -dir /path/to/project -port 9000

# Run in test mode (useful for development)
coderoom-ai -testmode
```

### Settings Directory

Settings are stored in `~/.config/coderoom-ai/`. The application will create this directory automatically on first run. The directory is printed on startup.

#### Global Settings (`global.json`)

Global settings are edited in the *Global Settings* dialog and stored in `~/.config/coderoom-ai/global.json` (written with `0600` permissions, since it contains API keys):

```json
{
  "version": 0,
  "auto_open": "firefox",
  "default_modifications": true,
  "dark_theme": true,
  "verbosity": "medium",
  "allowed_dirs": ["/home/user/my/project"],
  "providers": [
    {
      "provider_id": "deepseek",
      "model_id": "deepseek-v4-pro",
      "api_key": "sk-...",
      "default": true
    }
  ]
}
```

| Field | Description |
|-------|-------------|
| `auto_open` | Command used to open the browser on startup (`chrome`, `firefox`, `xdg-open`). Omitted or empty means the browser is not opened automatically. |
| `default_modifications` | Whether new sessions start in modification mode. |
| `dark_theme` | Whether the dark theme is active. |
| `verbosity` | One of `minimal`, `medium` or `high`. Controls how much detail is shown in the conversation; tool messages are hidden at `minimal` and reasoning messages are only shown at `high`. Unknown values fall back to `medium`. |
| `allowed_dirs` | Project directories the user has allowed. Removing an entry makes the next session ask again. |
| `providers` | Configured provider/model entries. `default` marks the provider used when no specific one is selected; API keys are never sent back to the browser. |

### Project Configuration (`coderoom-ai.json`)

You can configure tool behavior in your project's `coderoom-ai.json` file. It is edited in the *Project Settings* dialog, and a tool is only offered to the model when its command is set:

```json
{
  "version": 0,
  "name": "Coderoom AI",
  "build_project_tool": {
    "command": "make",
    "blocking_files": [
      "/Makefile",
      "/copy_wasm_exec.go"
    ]
  },
  "run_tests_tool": {
    "command": "make test",
    "blocking_files": [
      "*_test.go"
    ]
  }
}
```

| Field | Description |
|-------|-------------|
| `name` | Project name shown in the top bar. |
| `build_project_tool` | The `build_project` tool. `command` is executed with `bash -c` from the project root. |
| `run_tests_tool` | The `run_tests` tool, same structure as `build_project_tool`. |

- **`blocking_files`**: Tools will only run if the specified files have not been modified by the agent, preventing unsafe operations on changed files. The same list is offered as three policies in Project Settings:
  - `["*"]` — block on any file modification (the default for a new tool).
  - `[]` — don't block on file modifications.
  - an explicit list of patterns — block only on matching, modified files.
- **Patterns**:
  - A pattern with no `/` characters (e.g. `*_test.go`) matches files by name in any directory under the project root.
  - A pattern with one or more `/` characters (e.g. `internal/ui/*.go`) is matched as a path relative to the project root. A leading `/` is treated as being anchored at the project root (e.g. `/Makefile`).
  - `*` matches any sequence of non-`/` characters and `?` matches a single non-`/` character.
  - In Project Settings each pattern can be tested against the current project with the **Test** button, which lists the files that match.

### HTTP Endpoints

| Path | Description |
|------|-------------|
| `/` | The web UI (`index.html`), `styles.css`, `logo.png`, `main.wasm` and `wasm_exec.js` |
| `/chat` | WebSocket endpoint carrying the conversation |
| `/edits/{session}/{file}` | Content of a file changed during the session (unified diff or entire file) |
| `/settings/global` | Read / write the global settings (`GET`, `POST`) |
| `/settings/global/theme` | Persist the theme (`POST` form field `dark_theme`) |
| `/settings/project` | Read / write the project settings (`GET`, `POST`) |
| `/settings/project/blocking-files` | Test a blocking file pattern against the project |

## Security

Coderoom AI is designed with security as a primary concern:

### File System Restrictions
- All file operations are restricted to the project directory using Go's `os.Root` type
- The agent cannot access files outside the specified project directory

### Directory Consent
- Nothing is read or written until the user explicitly allows the project directory
- Allowed directories are listed in Global Settings and can be revoked there

### Source Control Protection
- Direct modification of source control metadata (`.git`, `.svn`, `.hg`, `.bzr`, `_darcs`, `cvs`) is prevented
- The check is case-insensitive, ignores the trailing dots and spaces that Windows strips, and treats both `/` and `\` as separators, so a path cannot slip past it
- This protects your version history from accidental AI modifications

### Confirmation Fatigue Prevention
- The AI avoids asking excessive confirmation questions for complex commands
- Instead, it uses safe defaults and blocking file mechanisms

### Tool Execution Safeguards
- Tools like `build_project` and `run_tests` only execute if their blocking files haven't been modified by the agent
- When a blocked tool is attempted, the changed files are shown to the user and the agent waits for an explicit allow/deny decision
- The agent never runs these commands if the tool is not configured in `coderoom-ai.json`

### Rollback Mechanisms
- Every file change in a session is tracked together with its original content
- Each turn lists the changed files with their diff, so all changes can be reviewed (and reverted) before anything is committed
- It is still strongly recommended to use a version control system for checkpointing

## Testing

### Running Tests

Run the test suite using Make:

```bash
make test
```

This runs the `internal/backend`, `internal/blockingfiles` and `internal/tools` test packages. Individual packages can also be tested directly:

```bash
go test -v internal/backend/*.go
go test -v internal/blockingfiles/*.go
go test -v internal/tools/*.go
```

On Windows, use the provided batch script instead:

```bat
runtests.bat
```

It runs the `internal/backend`, `internal/blockingfiles`, and `internal/tools` test packages.

Note that `internal/browser`, `internal/ui` and `web/wasm` build only for `GOOS=js GOARCH=wasm`, so they cannot be tested on the host platform.

### Test Coverage

The project includes unit tests for:

- All file tools: read, write, edit, delete, directory listing and grep
- File edit tracking and the generated diffs (`fileedits_test.go`)
- The `build_project` and `run_tests` command tools, including blocking file behaviour
- Source control path protection
- Plan proposal, option presentation and progress reporting
- Blocking file pattern matching (`internal/blockingfiles`), and testing a pattern against a project directory (`internal/backend`)

### Adding Tests

When contributing new features, please include corresponding tests. Follow the existing test patterns in the `internal/tools/` directory.

## Style Guidelines

In general, this project follows standard Go conventions and best practices. Try to follow the existing style as much as possible.

## Contributing

We welcome contributions! Please follow these guidelines:

### Getting Started
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Test thoroughly
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Review Process
- All submissions require review
- Be respectful and constructive in feedback
- Address review comments promptly
- Ensure CI passes before merging

### Commit Messages
- Use clear and descriptive commit messages
- Start with a verb in present tense (e.g., "Add", "Fix", "Update")
- Reference issues when applicable
- Keep the first line under 72 characters

### Questions?
Feel free to open an issue for any questions or discussions about the project.

## License

This project is licensed under the terms specified in the [LICENSE](LICENSE) file.
