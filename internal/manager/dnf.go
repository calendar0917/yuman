package manager

import (
	"bufio"
	"context"
	"os/exec"
	"strings"

	"github.com/calendar/yuman/internal/model"
)

type dnf struct{}

func NewDnf() model.Manager { return &dnf{} }

func (d *dnf) Name() string   { return "dnf" }
func (d *dnf) Available() bool { _, err := exec.LookPath("dnf"); return err == nil }

func (d *dnf) List(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx,
		"rpm", "-qa", "--qf=%{NAME}\\t%{VERSION}\\t%{SUMMARY}\\n").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseTabSeparated(out, "dnf"), nil
}

func (d *dnf) Search(ctx context.Context, query string) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "dnf", "search", query).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parseDnfSearch(out), nil
}

func (d *dnf) Install(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "dnf", "install", "-y", pkg).Run()
}
func (d *dnf) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "dnf", "remove", "-y", pkg).Run()
}

func (d *dnf) Outdated(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "dnf", "check-update").CombinedOutput()
	if err != nil {
		// dnf check-update returns exit 100 when updates are available
		if out == nil {
			return nil, nil
		}
	}
	return parseDnfOutdated(out), nil
}

func (d *dnf) Upgrade(ctx context.Context, pkg string) error {
	if pkg != "" {
		return exec.CommandContext(ctx, "dnf", "upgrade", "-y", pkg).Run()
	}
	return exec.CommandContext(ctx, "dnf", "upgrade", "-y").Run()
}

// parseDnfSearch parses "dnf search" output: "name.arch : description"
func parseDnfSearch(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "===") || strings.HasPrefix(line, "Last metadata") {
			continue
		}
		parts := strings.SplitN(line, " : ", 2)
		name := strings.TrimSpace(parts[0])
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			// Check if the suffix looks like an arch (no spaces)
			suffix := name[idx+1:]
			if isArch(suffix) {
				name = name[:idx]
			}
		}
		pkg := model.Package{Name: name, Manager: "dnf"}
		if len(parts) == 2 {
			pkg.Description = strings.TrimSpace(parts[1])
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

// parseDnfOutdated parses "dnf check-update" output.
// Format: "name.arch version repo" (no "Last metadata" header in check-update)
func parseDnfOutdated(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			suffix := name[idx+1:]
			if isArch(suffix) {
				name = name[:idx]
			}
		}
		pkgs = append(pkgs, model.Package{
			Name:     name,
			Latest:   parts[1],
			Outdated: true,
			Manager:  "dnf",
		})
	}
	return pkgs
}

func isArch(s string) bool {
	switch s {
	case "x86_64", "i686", "aarch64", "noarch", "amd64", "arm64", "ppc64le", "s390x":
		return true
	}
	return false
}