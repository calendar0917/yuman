package manager

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/calendar/yuman/internal/model"
)

type brew struct{}

func NewBrew() model.Manager { return &brew{} }

func (b *brew) Name() string { return "brew" }
func (b *brew) Available() bool {
	// Check common Linux brew paths
	for _, p := range []string{"brew", "/home/linuxbrew/.linuxbrew/bin/brew"} {
		if _, err := exec.LookPath(p); err == nil {
			return true
		}
	}
	return false
}

func (b *brew) List(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "brew", "list", "--formula", "--versions").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseBrewList(out), nil
}

func (b *brew) Search(ctx context.Context, query string) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "brew", "search", query).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parseBrewSearch(out), nil
}

func (b *brew) Install(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "brew", "install", pkg).Run()
}
func (b *brew) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "brew", "uninstall", pkg).Run()
}

func (b *brew) Outdated(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "brew", "outdated", "--json").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	pkgs, err := parseBrewOutdated(out)
	return pkgs, err
}

func (b *brew) Upgrade(ctx context.Context, pkg string) error {
	if pkg != "" {
		return exec.CommandContext(ctx, "brew", "upgrade", pkg).Run()
	}
	return exec.CommandContext(ctx, "brew", "upgrade").Run()
}

// parseBrewList parses "brew list --formula --versions": "name version1 version2..."
func parseBrewList(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "==>") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:      parts[0],
			Version:   parts[len(parts)-1],
			Manager:   "brew",
			Installed: true,
		})
	}
	return pkgs
}

// parseBrewSearch parses "brew search" output. Skip "==> Formulae" / "==> Casks" headers.
func parseBrewSearch(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "==>") {
			continue
		}
		pkgs = append(pkgs, model.Package{Name: line, Manager: "brew"})
	}
	return pkgs
}

// parseBrewOutdated parses "brew outdated --json".
func parseBrewOutdated(data []byte) ([]model.Package, error) {
	s := strings.TrimSpace(string(data))
	if s == "" || s == "[]" {
		return nil, nil
	}
	var raw []struct {
		Name          string `json:"name"`
		Installed     []struct{ Version string `json:"version"` } `json:"installed_versions"`
		CurrentVersion string `json:"current_version"`
		Pinned        bool   `json:"pinned"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("brew outdated JSON: %w", err)
	}
	pkgs := make([]model.Package, 0, len(raw))
	for _, r := range raw {
		version := r.CurrentVersion
		if version == "" && len(r.Installed) > 0 {
			version = r.Installed[len(r.Installed)-1].Version
		}
		pkgs = append(pkgs, model.Package{
			Name:     r.Name,
			Version:  version,
			Latest:   r.CurrentVersion,
			Outdated: true,
			Manager:  "brew",
		})
	}
	return pkgs, nil
}
func (b *brew) InstallCmd(pkg string) (string, []string) {
	return "brew", []string{"install", pkg}
}

func (b *brew) RemoveCmd(pkg string) (string, []string) {
	return "brew", []string{"uninstall", pkg}
}

func (b *brew) UpgradeCmd(pkg string) (string, []string) {
	if pkg != "" {
		return "brew", []string{"upgrade", pkg}
	}
	return "brew", []string{"upgrade"}
}
