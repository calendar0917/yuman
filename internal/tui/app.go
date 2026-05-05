package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/calendar/yuman/internal/backend"
	"github.com/calendar/yuman/internal/backup"
	"github.com/calendar/yuman/internal/config"
	"github.com/calendar/yuman/internal/model"
)

// App is the root TUI model.
type App struct {
	state   viewState
	backend *backend.Backend
	cfg     *config.Config
	width   int
	height  int
	DryRun  bool

	// Dashboard — runtime state tracked per manager for UI
	managers   []managerStatus
	dashCursor int

	// Installed view
	installedTable table.Model
	installedPkgs  []model.Package
	selectedMgr    int

	// Search
	searchInput           textinput.Model
	searchTable           table.Model
	searchPkgs            []model.Package
	searching             bool
	searchLoading         int
	searchInputFocused    bool
	searchFilterInstalled bool

	// Detail
	detailPkg model.Package
	prevState viewState

	// Help
	helpOpen bool

	// Outdated view
	outdatedTable       table.Model
	outdatedPkgs        []model.Package
	outdatedViewLoading int

	// Duplicates view
	duplicatesGroups []model.DuplicateGroup
	duplicatesTable  table.Model

	// Environment view
	env     *model.Environment
	envPath string

	// Confirmation dialog
	confirmOpen bool
	confirmPkg  model.Package
	confirmAct  string
	confirmYes  bool

	// Spinner
	spinner spinner.Model

	// Operation output
	operationActive bool
	operationLog    []string
	operationView   viewport.Model

	// Status
	loading         int
	outdatedLoading int
	statusMsg       string
}

// NewApp creates and initializes the TUI application.
func NewApp(cfg *config.Config) *App {
	b := backend.New(cfg)

	names := b.ManagerNames()
	statuses := make([]managerStatus, 0, len(names))
	for _, name := range names {
		m := b.Manager(name)
		statuses = append(statuses, managerStatus{
			name:          name,
			available:     m.Available(),
			count:         -1,
			outdatedCount: -1,
		})
	}

	si := textinput.New()
	si.Placeholder = "search across all managers..."
	si.CharLimit = 100
	si.SetWidth(40)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	vp := viewport.New()
	vp.SetWidth(80)
	vp.SetHeight(10)

	return &App{
		state:    viewDashboard,
		backend:  b,
		managers: statuses,
		cfg:      cfg,
		searchInput:    si,
		spinner:        sp,
		operationView:  vp,
	}
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Tick,
		a.loadAllManagers(),
	)
}

