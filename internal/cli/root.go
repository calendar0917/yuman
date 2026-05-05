package cli

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/calendar/yuman/internal/config"
	"github.com/calendar/yuman/internal/tui"
)

var (
	outputFmt      string
	manager        string
	versionStr     = "0.2.0"
	versionVerbose string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "output format: table, json, plain")
	rootCmd.PersistentFlags().StringVarP(&manager, "manager", "m", "", "filter by manager name")
}

// SetVersion sets the version string for the version command.
func SetVersion(verbose, short string) {
	versionVerbose = verbose
	versionStr = short
}

var rootCmd = &cobra.Command{
	Use:   "yuman",
	Short: "A terminal UI package manager for Linux",
	Long: `yuman unifies multiple package managers (pacman, apt, npm, pip, cargo, etc.)
into a single interface — both TUI and CLI.

Run without subcommands to launch the interactive TUI.`,
	Version: versionStr,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: config error: %v\n", err)
			cfg = config.DefaultConfig()
		}
		app := tui.NewApp(cfg)
		p := tea.NewProgram(app)
		_, err = p.Run()
		return err
	},
}

// Execute runs the CLI.
func Execute() error {
	return rootCmd.Execute()
}