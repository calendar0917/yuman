package manager

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/calendar/yuman/internal/model"
)

type npm struct{}

func NewNpm() Manager { return &npm{} }

func (n *npm) Name() string      { return "npm" }
func (n *npm) Available() bool    { _, err := exec.LookPath("npm"); return err == nil }

func (n *npm) List(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "npm", "ls", "--global", "--json", "--depth=0").CombinedOutput()
	if err != nil {
		// npm ls returns non-zero when there are unmet deps; ignore
	}
	return parseNpmJSON(out, "npm")
}

func (n *npm) Search(ctx context.Context, query string) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "npm", "search", query, "--json").CombinedOutput()
	if err != nil {
		return nil, err
	}
	var results []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Desc    string `json:"description"`
	}
	if err := json.Unmarshal(out, &results); err != nil {
		return nil, err
	}
	pkgs := make([]model.Package, 0, len(results))
	for _, r := range results {
		pkgs = append(pkgs, model.Package{
			Name:        r.Name,
			Version:     r.Version,
			Description: r.Desc,
			Manager:     "npm",
		})
	}
	return pkgs, nil
}

func (n *npm) Install(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "npm", "install", "-g", pkg).Run()
}
func (n *npm) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "npm", "uninstall", "-g", pkg).Run()
}

func (n *npm) Outdated(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "npm", "outdated", "--global", "--json").CombinedOutput()
	if err != nil {
		// npm outdated returns non-zero when outdated packages exist
	}
	var raw map[string]struct {
		Current string `json:"current"`
		Latest  string `json:"latest"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, nil
	}
	pkgs := make([]model.Package, 0, len(raw))
	for name, info := range raw {
		pkgs = append(pkgs, model.Package{
			Name:     name,
			Version:  info.Current,
			Latest:   info.Latest,
			Outdated: true,
			Manager:  "npm",
		})
	}
	return pkgs, nil
}

func (n *npm) Upgrade(ctx context.Context, pkg string) error {
	if pkg != "" {
		return exec.CommandContext(ctx, "npm", "install", "-g", pkg+"@latest").Run()
	}
	return exec.CommandContext(ctx, "npm", "update", "-g").Run()
}

// parseNpmJSON parses npm/pnpm --json output.
// npm:  { "dependencies": { name: { "version": ... } } }
// pnpm: { "/path/to/store": { "dependencies": { ... } } }
//        or [ { "name": ..., "version": ... } ]
func parseNpmJSON(data []byte, mgr string) ([]model.Package, error) {
	s := strings.TrimSpace(string(data))
	if s == "" || s == "{}" || s == "[]" {
		return nil, nil
	}

	// Try npm format: { "dependencies": { ... } }
	type depMap = map[string]struct {
		Version string `json:"version"`
	}
	var npmOut struct {
		Dependencies depMap `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &npmOut); err == nil && npmOut.Dependencies != nil {
		return depMapToPkgs(npmOut.Dependencies, mgr), nil
	}

	// Try pnpm array format: [ { "name": ..., "version": ... } ]
	var arr []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &arr); err == nil && len(arr) > 0 && arr[0].Name != "" {
		pkgs := make([]model.Package, 0, len(arr))
		for _, item := range arr {
			pkgs = append(pkgs, model.Package{
				Name:      item.Name,
				Version:   item.Version,
				Manager:   mgr,
				Installed: true,
			})
		}
		return pkgs, nil
	}

	// Try pnpm nested object format: { "/path": { "dependencies": { ... } } }
	var objMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &objMap); err == nil {
		for _, raw := range objMap {
			var nested struct {
				Dependencies depMap `json:"dependencies"`
			}
			if err := json.Unmarshal(raw, &nested); err == nil && nested.Dependencies != nil {
				return depMapToPkgs(nested.Dependencies, mgr), nil
			}
		}
	}

	return nil, nil
}

func depMapToPkgs(deps map[string]struct {
	Version string `json:"version"`
}, mgr string) []model.Package {
	pkgs := make([]model.Package, 0, len(deps))
	for name, info := range deps {
		pkgs = append(pkgs, model.Package{
			Name:      name,
			Version:   info.Version,
			Manager:   mgr,
			Installed: true,
		})
	}
	return pkgs
}