// loadAllManagers loads installed packages from all available managers in parallel.
func (a *App) loadAllManagers() tea.Cmd {
	for i := range a.managers {
		a.managers[i].outdatedCount = -1
	}

	var cmds []tea.Cmd
	for _, ms := range a.managers {
		if !ms.available {
			continue
		}
		name := ms.name
		a.loading++
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(),
				time.Duration(a.cfg.TimeoutSeconds())*time.Second)
			defer cancel()

			m := a.backend.Manager(name)
			if m == nil {
				return loadedMsg{managerName: name, err: fmt.Errorf("manager %q not found", name)}
			}
			pkgs, err := m.List(ctx)
			return loadedMsg{managerName: name, packages: pkgs, err: err}
		})
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// loadOutdatedCounts fires parallel Outdated() calls for all available managers.
func (a *App) loadOutdatedCounts() tea.Cmd {
	var cmds []tea.Cmd
	for _, ms := range a.managers {
		if !ms.available || ms.count <= 0 {
			continue
		}
		name := ms.name
		a.outdatedLoading++
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(),
				time.Duration(a.cfg.TimeoutSeconds())*time.Second)
			defer cancel()

			m := a.backend.Manager(name)
			if m == nil {
				return outdatedCountMsg{managerName: name, count: 0}
			}
			pkgs, err := m.Outdated(ctx)
			count := 0
			if err == nil {
				count = len(pkgs)
			}
			return outdatedCountMsg{managerName: name, count: count}
		})
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// reloadManager reloads a single manager by index.
func (a *App) reloadManager(idx int) tea.Cmd {
	if idx < 0 || idx >= len(a.managers) {
		return nil
	}
	ms := a.managers[idx]
	if !ms.available {
		return nil
	}
	a.managers[idx].count = -1
	a.managers[idx].outdatedCount = -1
	a.managers[idx].err = nil
	name := ms.name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(a.cfg.TimeoutSeconds())*time.Second)
		defer cancel()

		m := a.backend.Manager(name)
		if m == nil {
			return loadedMsg{managerName: name, err: fmt.Errorf("manager %q not found", name)}
		}
		pkgs, err := m.List(ctx)
		return loadedMsg{managerName: name, packages: pkgs, err: err}
	}
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.operationView.SetWidth(msg.Width - 4)
		a.operationView.SetHeight(a.operationHeight())
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
		for i := range a.managers {
			if a.managers[i].name == msg.managerName {
				if msg.err != nil {
					a.managers[i].count = -2
					a.managers[i].err = msg.err
				} else {
					a.managers[i].count = len(msg.packages)
					a.managers[i].err = nil
				}
				break
			}
		}
		if a.loading == 0 {
			a.statusMsg = "checking for updates..."
			return a, a.loadOutdatedCounts()
		}
		return a, nil

	case outdatedCountMsg:
		a.outdatedLoading--
		for i := range a.managers {
			if a.managers[i].name == msg.managerName {
				a.managers[i].outdatedCount = msg.count
				break
			}
		}
		if a.outdatedLoading <= 0 {
			a.statusMsg = "ready"
		}
		return a, nil

	case searchResultMsg:
		a.searchLoading--
		if msg.err == nil {
			for _, p := range msg.packages {
				if a.backend.IsInstalled(p.Manager, p.Name) {
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

	case outdatedResultMsg:
		a.outdatedViewLoading--
		if msg.err == nil && len(msg.packages) > 0 {
			a.outdatedPkgs = append(a.outdatedPkgs, msg.packages...)
		}
		if a.outdatedViewLoading <= 0 {
			a.updateOutdatedTable()
			a.statusMsg = fmt.Sprintf("%d outdated packages across all managers", len(a.outdatedPkgs))
		}
		return a, nil

	case installedWithOutdatedMsg:
		a.installedPkgs = msg.packages
		for i := range a.installedPkgs {
			if latest, ok := msg.outdated[a.installedPkgs[i].Name]; ok {
				a.installedPkgs[i].Outdated = true
				a.installedPkgs[i].Latest = latest
			}
		}
		a.updateInstalledTable()
		return a, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case operationOutputMsg:
		a.operationLog = append(a.operationLog, string(msg))
		if len(a.operationLog) > 1000 {
			a.operationLog = a.operationLog[1:]
		}
		a.operationView.SetContent(strings.Join(a.operationLog, "\n"))
		a.operationView.GotoBottom()

	case actionDoneMsg:
		a.operationActive = false
		if msg.err != nil {
			a.operationLog = append(a.operationLog,
				ErrorStyle.Render(fmt.Sprintf("✗ %s %s failed: %v", msg.action, msg.pkgName, msg.err)))
			a.statusMsg = fmt.Sprintf("%s %s failed: %v", msg.action, msg.pkgName, msg.err)
		} else {
			a.operationLog = append(a.operationLog,
				SuccessStyle.Render(fmt.Sprintf("✓ %s %s succeeded", msg.action, msg.pkgName)))
			a.statusMsg = fmt.Sprintf("%s %s succeeded", msg.action, msg.pkgName)
		}
		a.operationView.SetContent(strings.Join(a.operationLog, "\n"))
		a.operationView.GotoBottom()
		return a, nil

	case backupMsg:
		if msg.err != nil {
			a.statusMsg = fmt.Sprintf("%s failed: %v", msg.action, msg.err)
		} else {
			a.statusMsg = fmt.Sprintf("%s saved to %s", msg.action, msg.path)
		}
		return a, nil

		case duplicatesMsg:
			a.duplicatesGroups = msg.groups
			a.statusMsg = fmt.Sprintf("found %d duplicate groups across managers", len(msg.groups))
			return a, nil
		}

		return a, tea.Batch(cmds...)
	}

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.helpOpen {
		return a.handleHelpKey(msg)
	}
	if a.operationActive {
		return a.handleOperationKey(msg)
	}
	if a.confirmOpen {
		return a.handleConfirmKey(msg)
	}

	key := msg.String()

	switch key {
	case "ctrl+c":
		return a, tea.Quit
	case "q":
		if a.state != viewSearch {
			return a, tea.Quit
		}
	case "?":
		a.helpOpen = true
		return a, nil
	case "esc":
		if a.state == viewDetail {
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
	case viewOutdated:
		return a.handleOutdatedKey(key, msg)
	}

	return a, nil
}

func (a *App) handleHelpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "?" || key == "esc" || key == "q" {
		a.helpOpen = false
	}
	return a, nil
}

