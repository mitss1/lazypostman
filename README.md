# LazyPostman

A lazygit-style terminal UI for working with Postman collections. Built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue)

## Features

- **Collection tree navigation** — Browse folders and requests with vim-style keys
- **3-panel layout** — Collection tree | Request details | Response viewer
- **Postman Cloud integration** — Login with API key, browse and fetch collections/environments from Postman Cloud
- **Inline editing** — Edit URLs, params, headers, and request bodies directly in the TUI
- **Environment variables** — View, edit, toggle, and cycle environments with `{{variable}}` substitution
- **HTTP method color coding** — GET (green), POST (blue), PUT (orange), PATCH (yellow), DELETE (red)
- **Tab views** — Switch between Params, Headers, and Body for requests and responses
- **JSON pretty-print** — Automatic formatting of JSON responses
- **Auth support** — Bearer token and Basic authentication
- **Auto-discovery** — Automatically finds `.postman_environment.json` files in the same directory as the collection
- **Panel maximize** — Focus on any panel full-screen with `+`/`-`

## Project Structure

```
lazypostman/
├── cmd/lazypostman/main.go          # Entry point
├── internal/
│   ├── collection/collection.go     # Postman v2.1 parser + models
│   ├── environment/environment.go   # Environment variables + {{var}} substitution
│   ├── httpclient/client.go         # HTTP client with auth support
│   ├── postman/
│   │   ├── api.go                   # Postman Cloud API client
│   │   └── config.go               # API key config persistence
│   └── ui/
│       ├── app.go                   # Main TUI app (bubbletea)
│       ├── tree.go                  # Left panel — collection tree
│       ├── detail.go               # Right panels — request/response
│       ├── editor.go               # Inline text editor overlay
│       ├── envpanel.go             # Environment variables panel
│       ├── browser.go              # Postman Cloud browser overlay
│       └── login.go                # API key login overlay
├── testdata/
│   ├── sample.postman_collection.json
│   └── sample.postman_environment.json
├── go.mod
└── go.sum
```

## Installation

### Homebrew (macOS/Linux)

```bash
brew install mitss1/tap/lazypostman
```

### Build from source

```bash
# Requires Go 1.21+
git clone https://github.com/mitss1/lazypostman.git
cd lazypostman
go build -o lazypostman ./cmd/lazypostman/
```

### Move to PATH (optional)

```bash
sudo mv lazypostman /usr/local/bin/
```

## Usage

```bash
# Start empty — browse collections from Postman Cloud
lazypostman

# Open a local collection
lazypostman my-api.postman_collection.json

# With explicit environment file
lazypostman my-api.postman_collection.json dev.postman_environment.json

# Multiple environments
lazypostman my-api.postman_collection.json dev.json staging.json prod.json
```

Environment files matching `*.postman_environment.json` in the same directory as the collection are auto-loaded.

## Keyboard Shortcuts

### Navigation

| Key           | Action                              |
|---------------|-------------------------------------|
| `j` / `k`    | Navigate up/down / scroll           |
| `h` / `l`    | Collapse/expand folder              |
| `Space`       | Toggle folder open/close            |
| `Tab`         | Switch panel (tree -> request -> response) |
| `Shift+Tab`   | Switch panel backwards              |
| `t`           | Cycle request tabs (Params/Headers/Body) |
| `g` / `G`    | Jump to top / bottom                |
| `+` / `-`    | Maximize / restore panel            |

### Actions

| Key           | Action                              |
|---------------|-------------------------------------|
| `Enter`       | Send selected request               |
| `i`           | Edit current field (URL/param/header/body) |
| `v`           | Open environment variables panel    |
| `e`           | Cycle active environment            |

### Postman Cloud

| Key           | Action                              |
|---------------|-------------------------------------|
| `o`           | Browse collections from Postman Cloud |
| `E`           | Browse environments from Postman Cloud |
| `L`           | Login with Postman API key / Logout (press twice) |

### Editor (when open)

| Key           | Action                              |
|---------------|-------------------------------------|
| `Enter`       | Save (single-line) / New line (body) |
| `Ctrl+S`      | Save (multiline body editor)        |
| `Esc`         | Cancel editing                      |
| `Arrows`      | Move cursor                         |

### Environment Panel (when open)

| Key           | Action                              |
|---------------|-------------------------------------|
| `j` / `k`    | Navigate variables                  |
| `Space`       | Toggle variable enabled/disabled    |
| `Enter`       | Edit selected variable value        |
| `e`           | Cycle to next environment           |
| `Esc` / `v`  | Close panel                         |

### General

| Key           | Action                              |
|---------------|-------------------------------------|
| `?`           | Toggle help                         |
| `q`           | Quit                                |

## Postman Cloud Integration

LazyPostman can connect to the Postman API to fetch your collections and environments:

1. Press `L` to login with your [Postman API key](https://postman.co/settings/me/api-keys)
2. Press `o` to browse and select a collection
3. Press `E` to browse and select an environment

Your API key is stored locally at `~/.config/lazypostman/config.json`.

## Supported Formats

| Format | Status |
|--------|--------|
| Postman Collection v2.1 (`.postman_collection.json`) | Supported |
| Postman Environment (`.postman_environment.json`) | Supported |

## How It Works

1. **Collection parsing** — Reads Postman Collection v2.1 JSON and builds a navigable tree of folders and requests
2. **Variable substitution** — Replaces `{{variable}}` placeholders in URLs, headers, and body using the active environment
3. **Request execution** — Sends HTTP requests using Go's `net/http` with support for Bearer/Basic auth, form data, and JSON bodies
4. **Response display** — Shows status code, timing, size, headers, and pretty-printed body
5. **Cloud sync** — Fetches collections and environments from Postman Cloud via their REST API

## Tech Stack

- **Go** — Single binary, cross-platform
- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** — TUI framework (Elm architecture)
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** — Terminal styling
- **[Bubbles](https://github.com/charmbracelet/bubbles)** — Reusable TUI components

Same stack as [lazygit](https://github.com/jesseduffield/lazygit) and [lazydocker](https://github.com/jesseduffield/lazydocker).

## Roadmap

- [x] Load and browse Postman collections
- [x] Execute HTTP requests
- [x] Environment variable substitution
- [x] Bearer/Basic auth
- [x] JSON pretty-print responses
- [x] Vim-style navigation
- [x] Postman Cloud API sync
- [x] Edit requests inline
- [x] Environment variable panel
- [ ] Request history with persistence
- [ ] Copy response body / cURL command to clipboard
- [ ] Search/filter within collections (`/`)
- [ ] Save modified collections back to file
- [ ] OAuth2 flow support
- [ ] Import from OpenAPI/Swagger
- [ ] Custom themes and keybinding config
- [x] Logout capability
- [x] `goreleaser` for Homebrew/AUR/Scoop distribution
- [x] `--version` flag

## License

MIT
