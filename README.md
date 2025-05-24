# Todo TUI

A terminal-based todo application with a clean, collapsible tree UI, real-time file sync, and robust markdown file support. Bidirectionally syncs your todos between a local file of your choice and the terminal.

Built with Go and Bubbletea.

```sh
brew install camin-mccluskey/tools/td-file
```

---

## Features
- **Markdown file storage**: Todos are stored in `:td`-delimited blocks in markdown files.
- **Nested todos**: Supports unlimited hierarchy via indentation.
- **Collapsible tree UI**: Expand/collapse nested todos in the terminal.
- **Real-time sync**: Changes in the file or TUI are instantly reflected.
- **Robust error handling**: Handles malformed files, permission errors, and incomplete syntax gracefully.
- **Intuitive keybindings**: Vim-style and arrow key navigation, single-key state changes, inline editing.

---

## Project Structure

```
td-file/
├── config/         # Configuration loading and path resolution
│   ├── config.go
│   └── config_test.go
├── parser/         # File parsing, writing, and todo tree logic
│   ├── parser.go
│   └── parser_test.go
├── sync/           # File synchronization (fsnotify, save/reload)
│   ├── sync.go
│   └── sync_test.go
├── tui/            # Bubbletea TUI presentation and interaction
│   ├── tui.go
│   └── tui_test.go
├── main.go         # Entry point, wires together config, parser, sync, tui
├── go.mod
├── go.sum
└── README.md       # Project documentation
```

### Package Responsibilities

- **config**:  Loads YAML config, resolves file paths and patterns.
- **parser**:  Handles extracting, parsing, and writing todos from/to files. Contains all todo tree logic and mutation helpers. All parser-related tests are here.
- **sync**:    Watches the todo file for changes and synchronizes updates between file and TUI.
- **tui**:     Contains the Bubbletea model, view, and update logic. Exposes a simple `StartTUI` function for launching the TUI.
- **main.go**: Orchestrates config loading, file parsing, sync setup, and launches the TUI.

---

## Configuration

On first run, a configuration file will be created automatically at one of the following locations:

- `$XDG_CONFIG_HOME/td-file/config.yaml` (if `XDG_CONFIG_HOME` is set)
- `~/.config/td-file/config.yaml` (default)

You must edit this file to tell the app where your todos are stored. There are two main ways to configure this:

### 1. Use a Single Todo File
Set the `file_path` to the absolute path of your todo markdown file:

```yaml
file_path: "/absolute/path/to/todos.md"
```

### 2. Use a Daily Pattern (e.g., one file per day)
Set the `file_pattern` and `base_directory` to use a date-based filename:

```yaml
file_pattern: "todos-{YYYY-MM-DD}.md"
base_directory: "/absolute/path/to/todo-directory"
```

- The `{YYYY-MM-DD}` part will be replaced with today's date (e.g., `todos-2024-06-07.md`).
- The app will look for the file in the specified `base_directory`.

**Note:**
- The app will not create todo files for you; the specified file must exist.
- You can change the config file at any time to update where your todos are stored.
- If you want to reset the configuration, simply delete the config file and rerun the app.

---

## Usage
Run the app:

```sh
make run
```

Or build the binary:

```sh
make build
./td-file [config.yaml]
```

### Keybindings
| Key(s)         | Action                                 |
| -------------- | -------------------------------------- |
| j / k / ↑ / ↓  | Move cursor up/down                    |
| h / l          | Collapse/expand tree node              |
| x / - / > / ␣  | Complete, cancel, push, uncomplete     |
| e              | Edit todo text (inline)                |
| a              | Add sibling todo                       |
| A              | Add child todo                         |
| d              | Delete todo                            |
| q / ctrl+c     | Quit                                   |
| ? / esc        | Toggle help screen                     |

- Only todos are shown in the UI (no file content).
- All changes are synced to the file in real time.
- Inline editing protects markdown syntax characters.

---

## Developer Guide

### Prerequisites
- Go 1.20+

### Install & Run
```sh
git clone <repo-url>
cd td-file
make run
```

Or build and run:

```sh
make build
./td-file[config.yaml]
```

### Testing
```sh
make test
```
- Edge cases and tree logic are covered in `tree_test.go`.

### Linting
```sh
make lint
```
- Runs `go vet` on the codebase for basic linting.

### Contributing
- Fork and PRs welcome.
- Code is organized for clarity and extensibility.
- See `prd.md` for requirements and design.

---

## Troubleshooting
- **Missing file**: Ensure the todo file exists at the configured path.
- **Permission errors**: Run with appropriate file permissions.
- **Malformed todos**: The app will warn and skip malformed lines.

---

## License
MIT 