package manager

import (
	"bufio"
	"context"
	"os/exec"
	"strings"

	"github.com/calendar/yuman/internal/model"
)

type apt struct{}

func NewApt() model.Manager { return &apt{} }

func (a *apt) Name() string   { return "apt" }
func (a *apt) Available() bool { _, err := exec.LookPath("apt-get"); return err == nil }

func (a *apt) List(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx,
		"dpkg-query", "-W", "-f=${Package}\t${Version}\t${Description}\n").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseTabSeparated(out, "apt"), nil
}

func (a *apt) Search(ctx context.Context, query string) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "apt-cache", "search", query).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parseAptSearch(out), nil
}

func (a *apt) Install(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "apt-get", "install", "-y", pkg).Run()
}
func (a *apt) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "apt-get", "remove", "-y", pkg).Run()
}

func (a *apt) Outdated(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "apt", "list", "--upgradable").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseAptOutdated(out), nil
}

func (a *apt) Upgrade(ctx context.Context, pkg string) error {
	if pkg != "" {
		return exec.CommandContext(ctx, "apt-get", "install", "--only-upgrade", "-y", pkg).Run()
	}
	return exec.CommandContext(ctx, "apt-get", "upgrade", "-y").Run()
}

// parseAptSearch parses apt-cache search output: "name - description"
func parseAptSearch(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " - ", 2)
		pkg := model.Package{Name: parts[0], Manager: "apt"}
		if len(parts) == 2 {
			pkg.Description = parts[1]
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

// parseAptOutdated parses "apt list --upgradable" output.
// Format: "package/version version [arch]" after the "Listing..." header.
func parseAptOutdated(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "Listing") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		// "firefox/stable 115.0 amd64 [upgradable from: 114.0]"
		name := parts[0]
		if idx := strings.Index(name, "/"); idx >= 0 {
			name = name[:idx]
		}
		pkg := model.Package{
			Name:     name,
			Version:  extractOldVersion(parts),
			Latest:   parts[1],
			Outdated: true,
			Manager:  "apt",
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

func extractOldVersion(parts []string) string {
	for i, p := range parts {
		if p == "from:" && i+1 < len(parts) {
			v := strings.TrimRight(parts[i+1], "]")
			return v
		}
	}
	return ""
}
func (a *apt) InstallCmd(pkg string) (string, []string) {
	return "apt", []string{"install", "-y", pkg}
}

func (a *apt) RemoveCmd(pkg string) (string, []string) {
	return "apt", []string{"remove", "-y", pkg}
}

func (a *apt) UpgradeCmd(pkg string) (string, []string) {
	if pkg != "" {
		return "apt", []string{"install", "--only-upgrade", "-y", pkg}
	}
	return "apt", []string{"upgrade", "-y"}
}
