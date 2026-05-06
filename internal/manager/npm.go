package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/calendar/yuman/internal/model"
)

type npm struct{}

func NewNpm() model.Manager { return &npm{} }

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
	if err != nil || len(out) == 0 {
		return n.searchRegistry(ctx, query)
	}
	var results []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Desc    string `json:"description"`
	}
	if err := json.Unmarshal(out, &results); err != nil || len(results) == 0 {
		return n.searchRegistry(ctx, query)
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

func (n *npm) searchRegistry(ctx context.Context, query string) ([]model.Package, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/-/v1/search?text=%s&size=10", urlEnc(query))
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(1<<attempt) * time.Second):
			}
		}
		pkgs, retry, err := n.doSearchRegistry(ctx, url)
		if err != nil && retry {
			lastErr = err
			continue
		}
		return pkgs, err
	}
	return nil, lastErr
}

func (n *npm) doSearchRegistry(ctx context.Context, url string) ([]model.Package, bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, false, nil
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 429 || resp.StatusCode == 503 {
		return nil, true, fmt.Errorf("npm registry: %d", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		return nil, false, nil
	}
	var data struct {
		Objects []struct {
			Package struct {
				Name        string `json:"name"`
				Version     string `json:"version"`
				Description string `json:"description"`
			} `json:"package"`
		} `json:"objects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, false, nil
	}
	pkgs := make([]model.Package, 0, len(data.Objects))
	for _, o := range data.Objects {
		pkgs = append(pkgs, model.Package{
			Name:        o.Package.Name,
			Version:     o.Package.Version,
			Description: o.Package.Description,
			Manager:     "npm",
		})
	}
	return pkgs, false, nil
}

func urlEnc(s string) string {
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteRune(c)
		}
	}
	return b.String()
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
