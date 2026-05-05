# yuman

Multi-source TUI/CLI package manager for Linux — unifies **11 package managers** into a single interface.

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
│  j/k: navigate  enter: view  /: search  D: duplicates│
│  v: env  r: reload  s: snapshot  q: quit             │
└──────────────────────────────────────────────────────┘
```

## Supported Package Managers

| Manager | List | Search | Outdated | Needs sudo |
|---------|------|--------|----------|------------|
| pacman | `pacman -Q` | `pacman -Ss` | `checkupdates` | yes |
| paru / yay | `paru -Q` | `paru -Ss` | `paru -Qu` | no |
| apt | `dpkg-query -W` | `apt-cache search` | `apt list --upgradable` | yes |
| dnf | `rpm -qa` | `dnf search` | `dnf check-update` | yes |
| zypper | `zypper se -i` | `zypper se` | `zypper list-updates` | yes |
| brew | `brew list --formula` | `brew search` | `brew outdated --json` | no |
| pip | pip JSON API | `pip index versions` | `pip list --outdated` | no |
| npm | `npm ls -g --json` | npm registry API | `npm outdated -g` | no |
| pnpm | `pnpm ls -g --json` | npm registry API | `pnpm outdated -g` | no |
| cargo | `cargo install --list` | `cargo search` | `cargo-outdated` | no |
| flatpak | `flatpak list` | `flatpak search` | `flatpak remote-ls --updates` | no |

## Install

### Binary releases

Download the latest binary from [GitHub Releases](https://github.com/calendar0917/yuman/releases):

```bash
# Linux amd64
curl -L https://github.com/calendar0917/yuman/releases/latest/download/yuman_linux_amd64.tar.gz | tar xz
sudo mv yuman /usr/local/bin/
```

### AUR (Arch Linux)

```bash
paru -S yuman-bin
```

### Homebrew

```bash
brew tap calendar0917/tap
brew install yuman
```

### Go install

```bash
go install github.com/calendar/yuman/cmd/yuman@latest
```

### Build from source

```bash
git clone https://github.com/calendar0917/yuman.git
cd yuman
go build -o yuman ./cmd/yuman
```

## Usage

### TUI mode

```bash
yuman
```

**Keybindings:**

| Key | Context | Action |
|-----|---------|--------|
| `j` / `k` | all lists | Move cursor down / up |
| `/` | global | Open search |
| `Enter` | dashboard | View installed packages |
| `Enter` / `d` | installed / search | View package details |
| `Tab` | search | Toggle input ↔ results |
| `f` | search results | Toggle installed-only filter |
| `D` | dashboard | Duplicate detection |
| `v` | dashboard | Environment view |
| `s` | dashboard | Snapshot export/import |
| `r` | dashboard | Reload all managers |
| `i` | detail | Install package |
| `u` | detail / installed | Upgrade package |
| `Esc` | any | Back to dashboard |
| `q` | dashboard / installed | Quit |
| `Ctrl+C` | global | Force quit |

### CLI mode

```bash
# List installed packages
yuman list                     # all managers
yuman list -m pacman           # single manager
yuman list --output json       # JSON output

# Search for packages
yuman search ripgrep
yuman search -m cargo ripgrep

# Check for outdated packages
yuman outdated
yuman outdated -m npm --output json

# Install / remove / upgrade
yuman install ripgrep
yuman remove ripgrep
yuman upgrade                   # upgrade all
yuman upgrade ripgrep           # upgrade specific package

# Package info
yuman info ripgrep

# Duplicate detection
yuman duplicates

# Environment management
yuman env init                  # create yuman.tools.toml
yuman env install               # install all tools from env
yuman env status                # show installed/missing tools

# Snapshots
yuman snapshot export [path]
yuman snapshot import <path>

# Config & version
yuman config                    # show config file path
yuman version                   # print version

# Shell completion
yuman completion bash
yuman completion zsh
yuman completion fish
```

**Global flags:**
- `-o, --output` — `table` (default), `json`, `plain`
- `-m, --manager` — filter by manager name
- `-h, --help` — help for any command
- `--version` — print version

## Configuration

Config file: `~/.config/yuman/yuman.toml`

```toml
[general]
aur_helper = "paru"       # "paru" or "yay"
timeout = 30              # seconds per operation
dedup_ignore = []         # package names to exclude from dedup

[managers.pacman]
enabled = true

[managers.paru]
enabled = true

[managers.apt]
enabled = false

[managers.dnf]
enabled = false

[managers.zypper]
enabled = false

[managers.brew]
enabled = false

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

System package managers (apt, dnf, zypper) default to disabled. Enable them on the relevant distribution.

## Environment Management

Pin project-level tool versions with `yuman.tools.toml`:

```toml
[tools]
[tools.ripgrep]
name = "ripgrep"
manager = "pacman"
version = "14"

[tools.node]
name = "node"
manager = "pnpm"
version = "22"
```

Place this file in your project root. `yuman env install` installs all specified tools at the pinned versions.

## Architecture

```
cmd/yuman/main.go              Entry point + version injection
internal/
├── model/                     Shared types + Manager interface
│   ├── package.go             Package struct
│   ├── manager.go             Manager interface
│   └── v2.go                  V2 types (DuplicateGroup, Environment)
├── config/config.go           TOML config loader
├── manager/                   Adapters (11 managers)
│   ├── manager.go             All() registry
│   ├── parsers.go             Shared parsers
│   ├── pacman.go / paru.go
│   ├── apt.go / dnf.go / zypper.go / brew.go
│   ├── pip.go / npm.go / pnpm.go / cargo.go / flatpak.go
│   └── manager_test.go
├── backend/
│   ├── backend.go             Shared orchestration (parallel queries, cache)
│   ├── dedup.go               Cross-manager duplicate detection
│   ├── environment.go         yuman.tools.toml management
│   └── stream.go              Live command output streaming
├── cli/                       Cobra CLI frontend
│   ├── root.go                Root command + flags
│   ├── commands.go            list/search/outdated
│   ├── actions.go             install/remove/upgrade/info/snapshot/version
│   ├── v2.go                  duplicates/env
│   ├── output.go              Table/JSON/plain formatter
│   └── resolve.go             Manager disambiguation
├── tui/                       Bubbletea TUI frontend
│   ├── app.go                 Main app (~1100 lines)
│   ├── views.go               View rendering
│   ├── messages.go            Msg types + view state
│   └── styles.go              lipgloss styles
└── backup/                    Snapshot export/import
```

## Tech Stack

- [bubbletea v2](https://github.com/charmbracelet/bubbletea) — TUI framework
- [bubbles v2](https://github.com/charmbracelet/bubbles) — table, textinput, viewport, spinner
- [lipgloss v2](https://github.com/charmbracelet/lipgloss) — terminal styling
- [cobra](https://github.com/spf13/cobra) — CLI framework
- [go-toml v2](https://github.com/pelletier/go-toml) — config parsing

## Development

```bash
go build ./...    # build
go vet ./...      # lint
go test ./...     # test
```

CI runs on every push to master. Tagged releases (`v*`) trigger goreleaser to build binaries, DEB/RPM/Arch packages, AUR publication, and Homebrew tap update.

## License

MIT