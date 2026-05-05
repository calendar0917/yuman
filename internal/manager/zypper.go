package manager

import (
	"bufio"
	"context"
	"os/exec"
	"strings"

	"github.com/calendar/yuman/internal/model"
)

type zypper struct{}

func NewZypper() model.Manager { return &zypper{} }

func (z *zypper) Name() string   { return "zypper" }
func (z *zypper) Available() bool { _, err := exec.LookPath("zypper"); return err == nil }

func (z *zypper) List(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "zypper", "se", "-i", "-t", "package").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseZypperTable(out, "zypper", true), nil
}

func (z *zypper) Search(ctx context.Context, query string) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "zypper", "se", query).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parseZypperTable(out, "zypper", false), nil
}

func (z *zypper) Install(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "zypper", "install", "-y", pkg).Run()
}
func (z *zypper) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "zypper", "remove", "-y", pkg).Run()
}

func (z *zypper) Outdated(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "zypper", "list-updates").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseZypperUpdates(out), nil
}

func (z *zypper) Upgrade(ctx context.Context, pkg string) error {
	if pkg != "" {
		return exec.CommandContext(ctx, "zypper", "update", "-y", pkg).Run()
	}
	return exec.CommandContext(ctx, "zypper", "update", "-y").Run()
}

// parseZypperTable parses zypper table output:
// Format: "S | Name | Summary | Type" (with header bars)
func parseZypperTable(data []byte, mgr string, installedOnly bool) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inTable := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "S ") || strings.HasPrefix(line, "S|") {
			inTable = true
			continue
		}
		if !inTable {
			continue
		}
		if strings.HasPrefix(line, "--") {
			continue
		}
		parts := splitZyp(line)
		if len(parts) < 3 {
			continue
		}
		pkg := model.Package{
			Name:      parts[1],
			Manager:   mgr,
			Installed: installedOnly,
		}
		if len(parts) > 2 {
			pkg.Description = parts[2]
		}
		// Extract version from name if present (e.g. "package-1.2.3")
		if installedOnly {
			// Version is in a separate column in list-updates, not in se -i
			// zypper se -i doesn't show version; we'd need zypper info for each
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

// parseZypperUpdates parses "zypper list-updates" output.
// Format: "S | Name | Current Version | Available Version | Arch | Repository"
func parseZypperUpdates(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inTable := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "S ") || strings.HasPrefix(line, "S|") {
			inTable = true
			continue
		}
		if !inTable {
			continue
		}
		if strings.HasPrefix(line, "--") {
			continue
		}
		parts := splitZyp(line)
		if len(parts) < 4 {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:     parts[1],
			Version:  parts[2],
			Latest:   parts[3],
			Outdated: true,
			Manager:  "zypper",
		})
	}
	return pkgs
}

// splitZyp splits a zypper table row by "|" separators, trimming whitespace.
func splitZyp(line string) []string {
	var parts []string
	for _, part := range strings.Split(line, "|") {
		parts = append(parts, strings.TrimSpace(part))
	}
	return parts
}
func (z *zypper) InstallCmd(pkg string) (string, []string) {
	return "zypper", []string{"install", "-y", pkg}
}

func (z *zypper) RemoveCmd(pkg string) (string, []string) {
	return "zypper", []string{"remove", "-y", pkg}
}

func (z *zypper) UpgradeCmd(pkg string) (string, []string) {
	if pkg != "" {
		return "zypper", []string{"update", "-y", pkg}
	}
	return "zypper", []string{"update", "-y"}
}
