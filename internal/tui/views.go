package tui

import (
	"fmt"
	"strings"
)

func (a *App) viewHelp() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Help — Keybindings"))
	b.WriteString("\n\n")

	dr := DescStyle.Render

	lines := [][2]string{
		{"Global", ""},
		{"  q / ctrl+c ", dr("quit")},
		{"  esc        ", dr("back to dashboard")},
		{"  /          ", dr("search packages")},
		{"  ?          ", dr("this help")},
		{"", ""},
		{"Dashboard", ""},
		{"  j/k ↑ ↓   ", dr("navigate managers")},
		{"  enter      ", dr("browse installed packages")},
		{"  r          ", dr("reload selected manager")},
		{"  R          ", dr("reload all managers")},
		{"  o          ", dr("view outdated packages")},
		{"  e          ", dr("export snapshot to TOML")},
		{"  i          ", dr("import snapshot from TOML")},
		{"", ""},
		{"Installed Packages", ""},
		{"  j/k ↑ ↓   ", dr("navigate packages")},
		{"  enter      ", dr("package detail")},
		{"  u          ", dr("upgrade package")},
		{"  x          ", dr("remove package")},
		{"", ""},
		{"Search", ""},
		{"  enter      ", dr("execute search / open detail")},
		{"  tab        ", dr("switch input ↔ results")},
		{"  i          ", dr("install selected package")},
		{"  f          ", dr("toggle installed-only filter")},
		{"", ""},
		{"Package Detail", ""},
		{"  i / u / x  ", dr("install / upgrade / remove")},
		{"  esc        ", dr("return")},
	}

	for _, l := range lines {
		if l[0] == "" {
			b.WriteString("\n")
		} else if l[1] == "" {
			b.WriteString(TitleStyle.Render(l[0]))
			b.WriteString("\n")
		} else {
			b.WriteString(fmt.Sprintf("%s  %s\n", l[0], l[1]))
		}
	}

	return DialogBoxStyle.Render(b.String())
}

func (a *App) viewOperation() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Operation Output"))
	b.WriteString("\n\n")

	if len(a.operationLog) == 0 {
		b.WriteString(a.spinner.View())
		b.WriteString(" running...")
	} else {
		b.WriteString(a.operationView.View())
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("j/k: scroll  esc: wait for completion"))
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
	return DialogBoxStyle.Render(fmt.Sprintf("%s\n\n%s", msg, buttons))
}

func (a *App) viewOutdated() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Outdated Packages — All Managers"))
	b.WriteString("\n\n")

	if a.outdatedViewLoading > 0 {
		b.WriteString(a.spinner.View())
		b.WriteString(" checking for outdated packages...")
		b.WriteString("\n")
	} else if len(a.outdatedPkgs) == 0 {
		b.WriteString(SuccessStyle.Render("  all packages up to date!"))
		b.WriteString("\n")
	} else {
		b.WriteString(a.outdatedTable.View())
	}

	b.WriteString("\n")
	if len(a.outdatedPkgs) > 0 {
		b.WriteString(HelpStyle.Render("j/k: navigate  U: upgrade all  esc: back to dashboard"))
	} else {
		b.WriteString(HelpStyle.Render("esc: back to dashboard"))
	}
	return b.String()
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
			if ms.outdatedCount > 0 {
				avail += WarningStyle.Render(fmt.Sprintf(", %d updates", ms.outdatedCount))
			}
		}

		line := fmt.Sprintf("%s%s%s", cursor, ManagerTagStyle.Render(name), avail)
		if i == a.dashCursor {
			line = SelectedItemStyle.Render(fmt.Sprintf("%s%s", cursor, name)) + avail
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	if a.dashCursor < len(a.managers) {
		ms := a.managers[a.dashCursor]
		if ms.err != nil {
			b.WriteString("\n")
			b.WriteString(ErrorStyle.Render(fmt.Sprintf("  Error: %v", ms.err)))
			b.WriteString("\n")
		}
	}

	if a.DryRun {
		b.WriteString("\n")
		b.WriteString(WarningStyle.Render("  ⚠ DRY RUN MODE — no changes will be made"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("j/k: navigate  enter: browse  /: search  r: reload  o: outdated  e: export  i: import  ?: help  q: quit"))
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

	b.WriteString(fmt.Sprintf("  %s  %s\n",
		TitleStyle.Render(pkg.Name),
		ManagerTagStyle.Render(fmt.Sprintf("[%s]", pkg.Manager)),
	))
	b.WriteString("\n")

	if pkg.Version != "" {
		b.WriteString(fmt.Sprintf("  %s  %s", DescStyle.Render("Version:"), VersionStyle.Render(pkg.Version)))
		if pkg.Latest != "" {
			b.WriteString(fmt.Sprintf(" → %s", SuccessStyle.Render(pkg.Latest)))
		}
		b.WriteString("\n")
	}

	if pkg.Description != "" {
		b.WriteString(fmt.Sprintf("  %s  %s\n", DescStyle.Render("Description:"), pkg.Description))
	}

	status := "not installed"
	if pkg.Installed && pkg.Outdated {
		status = WarningStyle.Render("installed (outdated)")
	} else if pkg.Installed {
		status = SuccessStyle.Render("installed")
	} else if pkg.Outdated {
		status = WarningStyle.Render("outdated")
	}
	b.WriteString(fmt.Sprintf("  %s  %s\n", DescStyle.Render("Status:"), status))

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("i: install  u: upgrade  x: remove  esc: back"))
	return b.String()
}