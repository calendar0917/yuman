# Quality Guidelines

> Code quality standards for yuman (Go TUI package manager).

---

## Overview

This is a Go TUI application using bubbletea v2 + lipgloss v2 + bubbles v2.
Architecture: Manager adapters (CLI wrappers) → TUI views (bubbletea Model).

---

## Forbidden Patterns

### Don't: Bare goroutines writing to TUI state

```go
// WRONG — data race, TUI never refreshes
go func() {
    pkgs, _ := mgr.List(ctx)
    a.installedPkgs = pkgs  // race!
}()
```

**Why**: Bubbletea owns the model. Direct mutation from goroutines causes data races and the View is never re-rendered.

**Instead**: Always return a `tea.Cmd` that sends a message:

```go
// CORRECT — message-driven, race-free
func (a *App) loadPkgs() tea.Cmd {
    mgr := a.selectedMgr
    return func() tea.Msg {
        pkgs, _ := mgr.List(ctx)
        return loadedMsg{packages: pkgs}
    }
}
```

### Don't: Use `context.Background()` without timeout in subprocess calls

```go
// WRONG — can hang forever
exec.CommandContext(context.Background(), "pacman", "-Q")
```

**Instead**: Always wrap with timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(),
    time.Duration(cfg.TimeoutSeconds())*time.Second)
defer cancel()
exec.CommandContext(ctx, "pacman", "-Q")
```

---

## Required Patterns

### Manager adapter pattern

Every package manager adapter must:
1. Implement the full `Manager` interface (all 7 methods)
2. Set the `Manager` field on every returned `model.Package`
3. Return `errNotImplemented` for destructive operations (install/remove/upgrade) until real implementations are added
4. Handle CLI non-zero exit codes gracefully (e.g., `checkupdates` exit 2 = no updates)

### Parser functions must set Manager field

All shared parsers (`parseSpaceKV`, `parsePacmanSearch`, etc.) must accept a `mgr string` parameter and set it on every returned package.

---

## Bubbletea v2 Conventions

### Import paths

```go
import (
    tea "charm.land/bubbletea/v2"
    "charm.land/bubbles/v2/table"
    "charm.land/bubbles/v2/textinput"
    "charm.land/bubbles/v2/spinner"
    "charm.land/lipgloss/v2"
)
```

### View() returns tea.View

```go
func (m model) View() tea.View {
    return tea.NewView("rendered string")
}
```

### Key handling

```go
case tea.KeyPressMsg:
    switch msg.String() {
    case "q", "ctrl+c":
        return m, tea.Quit
    case "enter":
        // ...
    }
```

**Warning**: If a `textinput` is focused, letter keys like `q` will be consumed by the input before reaching your handler. Guard letter-key shortcuts with `if a.state != viewSearch`.

### Table component (bubbles v2)

```go
t := table.New(
    table.WithColumns([]table.Column{{Title: "Name", Width: 30}}),
    table.WithRows([]table.Row{{"value1", "value2"}}),
    table.WithFocused(true),
    table.WithHeight(10),
)
s := table.DefaultStyles()
s.Header = s.Header.Bold(true)
t.SetStyles(s)

// In Update: delegate
m.table, cmd = m.table.Update(msg)

// In View: render
return t.View()
```

---

## Testing Requirements

- `go vet ./...` must pass
- `go build ./cmd/yuman/` must produce a single binary
- No test files yet — add unit tests for parsers (`parseSpaceKV`, `parsePacmanSearch`, etc.) as next step

---

## Code Review Checklist

- [ ] All `model.Package` results have `Manager` field set
- [ ] No bare goroutines — async work goes through `tea.Cmd`
- [ ] CLI commands use `context.WithTimeout`
- [ ] `go vet` and `go build` pass
- [ ] Key shortcuts don't conflict with text input focus
