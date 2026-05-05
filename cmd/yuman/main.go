package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/calendar/yuman/internal/config"
	"github.com/calendar/yuman/internal/tui"
)

const version = "0.1.0"

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")
	helpFlag := flag.Bool("help", false, "print help and exit")
	dryRunFlag := flag.Bool("dry-run", false, "show what would be done without executing (no system changes)")
	flag.Parse()

	if *versionFlag {
		fmt.Println("yuman v" + version)
		return
	}
	if *helpFlag {
		fmt.Println("yuman — a terminal UI package manager for Linux")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  yuman [flags]")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Keybindings:")
		fmt.Println("  q / ctrl+c   quit")
		fmt.Println("  ?            help overlay")
		fmt.Println("  /            search packages")
		fmt.Println("  esc          back to dashboard")
		fmt.Println("  j/k or ↑↓    navigate")
		fmt.Println("  r            reload selected manager (dashboard)")
		fmt.Println("  R            reload all managers (dashboard)")
		fmt.Println()
		fmt.Println("Config: ~/.config/yuman/yuman.toml")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: config error: %v\n", err)
		cfg = config.DefaultConfig()
	}

	app := tui.NewApp(cfg)
	app.DryRun = *dryRunFlag

	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}