func (a *App) handleOperationKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.operationView, cmd = a.operationView.Update(msg)
	return a, cmd
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
		a.statusMsg = "reloading selected..."
		return a, a.reloadManager(a.dashCursor)
	case "R":
		a.statusMsg = "reloading all..."
		return a, a.loadAllManagers()
	case "o":
		a.state = viewOutdated
		a.statusMsg = "checking for outdated packages..."
		return a, a.loadAllOutdated()
	case "e":
		return a, a.exportSnapshot()
	case "i":
		return a, a.importSnapshot()
	case "D":
		a.state = viewDuplicates
		a.statusMsg = "checking for duplicates..."
		return a, a.loadDuplicates()
	case "v":
		return a.loadEnvironment()
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
			a.openDetail(row[0], a.managers[a.selectedMgr].name)
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

	switch key {
	case "enter", "d":
		if len(a.searchPkgs) > 0 {
			row := a.searchTable.SelectedRow()
			if row != nil && len(row) > 0 {
				a.openDetail(row[0], row[2])
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

func (a *App) handleOutdatedKey(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		a.outdatedTable, _ = a.outdatedTable.Update(msg)
	case "down", "j":
		a.outdatedTable, _ = a.outdatedTable.Update(msg)
	case "esc":
		a.state = viewDashboard
		a.statusMsg = "ready"
		return a, nil
	case "U":
		if len(a.outdatedPkgs) > 0 {
			a.state = viewDashboard
			a.statusMsg = "upgrading all outdated..."
			managersToUpgrade := make(map[string]bool)
			for _, p := range a.outdatedPkgs {
				managersToUpgrade[p.Manager] = true
			}
			var cmds []tea.Cmd
			for mgrName := range managersToUpgrade {
				cmds = append(cmds, a.executeAction("upgrade", model.Package{Name: "", Manager: mgrName}))
			}
			return a, tea.Sequence(cmds...)
		}
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
	a.confirmPkg = model.Package{Name: pkgName, Manager: a.managers[a.selectedMgr].name}
	a.confirmYes = true
}

func (a *App) openDetail(pkgName, mgrName string) {
	var pkg model.Package
	pkg.Name = pkgName
	pkg.Manager = mgrName

	for _, p := range a.installedPkgs {
		if p.Name == pkgName {
			pkg = p
			break
		}
	}
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

// executeAction dispatches a manager operation using the backend.
func (a *App) executeAction(action string, pkg model.Package) tea.Cmd {
	m := a.backend.Manager(pkg.Manager)
	if m == nil {
		return func() tea.Msg {
			return actionDoneMsg{action: action, pkgName: pkg.Name,
				err: fmt.Errorf("manager %q not found", pkg.Manager)}
		}
	}

	if a.DryRun {
		return func() tea.Msg {
			return actionDoneMsg{action: action, pkgName: pkg.Name, err: nil}
		}
	}

	a.operationActive = true
	a.operationLog = nil
	a.operationView.SetContent("")

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(a.cfg.TimeoutSeconds())*time.Second)
		defer cancel()

		var err error
		switch action {
		case "install":
			err = a.backend.Install(ctx, pkg.Manager, pkg.Name)
		case "remove":
			err = a.backend.Remove(ctx, pkg.Manager, pkg.Name)
		case "upgrade":
			err = a.backend.Upgrade(ctx, pkg.Manager, pkg.Name)
		default:
			err = fmt.Errorf("unknown action: %s", action)
		}

		return actionDoneMsg{action: action, pkgName: pkg.Name, err: err}
	}
}

// doSearch starts parallel searches across all managers.
func (a *App) doSearch(query string) tea.Cmd {
	a.searchPkgs = nil
	a.searchLoading = 0
	a.searchFilterInstalled = false

	var cmds []tea.Cmd
	for _, ms := range a.managers {
		if !ms.available {
			continue
		}
		name := ms.name
		a.searchLoading++
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			m := a.backend.Manager(name)
			if m == nil {
				return searchResultMsg{managerName: name}
			}
			pkgs, err := m.Search(ctx, query)
			if err == nil {
				for i := range pkgs {
					pkgs[i].Manager = name
				}
			}
			return searchResultMsg{managerName: name, packages: pkgs, err: err}
		})
	}

	if len(cmds) == 0 {
		a.searching = false
		return nil
	}
	return tea.Batch(cmds...)
}

