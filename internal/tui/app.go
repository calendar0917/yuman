package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"github.com/calendar/yuman/internal/config"
	"github.com/calendar/yuman/internal/manager"
	"github.com/calendar/yuman/internal/model"
)

// viewState tracks which view is active.
type viewState int

const (
	viewDashboard viewState = iota
	viewInstalled
	viewSearch
	viewDetail
)

// managerStatus holds runtime info about a manager.
type managerStatus struct {
	mgr       manager.Manager
	available bool
	count     int // installed count, -1 = loading, -2 = error
	err       error
}

// loadedMsg carries results from a background manager query.
type loadedMsg struct {
	managerName string
	packages    []model.Package
	err         error
}

// installedLoadedMsg carries packages for the currently viewed manager.
type installedLoadedMsg struct {
	packages []model.Package
}

// searchResultMsg carries partial search results from a single manager.
type searchResultMsg struct {
	managerName string
	packages    []model.Package
	err         error
}

// actionDoneMsg reports the result of an install/remove/upgrade action.
type actionDoneMsg struct {
	action  string
	pkgName string
	err     error
}

// App is the root TUI model.
type App struct {
	state    viewState
	managers []managerStatus
	cfg      *config.Config
	width    int
	height   int

	// Dashboard
	dashCursor int

	// Installed view
	installedTable table.Model
	installedPkgs  []model.Package
	selectedMgr    int // index into managers

	// Search
	searchInput         textinput.Model
	searchTable         table.Model
	searchPkgs          []model.Package
	searching           bool
	searchLoading       int  // pending manager search count
	searchInputFocused  bool // true = typing in input, false = navigating results
	searchFilterInstalled bool // show only installed packages

	// Detail
	detailPkg  model.Package
	prevState  viewState // where to go back on esc from detail

	// Confirmation dialog
	confirmOpen bool
	confirmPkg  model.Package
	confirmAct  string
	confirmYes  bool

	// Spinner
	spinner spinner.Model

	// Installed cache: "manager/name" → true
	installedCache map[string]bool

	// Status
	loading   int // number of pending manager loads
	statusMsg string
}

// NewApp creates and initializes the TUI application.
func NewApp(cfg *config.Config) *App {
	all := manager.All()

	statuses := make([]managerStatus, 0, len(all))
	for _, m := range all {
		if !cfg.IsEnabled(m.Name()) {
			continue
		}
		statuses = append(statuses, managerStatus{
			mgr:       m,
			available: m.Available(),
			count:     -1, // not loaded yet
		})
	}

	// Search input
	si := textinput.New()
	si.Placeholder = "search across all managers..."
	si.CharLimit = 100
	si.SetWidth(40)

	// Spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	return &App{
		state:          viewDashboard,
		managers:       statuses,
		cfg:            cfg,
		searchInput:    si,
		spinner:        sp,
		installedCache: make(map[string]bool),
	}
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Tick,
		a.loadAllManagers(),
	)
}

