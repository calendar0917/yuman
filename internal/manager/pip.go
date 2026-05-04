package manager

import (
	"context"
	"encoding/json"
	"os/exec"

	"github.com/calendar/yuman/internal/model"
)

type pip struct{}

func NewPip() Manager { return &pip{} }

func (p *pip) Name() string      { return "pip" }
func (p *pip) Available() bool    { _, err := exec.LookPath("pip"); return err == nil }

type pipPkg struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (p *pip) List(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "pip", "list", "--format=json").CombinedOutput()
	if err != nil {
		return nil, err
	}
	var raw []pipPkg
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	pkgs := make([]model.Package, 0, len(raw))
	for _, r := range raw {
		pkgs = append(pkgs, model.Package{
			Name:      r.Name,
			Version:   r.Version,
			Manager:   "pip",
			Installed: true,
		})
	}
	return pkgs, nil
}

func (p *pip) Search(ctx context.Context, query string) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "pip", "search", query).CombinedOutput()
	if err != nil {
		// pip search is often disabled; fall back to empty
		_ = out
		return nil, nil
	}
	return parseLines(out, "pip"), nil
}

func (p *pip) Install(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "pip", "install", pkg).Run()
}
func (p *pip) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "pip", "uninstall", "-y", pkg).Run()
}

func (p *pip) Outdated(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "pip", "list", "--outdated", "--format=json").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	var raw []struct {
		pipPkg
		Latest string `json:"latest_version"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	pkgs := make([]model.Package, 0, len(raw))
	for _, r := range raw {
		pkgs = append(pkgs, model.Package{
			Name:    r.Name,
			Version: r.Version,
			Latest:  r.Latest,
			Outdated: true,
			Manager: "pip",
		})
	}
	return pkgs, nil
}

func (p *pip) Upgrade(ctx context.Context, pkg string) error {
	if pkg != "" {
		return exec.CommandContext(ctx, "pip", "install", "--upgrade", pkg).Run()
	}
	// pip has no built-in upgrade-all; upgrade each outdated package
	outdated, err := p.Outdated(ctx)
	if err != nil {
		return err
	}
	for _, op := range outdated {
		if err := exec.CommandContext(ctx, "pip", "install", "--upgrade", op.Name).Run(); err != nil {
			return err
		}
	}
	return nil
}

// parseLines is a generic helper that splits output by newlines and creates
// bare packages (name only, no version parsing) — used as a fallback.
func parseLines(data []byte, mgr string) []model.Package {
	var pkgs []model.Package
	for _, line := range splitLines(string(data)) {
		if line == "" {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:    line,
			Manager: mgr,
		})
	}
	return pkgs
}
