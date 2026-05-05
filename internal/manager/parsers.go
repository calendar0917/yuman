package manager

import (
	"bufio"
	"strings"

	"github.com/calendar/yuman/internal/model"
)

// parseTabSeparated parses tab-separated lines into packages.
// Used by apt, dnf, and flatpak adapters.
// Column mapping: col0=Name, col1=Version, col2=Description.
func parseTabSeparated(data []byte, mgr string) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		pkg := model.Package{
			Name:      strings.TrimSpace(parts[0]),
			Manager:   mgr,
			Installed: true,
		}
		if len(parts) >= 2 {
			pkg.Version = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			pkg.Description = strings.TrimSpace(parts[2])
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}