package tui

import (
	"fmt"
)

func (a *App) operationHeight() int {
	h := (a.height * 60) / 100
	if h < 5 {
		h = 5
	}
	return h
}

func (a *App) viewStatus() string {
	statusW := a.width - 4
	if statusW < 20 {
		statusW = 20
	}

	left := ""
	if a.loading > 0 {
		left = fmt.Sprintf("%s loading %d manager(s)...", a.spinner.View(), a.loading)
	} else if a.outdatedLoading > 0 {
		left = fmt.Sprintf("%s checking updates...", a.spinner.View())
	} else if a.statusMsg != "" {
		left = a.statusMsg
	}

	if left == "" {
		left = "ready"
	}

	return StatusBarStyle.
		Width(statusW).
		Render(left)
}