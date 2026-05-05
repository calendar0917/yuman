package cli

import (
	"github.com/spf13/cobra"

	"github.com/calendar/yuman/internal/model"
)

func pkgRow(p model.Package) []string {
	return []string{p.Name, p.Version, p.Description}
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages across managers",
	RunE: func(cmd *cobra.Command, args []string) error {
		b := getBackend()
		w := &outputWriter{format: OutputFormat(outputFmt)}

		if manager != "" {
			pkgs, err := b.Manager(manager).List(Ctx())
			if err != nil {
				return err
			}
			w.write(pkgs, []string{"Name", "Version", "Description"}, pkgRow)
			return nil
		}

		loaded := b.LoadAll(Ctx())
		w.writeMap(loaded)
		return nil
	},
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search packages across all managers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		b := getBackend()
		w := &outputWriter{format: OutputFormat(outputFmt)}

		if manager != "" {
			m := b.Manager(manager)
			if m == nil {
				return nil
			}
			pkgs, err := m.Search(Ctx(), args[0])
			if err != nil {
				return err
			}
			w.write(pkgs, []string{"Name", "Version", "Manager", "Description"}, func(p model.Package) []string {
				return []string{p.Name, p.Version, p.Manager, p.Description}
			})
			return nil
		}

		pkgs := b.SearchAll(Ctx(), args[0])
		w.write(pkgs, []string{"Name", "Version", "Manager", "Description"}, func(p model.Package) []string {
			return []string{p.Name, p.Version, p.Manager, p.Description}
		})
		return nil
	},
}

var outdatedCmd = &cobra.Command{
	Use:   "outdated",
	Short: "Show outdated packages across all managers",
	RunE: func(cmd *cobra.Command, args []string) error {
		b := getBackend()
		w := &outputWriter{format: OutputFormat(outputFmt)}

		var pkgs []model.Package
		if manager != "" {
			m := b.Manager(manager)
			if m != nil {
				pkgs, _ = m.Outdated(Ctx())
			}
		} else {
			pkgs = b.GetOutdated(Ctx())
		}

		w.write(pkgs, []string{"Name", "Current", "Latest", "Manager"}, func(p model.Package) []string {
			return []string{p.Name, p.Version, p.Latest, p.Manager}
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(outdatedCmd)
}