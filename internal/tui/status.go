package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

func (a *App) operationHeight() int {
	// Conservative: use ~60% of available height
	// Account for: title(1) + header(1) + help(1) + status(1) + border(2) + margins(~3)
	h := (a.height * 60) / 100
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