// loadAllManagers returns a tea.Cmd that loads installed packages from all
// available managers in parallel.
func (a *App) loadAllManagers() tea.Cmd {
	var cmds []tea.Cmd
	for _, ms := range a.managers {
		if !ms.available {
			continue
		}
		mgr := ms.mgr
		a.loading++
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(),
				time.Duration(a.cfg.TimeoutSeconds())*time.Second)
			defer cancel()
			pkgs, err := mgr.List(ctx)
			return loadedMsg{
				managerName: mgr.Name(),
				packages:    pkgs,
				err:         err,
			}
		})
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Rebuild tables with new height
		if len(a.installedPkgs) > 0 {
			a.updateInstalledTable()
		}
		if len(a.searchPkgs) > 0 {
			a.updateSearchTable()
		}
		return a, nil

	case tea.KeyPressMsg:
		return a.handleKey(msg)

	case loadedMsg:
		a.loading--
		for i, ms := range a.managers {
			if ms.mgr.Name() == msg.managerName {
				if msg.err != nil {
					a.managers[i].count = -2
					a.managers[i].err = msg.err
				} else {
					a.managers[i].count = len(msg.packages)
					a.managers[i].err = nil
					// Populate installed cache
					for _, p := range msg.packages {
						a.installedCache[msg.managerName+"/"+p.Name] = true
					}
				}
				break
			}
		}
		if a.loading == 0 {
			a.statusMsg = "all managers loaded"
		}
		return a, nil

	case searchResultMsg:
		a.searchLoading--
		if msg.err == nil {
			// Cross-reference with installed cache
			for _, p := range msg.packages {
				if a.installedCache[p.Manager+"/"+p.Name] {
					p.Installed = true
				}
				a.searchPkgs = append(a.searchPkgs, p)
			}
			a.updateSearchTable()
		}
		if a.searchLoading <= 0 {
			a.searching = false
			a.statusMsg = fmt.Sprintf("found %d results", len(a.searchPkgs))
		}
		return a, nil

	case installedLoadedMsg:
		a.installedPkgs = msg.packages
		a.updateInstalledTable()
		return a, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case actionDoneMsg:
		if msg.err != nil {
			a.statusMsg = fmt.Sprintf("%s %s failed: %v", msg.action, msg.pkgName, msg.err)
		} else {
			a.statusMsg = fmt.Sprintf("%s %s succeeded", msg.action, msg.pkgName)
		}
		return a, nil
	}

	return a, tea.Batch(cmds...)
}

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Confirmation dialog has priority
	if a.confirmOpen {
		return a.handleConfirmKey(msg)
	}

	key := msg.String()

	// Global keys
	switch key {
	case "ctrl+c":
		return a, tea.Quit
	case "q":
		if a.state != viewSearch {
			return a, tea.Quit
		}
	case "esc":
		if a.state == viewDetail {
			// Let handleDetailKey handle it — returns to prevState
			break
		}
		if a.state != viewDashboard {
			a.state = viewDashboard
			a.statusMsg = ""
			a.searching = false
			a.searchLoading = 0
			a.searchFilterInstalled = false
			a.searchInput.Blur()
			return a, nil
		}
	case "/":
		if a.state != viewSearch {
			a.state = viewSearch
			a.searchInputFocused = true
			a.searchInput.Focus()
			return a, textinput.Blink
		}
	}

	switch a.state {
	case viewDashboard:
		return a.handleDashboardKey(key)
	case viewInstalled:
		return a.handleInstalledKey(key, msg)
	case viewSearch:
		return a.handleSearchKey(key, msg)
	case viewDetail:
		return a.handleDetailKey(key)
	}

	return a, nil
}

func (a *App) handleDashboardKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if a.dashCursor > 0 {
			a.dashCursor--
		}
	case "down", "j":
		if a.dashCursor < len(a.managers)-1 {
			a.dashCursor++
		}
	case "enter":
		ms := a.managers[a.dashCursor]
		if ms.available && ms.count > 0 {
			a.selectedMgr = a.dashCursor
			a.state = viewInstalled
			return a, a.buildInstalledTable()
		}
	case "r":
		// Reload
		a.statusMsg = "reloading..."
		return a, a.loadAllManagers()
	}
	return a, nil
}

func (a *App) handleInstalledKey(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		a.installedTable, _ = a.installedTable.Update(msg)
	case "down", "j":
		a.installedTable, _ = a.installedTable.Update(msg)
	case "enter":
		row := a.installedTable.SelectedRow()
		if row != nil && len(row) > 0 {
			a.openDetail(row[0], a.managers[a.selectedMgr].mgr.Name())
		}
	case "u":
		row := a.installedTable.SelectedRow()
		if row != nil && len(row) > 0 {
			a.openConfirm("upgrade", row[0])
		}
	case "x":
		row := a.installedTable.SelectedRow()
		if row != nil && len(row) > 0 {
			a.openConfirm("remove", row[0])
		}
	}
	return a, nil
}

func (a *App) handleSearchKey(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Tab toggles focus between input and results table
	if key == "tab" {
		if len(a.searchPkgs) > 0 {
			a.searchInputFocused = !a.searchInputFocused
			if a.searchInputFocused {
				a.searchInput.Focus()
				return a, textinput.Blink
			}
			a.searchInput.Blur()
		}
		return a, nil
	}

	// Input-focused mode: typing goes to textinput
	if a.searchInputFocused {
		switch key {
		case "enter":
			query := strings.TrimSpace(a.searchInput.Value())
			if query != "" && !a.searching {
				a.searching = true
				a.statusMsg = "searching..."
				return a, a.doSearch(query)
			}
			return a, nil
		default:
			var cmd tea.Cmd
			a.searchInput, cmd = a.searchInput.Update(msg)
			return a, cmd
		}
	}

	// Table-focused mode: navigation keys go to table
	switch key {
	case "enter", "d":
		if len(a.searchPkgs) > 0 {
			row := a.searchTable.SelectedRow()
			if row != nil && len(row) > 0 {
				a.openDetail(row[0], row[2]) // row[2] is manager name
			}
		}
	case "up", "k":
		if len(a.searchPkgs) > 0 {
			a.searchTable, _ = a.searchTable.Update(msg)
		}
	case "down", "j":
		if len(a.searchPkgs) > 0 {
			a.searchTable, _ = a.searchTable.Update(msg)
		}
	case "i":
		if len(a.searchPkgs) > 0 {
			row := a.searchTable.SelectedRow()
			if row != nil && len(row) > 0 {
				a.openConfirm("install", row[0])
			}
		}
	case "f":
		a.searchFilterInstalled = !a.searchFilterInstalled
		a.updateSearchTable()
	}
	return a, nil
}