// buildInstalledTable triggers an async load of installed+outdated packages.
func (a *App) buildInstalledTable() tea.Cmd {
	ms := a.managers[a.selectedMgr]
	a.installedPkgs = nil
	a.updateInstalledTable()

	if !ms.available {
		return nil
	}

	name := ms.name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(a.cfg.TimeoutSeconds())*time.Second)
		defer cancel()

		m := a.backend.Manager(name)
		if m == nil {
			return installedWithOutdatedMsg{packages: nil}
		}

		pkgs, err := m.List(ctx)
		if err != nil {
			return installedWithOutdatedMsg{packages: nil}
		}

		outdated, _ := m.Outdated(ctx)
		omap := make(map[string]string, len(outdated))
		for _, o := range outdated {
			omap[o.Name] = o.Latest
		}

		return installedWithOutdatedMsg{packages: pkgs, outdated: omap}
	}
}

func (a *App) updateInstalledTable() {
	widths := a.computeColumnWidths(installedColSpecs)
	cols := []table.Column{
		{Title: "Name", Width: widths[0]},
		{Title: "Version", Width: widths[1]},
		{Title: "Description", Width: widths[2]},
	}

	rows := make([]table.Row, 0, len(a.installedPkgs))
	for _, p := range a.installedPkgs {
		name := p.Name
		version := p.Version
		desc := p.Description

		if p.Outdated {
			name = WarningStyle.Render(p.Name)
			version = WarningStyle.Render(p.Version + " -> " + p.Latest)
		}

		if len(desc) > 38 {
			desc = desc[:35] + "..."
		}
		rows = append(rows, table.Row{name, version, desc})
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
	widths := a.computeColumnWidths(searchColSpecs)
	cols := []table.Column{
		{Title: "Name", Width: widths[0]},
		{Title: "Version", Width: widths[1]},
		{Title: "Manager", Width: widths[2]},
		{Title: "Description", Width: widths[3]},
	}

	rows := make([]table.Row, 0, len(a.searchPkgs))
	for _, p := range a.searchPkgs {
		if a.searchFilterInstalled && !p.Installed {
			continue
		}
		name := p.Name
		version := p.Version
		mgr := p.Manager
		desc := p.Description

		if p.Installed {
			name = SuccessStyle.Render(p.Name)
		}

		if len(desc) > 38 {
			desc = desc[:35] + "..."
		}
		rows = append(rows, table.Row{name, version, mgr, desc})
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

func (a *App) computeColumnWidths(specs []colSpec) []int {
	available := a.width - 2
	if available < 20 {
		available = 20
	}

	widths := make([]int, len(specs))
	totalFlex := 0
	used := 0

	for i, s := range specs {
		widths[i] = s.min
		used += s.min
		totalFlex += s.flex
	}

	remaining := available - used - (len(specs) - 1)
	if remaining <= 0 || totalFlex == 0 {
		return widths
	}

	for i, s := range specs {
		if s.flex == 0 {
			continue
		}
		extra := remaining * s.flex / totalFlex
		if widths[i]+extra > s.max {
			extra = s.max - widths[i]
		}
		widths[i] += extra
	}

	return widths
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

	title := TitleStyle.Render("yuman")
	b.WriteString(title)
	b.WriteString("\n")

	if a.helpOpen {
		b.WriteString(a.viewHelp())
		b.WriteString("\n")
		b.WriteString(a.viewStatus())
		return tea.NewView(b.String())
	}

	if a.operationActive {
		b.WriteString(a.viewOperation())
		b.WriteString("\n")
		b.WriteString(a.viewStatus())
		return tea.NewView(b.String())
	}

	switch a.state {
	case viewDashboard:
		b.WriteString(a.viewDashboard())
	case viewInstalled:
		b.WriteString(a.viewInstalled())
	case viewSearch:
		b.WriteString(a.viewSearch())
	case viewDetail:
		b.WriteString(a.viewDetail())
	case viewOutdated:
		b.WriteString(a.viewOutdated())
	case viewDuplicates:
		b.WriteString(a.viewDuplicatesView())
	case viewEnvironment:
		b.WriteString(a.viewEnvironmentView())
	}

	if a.confirmOpen {
		b.WriteString("\n")
		b.WriteString(a.viewConfirm())
	}

	b.WriteString("\n")
	b.WriteString(a.viewStatus())

	return tea.NewView(b.String())
}

func (a *App) loadAllOutdated() tea.Cmd {
	a.outdatedPkgs = nil
	a.outdatedViewLoading = 0

	var cmds []tea.Cmd
	for _, ms := range a.managers {
		if !ms.available || ms.count <= 0 {
			continue
		}
		name := ms.name
		a.outdatedViewLoading++
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(),
				time.Duration(a.cfg.TimeoutSeconds())*time.Second)
			defer cancel()

			m := a.backend.Manager(name)
			if m == nil {
				return outdatedResultMsg{managerName: name}
			}
			pkgs, err := m.Outdated(ctx)
			return outdatedResultMsg{managerName: name, packages: pkgs, err: err}
		})
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (a *App) updateOutdatedTable() {
	widths := a.computeColumnWidths(outdatedColSpecs)
	cols := []table.Column{
		{Title: "Name", Width: widths[0]},
		{Title: "Current", Width: widths[1]},
		{Title: "Latest", Width: widths[2]},
		{Title: "Manager", Width: widths[3]},
	}

	rows := make([]table.Row, 0, len(a.outdatedPkgs))
	for _, p := range a.outdatedPkgs {
		rows = append(rows, table.Row{
			WarningStyle.Render(p.Name),
			p.Version,
			WarningStyle.Render(p.Latest),
			p.Manager,
		})
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
		Foreground(colorWarning)
	s.Selected = s.Selected.
		Foreground(colorWarning).
		Bold(true)
	t.SetStyles(s)

	a.outdatedTable = t
}

// loadDuplicates fires a background load of duplicate groups.
func (a *App) loadDuplicates() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(a.cfg.TimeoutSeconds())*time.Second)
		defer cancel()
		return duplicatesMsg{groups: a.backend.FindDuplicates(ctx)}
	}
}

