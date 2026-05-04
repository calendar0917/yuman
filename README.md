# yuman

A terminal UI package manager for Linux that unifies **7 package managers** into a single interface.

```
┌──────────────────────────────────────────────────────┐
│  yuman                                        [q]uit │
├──────────────────────────────────────────────────────┤
│                                                      │
│  Package Managers                                    │
│  ─────────────────                                   │
│  ▸ pacman        1247 installed    3 updates         │
│    paru            89 installed    0 updates         │
│    pip              42 installed    5 updates         │
│    npm              23 installed    2 updates         │
│    pnpm             15 installed    0 updates         │
│    cargo             8 installed    1 updates         │
│    flatpak           6 installed    0 updates         │
│                                                      │
│  j/k: navigate  enter: view  /: search  r: reload   │
└──────────────────────────────────────────────────────┘
```

## Supported Package Managers

| Manager | List | Search | Install | Remove | Outdated |
|---------|------|--------|---------|--------|----------|
| pacman | `pacman -Qs` | `pacman -Ss` | `pacman -S` | `pacman -R` | `checkupdates` |
| paru / yay | `paru -Qs` | `paru -Ss` | `paru -S` | `paru -R` | `paru -Qu` |
| pip | `pip list --format=json` | `pip index versions` | `pip install` | `pip uninstall` | `pip list --outdated` |
| npm | `npm ls -g --json` | `npm search --json` | `npm install -g` | `npm uninstall -g` | `npm outdated -g` |
| pnpm | `pnpm ls -g --json` | `npm search --json` | `pnpm install -g` | `pnpm uninstall -g` | `pnpm outdated -g` |
| cargo | `cargo install --list` | `cargo search` | `cargo install` | `cargo install` (no uninstall) | registry check |
| flatpak | `flatpak list` | `flatpak search` | `flatpak install` | `flatpak uninstall` | `flatpak remote-ls --updates` |

## Install

```bash
go install github.com/calendar/yuman/cmd/yuman@latest
```

Or build from source:

```bash
git clone https://github.com/calendar0917/yuman.git
cd yuman
go build -o yuman ./cmd/yuman
```

## Usage

```bash
yuman
```

### Keybindings

**Global**

| Key | Action |
|-----|--------|
| `/` | Open search |
| `Esc` | Back to dashboard |
| `q` | Quit (dashboard / installed views) |
| `Ctrl+C` | Force quit |

**Dashboard**

| Key | Action |
|-----|--------|
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `Enter` | View installed packages for selected manager |
| `r` | Reload all managers |

**Installed Packages**

| Key | Action |
|-----|--------|
| `j` / `Down` | Move cursor down |
| `k` / `Up` | Move cursor up |
| `Enter` / `d` | View package details |
| `u` | Update selected package |

**Search**

| Key | Action |
|-----|--------|
| `Tab` | Toggle focus between input and results |
| `Enter` (in input) | Execute search |
| `Enter` / `d` (in results) | View package details |
| `j` / `Down` (in results) | Move cursor down |
| `k` / `Up` (in results) | Move cursor up |
| `f` (in results) | Toggle installed-only filter |

**Detail**

| Key | Action |
|-----|--------|
| `i` | Install package |
| `u` | Update package |
| `Esc` | Return to previous view |

## Configuration

Config file location: `~/.config/yuman/yuman.toml`

```toml
[general]
aur_helper = "paru"   # "paru" or "yay"
timeout = 30          # seconds per operation

[managers.pacman]
enabled = true

[managers.paru]
enabled = true

[managers.pip]
enabled = true

[managers.npm]
enabled = true

[managers.pnpm]
enabled = true

[managers.cargo]
enabled = true

[managers.flatpak]
enabled = true
```

Copy the example config:

```bash
mkdir -p ~/.config/yuman
cp yuman.toml.example ~/.config/yuman/yuman.toml
```

## Project Structure

```
cmd/yuman/main.go              Entry point
internal/
├── model/package.go           Package data structure
├── config/config.go           TOML config loader with defaults
├── manager/
│   ├── manager.go             Manager interface
│   ├── pacman.go              pacman adapter + shared parsers
│   ├── paru.go                paru/yay adapter
│   ├── pip.go                 pip adapter
│   ├── npm.go                 npm adapter + JSON parser
│   ├── pnpm.go                pnpm adapter
│   ├── cargo.go               cargo adapter
│   ├── flatpak.go             flatpak adapter
│   └── manager_test.go        19 unit tests
└── tui/
    ├── app.go                 TUI views and key handlers (~900 lines)
    └── styles.go              lipgloss style definitions
```

## Tech Stack

- [bubbletea v2](https://github.com/charmbracelet/bubbletea) — TUI framework (Elm architecture)
- [bubbles v2](https://github.com/charmbracelet/bubbles) — table, text input, viewport components
- [lipgloss v2](https://github.com/charmbracelet/lipgloss) — terminal styling
- [go-toml v2](https://github.com/pelletier/go-toml) — config parsing

## Development

```bash
# Run tests
go test ./...

# Build
go build -o yuman ./cmd/yuman

# Vet
go vet ./...
```

## License

MIT
