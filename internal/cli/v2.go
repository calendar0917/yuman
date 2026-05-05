package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/calendar/yuman/internal/backend"
	"github.com/calendar/yuman/internal/model"
)

var (
	duplicatesCmd = &cobra.Command{
		Use:   "duplicates",
		Short: "Find packages installed in multiple managers",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := getBackend()
			groups := b.FindDuplicates(Ctx())
			w := &outputWriter{format: OutputFormat(outputFmt)}
			if w.format == FormatJSON {
				w.writeJSON(groups)
				return nil
			}
			if len(groups) == 0 {
				fmt.Println("no duplicates found")
				return nil
			}
			for _, g := range groups {
				fmt.Printf("%s:\n", g.Name)
				for _, e := range g.Entries {
					fmt.Printf("  %-8s  %-12s\n", e.Manager, e.Package.Version)
				}
			}
			return nil
		},
	}

	envCmd = &cobra.Command{
		Use:   "env",
		Short: "Project environment management",
	}

	envInitCmd = &cobra.Command{
		Use:   "init [path]",
		Short: "Create a yuman.tools.toml from installed packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := getBackend()
			dir, _ := os.Getwd()
			if len(args) > 0 {
				dir = args[0]
			}
			path := dir + "/yuman.tools.toml"

			// Interactive prompt for which tools to include
			fmt.Println("Creating yuman.tools.toml...")
			fmt.Println("Add tool specs interactively. Press Ctrl+C to finish.")
			fmt.Println("Format: <name> <manager> <version>")
			fmt.Println("Example: python pip 3.12")
			fmt.Println()

			var tools []model.ToolSpec
			loaded := b.LoadAll(Ctx())
			for mgr, pkgs := range loaded {
				if pkgs == nil {
					continue
				}
				for _, p := range pkgs {
					tools = append(tools, model.ToolSpec{
						Name:    p.Name,
						Manager: p.Manager,
						Version: p.Version,
					})
					_ = mgr
				}
			}

			if len(tools) == 0 {
				return fmt.Errorf("no installed packages found")
			}
			if err := backend.WriteEnvironment(path, tools); err != nil {
				return err
			}
			fmt.Printf("wrote %d tools to %s\n", len(tools), path)
			return nil
		},
	}

	envInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install tools from yuman.tools.toml",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := getBackend()
			dir, _ := os.Getwd()
			env, err := backend.ReadEnvironment(dir)
			if err != nil {
				return err
			}
			n, err := b.InstallEnvironment(Ctx(), env)
			if err != nil {
				return err
			}
			fmt.Printf("installed %d tools\n", n)
			return nil
		},
	}

	envStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show environment status",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := os.Getwd()
			env, err := backend.ReadEnvironment(dir)
			if err != nil {
				fmt.Println("no yuman.tools.toml found")
				return nil
			}
			b := getBackend()
			for _, tool := range env.Tools {
				mark := "✗"
				if b.IsInstalled(tool.Manager, tool.Name) {
					mark = "✓"
				}
				fmt.Printf("  %s %s  %s (wanted: %s)\n", mark, tool.Name, tool.Manager, tool.Version)
			}
			fmt.Printf("\n  config: %s\n", env.Path)
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(duplicatesCmd)
	rootCmd.AddCommand(envCmd)
	envCmd.AddCommand(envInitCmd)
	envCmd.AddCommand(envInstallCmd)
	envCmd.AddCommand(envStatusCmd)
}