package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

func (a *App) operationHeight() int {
	h := a.height - 6
	if h < 5 {
		h = 5
	}
	return h
}

func (a *App) viewStatus() string {
	left := ""
	if a.loading > 0 {
		left = fmt.Sprintf("%s loading %d manager(s)...", a.spinner.View(), a.loading)
	} else if a.outdatedLoading > 0 {
		left = fmt.Sprintf("%s checking updates...", a.spinner.View())
	} else if a.statusMsg != "" {
		left = a.statusMsg
	}
	return StatusBarStyle.
		Background(lipgloss.Color("236")).
		Render(left)
}
