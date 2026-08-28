# Coderoom AI
A simple AI code assistant with a web based UI.

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

## Project Structure

```
coderoom-ai/
├── cmd/
│   └── coderoom-ai/          # Main application entry point
├── internal/                  # Private application code
│   ├── backend/              # Backend implementations (OpenAI, test backend, WebSocket)
│   ├── browser/              # Browser interaction layer (DOM elements, objects, WebSocket)
│   ├── ui/                   # UI logic and message handling
│   └── wire/                 # Message definitions and communication protocols
├── web/
│   ├── assets/               # Static web assets (HTML, CSS, images)
│   ├── dist/                 # Compiled WASM and JavaScript files
│   ├── wasm/                 # WebAssembly source code
│   └── web.go                # Embedded web resources
├── bin/                      # Compiled binaries
├── go.mod                    # Go module definition
├── go.sum                    # Go module checksums
├── Makefile                  # Build automation
└── copy_wasm_exec.go         # Utility to copy WASM exec file
```

### Directory Descriptions

- **`cmd/`**: Contains the main application entry points. The `coderoom-ai` subdirectory holds the primary executable.
- **`internal/`**: Private application code not intended for external use.
  - **`backend/`**: Backend service implementations including OpenAI integration, test backend, and WebSocket handling.
  - **`browser/`**: Browser interaction layer for DOM manipulation and WebSocket communication.
  - **`ui/`**: User interface logic and message handling.
  - **`wire/`**: Message definitions and communication protocols between frontend and backend.
- **`web/`**: Web frontend components.
  - **`assets/`**: Static files served to the browser (HTML, CSS, images).
  - **`dist/`**: Compiled output including WebAssembly binaries and JavaScript support files.
  - **`wasm/`**: Source code compiled to WebAssembly.
- **`bin/`**: Compiled binary outputs.

## How to Build

### Prerequisites
- Go 1.25.12 or later
- Make (optional, for using Makefile)

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

## How to Use

1. **Start the application**
   ```bash
   cd /home/user/my/project
   coderoom-ai
   ```

2. **Open your web browser** and navigate to the application URL (typically `http://localhost:8037` or as specified in the application output)

3. **Begin coding** with AI assistance through the web interface

## Style Guidelines

In general, this project follows standard Go conventions and best practices. Try to follow the exisiting style as much as possible.

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
