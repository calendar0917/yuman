package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/calendar/yuman/internal/backup"
	"github.com/calendar/yuman/internal/model"
)

var (
	installCmd = &cobra.Command{
		Use:   "install <package>",
		Short: "Install a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b := getBackend()
			pkg := args[0]
			mgrName := resolveManager(b, pkg)
			if mgrName == "" {
				return fmt.Errorf("manager is required when the package exists in multiple managers")
			}
			return b.Install(Ctx(), mgrName, pkg)
		},
	}

	removeCmd = &cobra.Command{
		Use:   "remove <package>",
		Short: "Remove a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b := getBackend()
			pkg := args[0]
			mgrName := resolveManager(b, pkg)
			if mgrName == "" {
				return fmt.Errorf("manager is required when the package exists in multiple managers")
			}
			return b.Remove(Ctx(), mgrName, pkg)
		},
	}

	upgradeCmd = &cobra.Command{
		Use:   "upgrade [package]",
		Short: "Upgrade packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := getBackend()

			if len(args) > 0 {
				mgrName := manager
				if mgrName == "" {
					mgrName = resolveManager(b, args[0])
					if mgrName == "" {
						return fmt.Errorf("manager is required")
					}
				}
				return b.Upgrade(Ctx(), mgrName, args[0])
			}

			// Upgrade all
			outdated := b.GetOutdated(Ctx())
			byMgr := make(map[string]bool)
			for _, p := range outdated {
				byMgr[p.Manager] = true
			}
			for mgrName := range byMgr {
				if err := b.Upgrade(Ctx(), mgrName, ""); err != nil {
					return err
				}
			}
			return nil
		},
	}

	infoCmd = &cobra.Command{
		Use:   "info <package>",
		Short: "Show package details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b := getBackend()
			mgrName := manager
			if mgrName == "" {
				mgrName = resolveManager(b, args[0])
			}
			m := b.Manager(mgrName)
			if m == nil {
				return fmt.Errorf("manager %q not found", mgrName)
			}
			pkgs, err := m.Search(Ctx(), args[0])
			if err != nil {
				return err
			}
			for _, p := range pkgs {
				if p.Name == args[0] {
					w := &outputWriter{format: OutputFormat(outputFmt)}
					w.write([]model.Package{p}, []string{"Name", "Version", "Manager", "Description", "Installed"},
						func(p model.Package) []string {
							inst := "no"
							if b.IsInstalled(p.Manager, p.Name) {
								inst = "yes"
							}
							return []string{p.Name, p.Version, p.Manager, p.Description, inst}
						})
					return nil
				}
			}
			return fmt.Errorf("package %q not found in %s", args[0], mgrName)
		},
	}

	snapshotCmd = &cobra.Command{
		Use:   "snapshot",
		Short: "Export or import package snapshots",
	}

	snapshotExportCmd = &cobra.Command{
		Use:   "export [path]",
		Short: "Export snapshot to TOML",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := getBackend()
			loaded := b.LoadAll(Ctx())
			var pkgs []model.Package
			for _, p := range loaded {
				pkgs = append(pkgs, p...)
			}

			path := ""
			if len(args) > 0 {
				path = args[0]
			} else {
				home, _ := os.UserHomeDir()
				path = filepath.Join(home, fmt.Sprintf("yuman-snapshot-%s.toml", time.Now().Format("20060102")))
			}

			if err := backup.ExportTOML(path, pkgs); err != nil {
				return err
			}
			fmt.Println("snapshot exported to", path)
			return nil
		},
	}

	snapshotImportCmd = &cobra.Command{
		Use:   "import <path>",
		Short: "Import snapshot from TOML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b := getBackend()
			snap, err := backup.ImportTOML(args[0])
			if err != nil {
				return err
			}

			var installed int
			for _, p := range snap.Packages {
				if b.IsInstalled(p.Manager, p.Name) {
					continue
				}
				if m := b.Manager(p.Manager); m != nil {
					if err := b.Install(Ctx(), p.Manager, p.Name); err == nil {
						installed++
					}
				}
			}
			fmt.Printf("installed %d new packages from snapshot\n", installed)
			return nil
		},
	}

	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Show config path",
		RunE: func(cmd *cobra.Command, args []string) error {
			xdg := os.Getenv("XDG_CONFIG_HOME")
			if xdg != "" {
				fmt.Println(filepath.Join(xdg, "yuman", "yuman.toml"))
			} else {
				home, _ := os.UserHomeDir()
				fmt.Println(filepath.Join(home, ".config", "yuman", "yuman.toml"))
			}
			return nil
		},
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("yuman", versionVerbose)
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)

	snapshotCmd.AddCommand(snapshotExportCmd)
	snapshotCmd.AddCommand(snapshotImportCmd)
	rootCmd.AddCommand(snapshotCmd)
}