type duplicatesMsg struct {
	groups []model.DuplicateGroup
}

// loadEnvironment reads the current directory's yuman.tools.toml.
func (a *App) loadEnvironment() (tea.Model, tea.Cmd) {
	dir, _ := os.Getwd()
	env, err := backend.ReadEnvironment(dir)
	if err != nil {
		a.statusMsg = "no yuman.tools.toml found in current or parent directories"
		return a, nil
	}
	a.env = env
	a.envPath = env.Path
	a.state = viewEnvironment
	return a, nil
}

// exportSnapshot collects all installed packages and writes them to a TOML file.
func (a *App) exportSnapshot() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(),
			time.Duration(a.cfg.TimeoutSeconds())*time.Second)
		defer cancel()

		loaded := a.backend.LoadAll(ctx)
		var pkgs []model.Package
		for _, p := range loaded {
			pkgs = append(pkgs, p...)
		}

		home, _ := os.UserHomeDir()
		name := fmt.Sprintf("yuman-snapshot-%s.toml", time.Now().Format("20060102"))
		path := filepath.Join(home, name)

		err := backup.ExportTOML(path, pkgs)
		return backupMsg{action: "export", path: path, err: err}
	}
}

// importSnapshot reads a TOML snapshot and attempts to install missing packages.
func (a *App) importSnapshot() tea.Cmd {
	return func() tea.Msg {
		home, _ := os.UserHomeDir()
		name := fmt.Sprintf("yuman-snapshot-%s.toml", time.Now().Format("20060102"))
		path := filepath.Join(home, name)

		snap, err := backup.ImportTOML(path)
		if err != nil {
			return backupMsg{action: "import", path: path, err: err}
		}

		var installed int
		for _, p := range snap.Packages {
			if a.backend.IsInstalled(p.Manager, p.Name) {
				continue
			}
			if m := a.backend.Manager(p.Manager); m != nil {
				ctx, cancel := context.WithTimeout(context.Background(),
					time.Duration(a.cfg.TimeoutSeconds())*time.Second)
				if err := a.backend.Install(ctx, p.Manager, p.Name); err == nil {
					installed++
				}
				cancel()
			}
		}

		if installed == 0 {
			return backupMsg{action: "import", path: path, err: fmt.Errorf("no new packages to install")}
		}
		return backupMsg{action: "import", path: path}
	}
}