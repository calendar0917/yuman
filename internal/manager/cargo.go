package manager

import (
	"bufio"
	"context"
	"os/exec"
	"strings"

	"github.com/calendar/yuman/internal/model"
)

type cargo struct{}

func NewCargo() Manager { return &cargo{} }

func (c *cargo) Name() string      { return "cargo" }
func (c *cargo) Available() bool    { _, err := exec.LookPath("cargo"); return err == nil }

func (c *cargo) List(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "cargo", "install", "--list").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseCargoList(out), nil
}

func (c *cargo) Search(ctx context.Context, query string) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "cargo", "search", query).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parseCargoSearch(out), nil
}

func (c *cargo) Install(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "cargo", "install", pkg).Run()
}
func (c *cargo) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "cargo", "uninstall", pkg).Run()
}

func (c *cargo) Outdated(_ context.Context) ([]model.Package, error) {
	// cargo has no built-in outdated command without cargo-outdated
	return nil, nil
}

func (c *cargo) Upgrade(ctx context.Context, pkg string) error {
	// cargo install re-downloads and installs the latest version
	if pkg != "" {
		return exec.CommandContext(ctx, "cargo", "install", pkg).Run()
	}
	// No built-in upgrade-all for cargo
	return nil
}

// parseCargoList parses `cargo install --list` output.
// Format: name version:\n    location
func parseCargoList(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, " ") {
			continue
		}
		// line: "name v1.2.3:" or "name v1.2.3:"
		line = strings.TrimSuffix(line, ":")
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			version := strings.TrimPrefix(parts[1], "v")
			pkgs = append(pkgs, model.Package{
				Name:      parts[0],
				Version:   version,
				Manager:   "cargo",
				Installed: true,
			})
		}
	}
	return pkgs
}

// parseCargoSearch parses `cargo search` output.
// Format: name = "version"    # description
func parseCargoSearch(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, " ") {
			continue
		}
		parts := strings.SplitN(line, " = ", 2)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		rest := parts[1]
		version := ""
		if idx := strings.Index(rest, "\""); idx >= 0 {
			end := strings.Index(rest[idx+1:], "\"")
			if end >= 0 {
				version = rest[idx+1 : idx+1+end]
			}
		}
		desc := ""
		if hash := strings.Index(line, "# "); hash >= 0 {
			desc = strings.TrimSpace(line[hash+2:])
		}
		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Description: desc,
			Manager:     "cargo",
		})
	}
	return pkgs
}
