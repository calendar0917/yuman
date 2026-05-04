# PRD: Multi-source TUI Package Manager (yuman)

## Background

Managing packages across multiple package managers (pacman, pip, npm, cargo, etc.) is painful — each has different syntax, no shared view, no unified search. Existing tools like mpm cover 50+ managers but are CLI-only with no visual interface and poor parallelism. We want a **TUI-based** unified package manager focused on **Linux (Arch)**.

## Project Name

**yuman** (the repo name)

## Problem Statement

A developer on Arch Linux might have 5-6 package managers active simultaneously. There's no way to:
- See all installed packages in one view
- Search across all managers at once
- Compare versions or find duplicates
- Perform install/remove/upgrade with a consistent interface

## Research Summary

### Existing Products (Competitors)

| Tool | Type | Coverage | Limitation |
|------|------|---------|------------|
| **mpm** | Python CLI | 50 managers, full CRUD | No TUI/GUI, serial execution, no plugin system, GPL-2.0 |
| **GlazePKG** | Go TUI | 36 managers, read-only + snapshot | Read-only focused, no real install/remove |
| **upt** | Rust CLI | 28 system PMs | Syntax translation only, no language-ecosystem PMs |
| **topgrade** | Rust CLI | 50+ updaters | Upgrade-only, no install/search/remove |
| **metapac** | Rust CLI | 19 backends | Declarative config, no TUI, basic |
| **UniGetUI** | C# GUI | 7 managers | Windows-only |
| **Helm** | Rust+Swift GUI | 15+ managers | macOS-only |

### Market Gap

- No cross-platform TUI that does full CRUD across system + language-ecosystem package managers
- Linux has zero visual unified package managers (UniGetUI=Windows, Helm=macOS)
- mpm is the closest in functionality but purely CLI — our TUI would be a differentiator

## Scope

### Target Platform

- **Primary**: Linux (Arch-based)
- **Stretch**: other Linux distros (Ubuntu/Debian, Fedora)
- **Out of scope**: macOS, Windows

### V1 Managers (6-7 managers)

| Manager | Type | Priority |
|---------|------|----------|
| pacman | System (Arch official) | P0 |
| paru / yay | AUR helper | P0 |
| pip | Python | P0 |
| npm | Node.js | P0 |
| pnpm | Node.js (alt) | P1 |
| cargo | Rust | P1 |
| flatpak | Sandboxed apps | P1 |

### V1 Features

**Core operations** (per manager):
- `list` — show installed packages
- `search <query>` — search packages
- `install <pkg>` — install package
- `remove <pkg>` — remove package
- `outdated` — show upgradable packages
- `upgrade [pkg]` — upgrade specific or all

**TUI interface**:
- Unified dashboard showing all managers and their status
- Browsable installed package list (filterable by manager)
- Search view with cross-manager results
- Package detail pane (version, description, dependencies)
- Action confirmation dialogs for install/remove/upgrade
- Status bar showing operation progress

**Cross-cutting**:
- Parallel queries across managers (async subprocess execution)
- Configurable (TOML config: which managers to enable, default AUR helper, etc.)

### V2 Features (future)

- Backup/restore (export installed list to TOML/JSON)
- Duplicate detection across managers
- Environment management (mise/asdf-like .tool-versions)
- Plugin system for adding custom managers
- Notification daemon for outdated packages

## Technical Design

### Language & Stack

- **Language**: Go
- **TUI framework**: [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss) + [bubbles](https://github.com/charmbracelet/bubbles)
- **Config**: TOML (via `toml` or `viper`)
- **Subprocess**: `os/exec` with `context` for cancellation and timeouts

### Architecture

```
yuman/
├── cmd/
│   └── yuman/
│       └── main.go           # entry point
├── internal/
│   ├── manager/              # package manager adapters
│   │   ├── manager.go        # Manager interface
│   │   ├── pacman.go
│   │   ├── paru.go
│   │   ├── pip.go
│   │   ├── npm.go
│   │   ├── pnpm.go
│   │   ├── cargo.go
│   │   └── flatpak.go
│   ├── tui/                  # TUI views
│   │   ├── app.go            # root model
│   │   ├── dashboard.go      # overview
│   │   ├── installed.go      # installed packages list
│   │   ├── search.go         # search view
│   │   ├── detail.go         # package detail
│   │   └── styles.go         # lipgloss styles
│   ├── config/               # TOML config loading
│   │   └── config.go
│   └── model/                # data types
│       └── package.go
├── go.mod
├── go.sum
└── yuman.toml.example
```

### Manager Interface

```go
type Manager interface {
    Name() string
    Available() bool                         // is this manager installed?
    List(ctx context.Context) ([]Package, error)
    Search(ctx context.Context, query string) ([]Package, error)
    Install(ctx context.Context, pkg string) error
    Remove(ctx context.Context, pkg string) error
    Outdated(ctx context.Context) ([]Package, error)
    Upgrade(ctx context.Context, pkg string) error
}
```

Each adapter implements this interface by calling the underlying CLI (e.g., `pacman -Qe`) and parsing stdout.

### Parallel Execution

All cross-manager queries (`installed`, `outdated`, `search`) run concurrently via goroutines with a shared `errgroup`. Individual manager operations have timeouts (default 30s).

## Completion Criteria

- [ ] TUI launches and shows dashboard with all detected managers
- [ ] Can browse installed packages across all 7 managers
- [ ] Can search packages across managers with results in unified view
- [ ] Can install/remove/upgrade with confirmation dialog
- [ ] Parallel queries complete in <5s for all managers
- [ ] TOML config file for enabling/disabling managers
- [ ] `go install` or `go build` produces single binary

## Milestones

| Milestone | Scope |
|-----------|-------|
| **M1: Skeleton** | Project setup, Manager interface, pacman adapter, basic TUI shell |
| **M2: Adapters** | All 7 manager adapters with list/search/outdated |
| **M3: TUI** | Full TUI with dashboard, list, search, detail views |
| **M4: Actions** | Install/remove/upgrade with confirmation dialogs |
| **M5: Polish** | Config file, error handling, parallel queries, docs |