func (a *App) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "y", "Y":
		a.confirmOpen = false
		a.statusMsg = fmt.Sprintf("%s %s...", a.confirmAct, a.confirmPkg.Name)
		return a, a.executeAction(a.confirmAct, a.confirmPkg)
	case "n", "N", "esc":
		a.confirmOpen = false
		a.statusMsg = "cancelled"
	case "left", "h":
		a.confirmYes = true
	case "right", "l":
		a.confirmYes = false
	}
	return a, nil
}

func (a *App) openConfirm(action, pkgName string) {
	a.confirmOpen = true
	a.confirmAct = action
	a.confirmPkg = model.Package{Name: pkgName, Manager: a.managers[a.selectedMgr].mgr.Name()}
	a.confirmYes = true
}

func (a *App) openDetail(pkgName, mgrName string) {
	// Find the package details from our loaded data
	var pkg model.Package
	pkg.Name = pkgName
	pkg.Manager = mgrName

	// Search installed packages
	for _, p := range a.installedPkgs {
		if p.Name == pkgName {
			pkg = p
			break
		}
	}
	// Search results override (more info available)
	for _, p := range a.searchPkgs {
		if p.Name == pkgName && p.Manager == mgrName {
			pkg = p
			break
		}
	}

	a.prevState = a.state
	a.detailPkg = pkg
	a.state = viewDetail
}

func (a *App) handleDetailKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		a.state = a.prevState
		return a, nil
	case "i":
		a.openConfirm("install", a.detailPkg.Name)
	case "u":
		a.openConfirm("upgrade", a.detailPkg.Name)
	case "x":
		a.openConfirm("remove", a.detailPkg.Name)
	}
	return a, nil
}

// executeAction dispatches a manager operation as a tea.Cmd.
func (a *App) executeAction(action string, pkg model.Package) tea.Cmd {
	// Find the manager by name
	var mgr manager.Manager
	for _, ms := range a.managers {
		if ms.mgr.Name() == pkg.Manager {
			mgr = ms.mgr
			break
		}
	}
	if mgr == nil {
		return func() tea.Msg {
			return actionDoneMsg{action: action, pkgName: pkg.Name,
				err: fmt.Errorf("manager %q not found", pkg.Manager)}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(a.cfg.TimeoutSeconds())*time.Second)
		defer cancel()

		var err error
		switch action {
		case "install":
			err = mgr.Install(ctx, pkg.Name)
		case "remove":
			err = mgr.Remove(ctx, pkg.Name)
		case "upgrade":
			err = mgr.Upgrade(ctx, pkg.Name)
		default:
			err = fmt.Errorf("unknown action: %s", action)
		}
		return actionDoneMsg{action: action, pkgName: pkg.Name, err: err}
	}
}

// doSearch starts parallel searches across all managers. Each manager sends
// its own searchResultMsg when done, so the UI updates incrementally.
func (a *App) doSearch(query string) tea.Cmd {
	a.searchPkgs = nil
	a.searchLoading = 0
	a.searchFilterInstalled = false

	var cmds []tea.Cmd
	for _, ms := range a.managers {
		if !ms.available {
			continue
		}
		mgr := ms.mgr
		a.searchLoading++
		cmds = append(cmds, func() tea.Msg {
			// 5s timeout per manager for search — fast feedback
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			pkgs, err := mgr.Search(ctx, query)
			if err == nil {
				for i := range pkgs {
					pkgs[i].Manager = mgr.Name()
				}
			}
			return searchResultMsg{
				managerName: mgr.Name(),
				packages:    pkgs,
				err:         err,
			}
		})
	}

	if len(cmds) == 0 {
		a.searching = false
		return nil
	}
	return tea.Batch(cmds...)
}

