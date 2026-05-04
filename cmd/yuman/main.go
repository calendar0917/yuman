package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/calendar/yuman/internal/config"
	"github.com/calendar/yuman/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: config error: %v\n", err)
		cfg = config.DefaultConfig()
	}

	app := tui.NewApp(cfg)
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
