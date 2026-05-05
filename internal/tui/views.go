package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (a *App) viewHelp() string {
	var b strings.Builder

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
		{"  ctrl+d/u   ", dr("half-page down/up")},
		{"  enter      ", dr("browse installed packages")},
		{"  r          ", dr("reload selected manager")},
		{"  R          ", dr("reload all managers")},
		{"  o          ", dr("view outdated packages")},
		{"  e          ", dr("export snapshot to TOML")},
		{"  i          ", dr("import snapshot to TOML")},
		{"  D          ", dr("find duplicates")},
		{"  v          ", dr("load environment from .tools.toml")},
		{"", ""},
		{"Installed Packages", ""},
		{"  j/k ↑ ↓   ", dr("navigate packages")},
		{"  ctrl+d/u   ", dr("half-page down/up")},
		{"  /          ", dr("filter packages by name")},
		{"  o          ", dr("view outdated in this manager")},
		{"  enter      ", dr("package detail")},
		{"  u          ", dr("upgrade package")},
		{"  x          ", dr("remove package")},
		{"", ""},
		{"Outdated Packages", ""},
		{"  j/k ↑ ↓   ", dr("navigate packages")},
		{"  enter/d    ", dr("package detail")},
		{"  u          ", dr("upgrade single package")},
		{"  U          ", dr("upgrade all outdated")},
		{"", ""},
		{"Search", ""},
		{"  enter      ", dr("execute search / open detail")},
		{"  tab        ", dr("switch input ↔ results")},
		{"  i          ", dr("install selected package")},
		{"  f          ", dr("toggle installed-only filter")},
		{"  0-9        ", dr("filter by manager (0 = clear)")},
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

	// Use viewport for scrolling
	dlgWidth := a.width - 10
	if dlgWidth < 40 {
		dlgWidth = 40
	}
	if dlgWidth > 60 {
		dlgWidth = 60
	}
	vpHeight := a.height - 8
	if vpHeight < 10 {
		vpHeight = 10
	}

	a.helpViewport.SetContent(b.String())
	a.helpViewport.SetWidth(dlgWidth - 6)
	a.helpViewport.SetHeight(vpHeight)

	dlgStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 3).
		Width(dlgWidth)

	return dlgStyle.Render(a.helpViewport.View())
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
	headerText := "Outdated Packages"
	if a.outdatedMgrFilter != "" {
		headerText += " — " + a.outdatedMgrFilter
	} else {
		headerText += " — All Managers"
	}
	b.WriteString(HeaderStyle.Render(headerText))
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
		b.WriteString(HelpStyle.Render("j/k: move  enter: detail  u: upgrade  U: all  esc: back"))
	} else {
		b.WriteString(HelpStyle.Render("esc: back"))
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

	// Check if pacman is available for Arch grouping
	pacmanAvail := false
	for _, ms := range a.managers {
		if ms.name == "pacman" && ms.available {
			pacmanAvail = true
			break
		}
	}

	// Group managers: show Arch family together
	for i, ms := range a.managers {
		// Skip yay/paru when pacman is available (they share the same DB)
		if pacmanAvail && isArchHelper(ms.name) {
			continue
		}

		cursor := "  "
		isCursor := i == a.dashCursor
		if isCursor {
			cursor = CursorStyle.Render("▸ ")
		}

		name := ms.name
		avail := ""
		if !ms.available {
			avail = ErrorStyle.Render("(not installed)")
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
		if isCursor {
			line = SelectedItemStyle.Render(fmt.Sprintf("%s%s", cursor, name)) + avail
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Show indented AUR helpers after pacman if pacman is available
	if pacmanAvail {
		for _, ms := range a.managers {
			if isArchHelper(ms.name) {
				cursor := "    "
				isCursor := false
				for idx, m := range a.managers {
					if m.name == ms.name && idx == a.dashCursor {
						isCursor = true
						cursor = " " + CursorStyle.Render("▸ ")
						break
					}
				}

				var avail string
				if !ms.available {
					avail = ErrorStyle.Render("(not installed)")
				} else {
					avail = DescStyle.Render("(shared with pacman)")
				}

				name := "  " + ms.name
				line := fmt.Sprintf("%s%s%s", cursor, ManagerTagStyle.Render(name), avail)
				if isCursor {
					line = " " + SelectedItemStyle.Render(fmt.Sprintf("%s%s", cursor, name)) + avail
				}
				b.WriteString(line)
				b.WriteString("\n")
			}
		}
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
	b.WriteString(HelpStyle.Render("j/k: move  enter: browse  /: search  o: outdated  ?: help  q: quit"))
	return b.String()
}

func (a *App) viewInstalled() string {
	var b strings.Builder
	ms := a.managers[a.selectedMgr]
	header := fmt.Sprintf("Installed: %s", ms.name)
	if a.filtering {
		header += "  [filtering]"
	}
	if a.installedFilter.Value() != "" {
		header += "  [" + a.installedFilter.Value() + "]"
	}
	b.WriteString(HeaderStyle.Render(header))
	b.WriteString("\n\n")

	if a.filtering {
		b.WriteString(SearchPromptStyle.Render("/ "))
		b.WriteString(a.installedFilter.View())
		b.WriteString("\n\n")
	}

	if len(a.installedPkgs) == 0 && len(a.installedCached) == 0 {
		b.WriteString(HelpStyle.Render("  loading packages..."))
		b.WriteString("\n")
	} else {
		b.WriteString(a.installedTable.View())
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("/: filter  o: outdated  j/k: move  enter: detail  u: upgrade  x: remove  esc: back"))
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
	if a.searchFilterManager != "" {
		headerText += "  [" + a.searchFilterManager + "]"
	}
	b.WriteString(HeaderStyle.Render(headerText))
	b.WriteString("\n\n")

	b.WriteString(SearchPromptStyle.Render("❯ "))
	b.WriteString(a.searchInput.View())
	b.WriteString("\n")

	// Show manager filter mapping
	availMgrs := a.availableManagerNames()
	if len(availMgrs) > 0 {
		b.WriteString("\n")
		b.WriteString(DescStyle.Render("  filter: [0:all]"))
		for i, name := range availMgrs {
			if i >= 9 {
				break
			}
			mark := fmt.Sprintf(" [%d:%s]", i+1, name)
			if a.searchFilterManager == name {
				b.WriteString(SelectedItemStyle.Render(mark))
			} else {
				b.WriteString(DescStyle.Render(mark))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
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
		b.WriteString(HelpStyle.Render("enter: search  tab: results  esc: back"))
	} else {
		b.WriteString(HelpStyle.Render("j/k: move  ctrl+d/u: half-page  enter: detail  i: install  f: filter  tab: input  esc: back"))
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

func (a *App) viewDuplicatesView() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Duplicate Packages"))
	b.WriteString("\n\n")

	if len(a.duplicatesGroups) == 0 {
		b.WriteString(SuccessStyle.Render("  no duplicates found"))
		b.WriteString("\n")
	} else {
		for _, g := range a.duplicatesGroups {
			b.WriteString(fmt.Sprintf("  %s\n", WarningStyle.Render(g.Name)))
			for _, e := range g.Entries {
				b.WriteString(fmt.Sprintf("    %s  %s\n",
					ManagerTagStyle.Render(e.Manager),
					VersionStyle.Render(e.Package.Version)))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(HelpStyle.Render("esc: back to dashboard"))
	return b.String()
}

func (a *App) viewEnvironmentView() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Environment"))
	b.WriteString("\n\n")

	if a.env == nil {
		b.WriteString(HelpStyle.Render("  no yuman.tools.toml found"))
		b.WriteString("\n")
	} else {
		for _, tool := range a.env.Tools {
			mark := "✗"
			if a.backend.IsInstalled(tool.Manager, tool.Name) {
				mark = SuccessStyle.Render("✓")
			}
			b.WriteString(fmt.Sprintf("  %s  %s  %s (wanted: %s)\n",
				mark, tool.Name, ManagerTagStyle.Render(tool.Manager), VersionStyle.Render(tool.Version)))
		}
		b.WriteString(fmt.Sprintf("\n  config: %s\n", DimStyle.Render(a.envPath)))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("esc: back to dashboard"))
	return b.String()
}