// buildInstalledTable triggers an async load of installed packages for the
// selected manager. Results arrive via installedLoadedMsg.
func (a *App) buildInstalledTable() tea.Cmd {
	ms := a.managers[a.selectedMgr]
	a.installedPkgs = nil
	a.updateInstalledTable()

	if !ms.available {
		return nil
	}

	mgr := ms.mgr
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(a.cfg.TimeoutSeconds())*time.Second)
		defer cancel()
		pkgs, err := mgr.List(ctx)
		if err != nil {
			// Send empty list on error so the loading state clears
			return installedLoadedMsg{packages: nil}
		}
		return installedLoadedMsg{packages: pkgs}
	}
}

func (a *App) updateInstalledTable() {
	cols := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Version", Width: 15},
		{Title: "Description", Width: 40},
	}

	rows := make([]table.Row, 0, len(a.installedPkgs))
	for _, p := range a.installedPkgs {
		desc := p.Description
		if len(desc) > 38 {
			desc = desc[:35] + "..."
		}
		rows = append(rows, table.Row{p.Name, p.Version, desc})
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(a.tableHeight()),
		table.WithWidth(a.width),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(true).
		Foreground(colorPrimary)
	s.Selected = s.Selected.
		Foreground(colorPrimary).
		Bold(true)
	t.SetStyles(s)

	a.installedTable = t
}

func (a *App) updateSearchTable() {
	cols := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Version", Width: 12},
		{Title: "Manager", Width: 8},
		{Title: "Description", Width: 40},
	}

	rows := make([]table.Row, 0, len(a.searchPkgs))
	for _, p := range a.searchPkgs {
		if a.searchFilterInstalled && !p.Installed {
			continue
		}
		desc := p.Description
		if len(desc) > 38 {
			desc = desc[:35] + "..."
		}
		rows = append(rows, table.Row{p.Name, p.Version, p.Manager, desc})
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(a.tableHeight()),
		table.WithWidth(a.width),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(true).
		Foreground(colorAccent)
	s.Selected = s.Selected.
		Foreground(colorAccent).
		Bold(true)
	t.SetStyles(s)

	a.searchTable = t
}

func (a *App) tableHeight() int {
	h := a.height - 8
	if h < 5 {
		h = 5
	}
	return h
}

// View implements tea.Model.
func (a *App) View() tea.View {
	if a.width == 0 {
		return tea.NewView("loading...")
	}

	var b strings.Builder

	// Header
	title := TitleStyle.Render("yuman")
	b.WriteString(title)
	b.WriteString("\n")

	switch a.state {
	case viewDashboard:
		b.WriteString(a.viewDashboard())
	case viewInstalled:
		b.WriteString(a.viewInstalled())
	case viewSearch:
		b.WriteString(a.viewSearch())
	case viewDetail:
		b.WriteString(a.viewDetail())
	}

	// Confirmation dialog overlay
	if a.confirmOpen {
		b.WriteString("\n")
		b.WriteString(a.viewConfirm())
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(a.viewStatus())

	return tea.NewView(b.String())
}

func (a *App) viewDashboard() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Package Managers"))
	b.WriteString("\n\n")

	if len(a.managers) == 0 {
		b.WriteString(ErrorStyle.Render("  No managers configured. Edit ~/.config/yuman/yuman.toml"))
		b.WriteString("\n")
		return b.String()
	}

	for i, ms := range a.managers {
		cursor := "  "
		if i == a.dashCursor {
			cursor = CursorStyle.Render("▸ ")
		}

		name := ms.mgr.Name()
		avail := ""
		if !ms.available {
			avail = ErrorStyle.Render(" (not installed)")
		} else if ms.count == -1 {
			avail = a.spinner.View() + " loading..."
		} else if ms.count == -2 {
			errMsg := "error"
			if ms.err != nil {
				errMsg = ms.err.Error()
				if len(errMsg) > 40 {
					errMsg = errMsg[:37] + "..."
				}
			}
			avail = ErrorStyle.Render(fmt.Sprintf(" error: %s", errMsg))
		} else {
			avail = SuccessStyle.Render(fmt.Sprintf(" %d packages", ms.count))
		}

		line := fmt.Sprintf("%s%s%s", cursor, ManagerTagStyle.Render(name), avail)
		if i == a.dashCursor {
			line = SelectedItemStyle.Render(fmt.Sprintf("%s%s", cursor, name)) + avail
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Show error detail for selected manager
	if a.dashCursor < len(a.managers) {
		ms := a.managers[a.dashCursor]
		if ms.err != nil {
			b.WriteString("\n")
			b.WriteString(ErrorStyle.Render(fmt.Sprintf("  Error: %v", ms.err)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("j/k: navigate  enter: browse  /: search  r: reload  q: quit"))
	return b.String()
}

func (a *App) viewInstalled() string {
	var b strings.Builder
	ms := a.managers[a.selectedMgr]
	header := fmt.Sprintf("Installed: %s", ms.mgr.Name())
	b.WriteString(HeaderStyle.Render(header))
	b.WriteString("\n\n")

	if len(a.installedPkgs) == 0 {
		b.WriteString(HelpStyle.Render("  loading packages..."))
		b.WriteString("\n")
	} else {
		b.WriteString(a.installedTable.View())
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("j/k: navigate  enter: detail  u: upgrade  x: remove  esc: back"))
	return b.String()
}

func (a *App) viewSearch() string {
	var b strings.Builder

	// Show focus indicator and filter state in header
	headerText := "Search Packages"
	if a.searchInputFocused {
		headerText += "  [input]"
	} else if len(a.searchPkgs) > 0 {
		headerText += "  [results]"
	}
	if a.searchFilterInstalled {
		headerText += "  [installed only]"
	}
	b.WriteString(HeaderStyle.Render(headerText))
	b.WriteString("\n\n")

	b.WriteString(SearchPromptStyle.Render("❯ "))
	b.WriteString(a.searchInput.View())
	b.WriteString("\n\n")

	if a.searching {
		b.WriteString(a.spinner.View())
		b.WriteString(" searching across all managers...")
		b.WriteString("\n")
	} else if len(a.searchPkgs) > 0 {
		b.WriteString(a.searchTable.View())
	} else if a.searchInput.Value() != "" && !a.searching {
		b.WriteString(HelpStyle.Render("  no results found"))
		b.WriteString("\n")
	} else {
		b.WriteString(HelpStyle.Render("  type a query and press enter"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if a.searchInputFocused {
		b.WriteString(HelpStyle.Render("enter: search  tab: switch to results  esc: back"))
	} else {
		b.WriteString(HelpStyle.Render("j/k: navigate  enter/d: detail  i: install  f: filter installed  tab: switch to input  esc: back"))
	}
	return b.String()
}

func (a *App) viewDetail() string {
	var b strings.Builder
	pkg := a.detailPkg

	b.WriteString(HeaderStyle.Render("Package Detail"))
	b.WriteString("\n\n")

	// Package name with manager tag
	b.WriteString(fmt.Sprintf("  %s  %s\n",
		TitleStyle.Render(pkg.Name),
		ManagerTagStyle.Render(fmt.Sprintf("[%s]", pkg.Manager)),
	))
	b.WriteString("\n")

	// Version info
	if pkg.Version != "" {
		b.WriteString(fmt.Sprintf("  %s  %s", DescStyle.Render("Version:"), VersionStyle.Render(pkg.Version)))
		if pkg.Latest != "" {
			b.WriteString(fmt.Sprintf(" → %s", SuccessStyle.Render(pkg.Latest)))
		}
		b.WriteString("\n")
	}

	// Description
	if pkg.Description != "" {
		b.WriteString(fmt.Sprintf("  %s  %s\n", DescStyle.Render("Description:"), pkg.Description))
	}

	// Status
	status := "not installed"
	if pkg.Installed {
		status = SuccessStyle.Render("installed")
	} else if pkg.Outdated {
		status = WarningStyle.Render("outdated")
	}
	b.WriteString(fmt.Sprintf("  %s  %s\n", DescStyle.Render("Status:"), status))

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("i: install  u: upgrade  x: remove  esc: back"))
	return b.String()
}

func (a *App) viewConfirm() string {
	msg := fmt.Sprintf("%s %s from %s?",
		strings.ToUpper(a.confirmAct),
		SelectedItemStyle.Render(a.confirmPkg.Name),
		ManagerTagStyle.Render(a.confirmPkg.Manager),
	)
	buttons := fmt.Sprintf("[ %s ] [ %s ]",
		SuccessStyle.Render("Yes (y)"),
		ErrorStyle.Render("No (n)"),
	)
	content := fmt.Sprintf("%s\n\n%s", msg, buttons)
	return DialogBoxStyle.Render(content)
}

func (a *App) viewStatus() string {
	left := ""
	if a.loading > 0 {
		left = fmt.Sprintf("%s loading %d manager(s)...", a.spinner.View(), a.loading)
	} else if a.statusMsg != "" {
		left = a.statusMsg
	}
	right := fmt.Sprintf("q: quit")
	padding := a.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if padding < 0 {
		padding = 0
	}
	return StatusBarStyle.Render(
		left + strings.Repeat(" ", padding) + HelpStyle.Render(right),
	)
}

