package manager

import (
	"context"
	"os/exec"

	"github.com/calendar/yuman/internal/model"
)

type pnpm struct{}

func NewPnpm() model.Manager { return &pnpm{} }

func (p *pnpm) Name() string      { return "pnpm" }
func (p *pnpm) Available() bool    { _, err := exec.LookPath("pnpm"); return err == nil }

func (p *pnpm) List(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "pnpm", "ls", "--global", "--json").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseNpmJSON(out, "pnpm")
}

func (p *pnpm) Search(ctx context.Context, query string) ([]model.Package, error) {
	n := &npm{}
	pkgs, err := n.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	for i := range pkgs {
		pkgs[i].Manager = "pnpm"
		pkgs[i].Description = "[via npm registry] " + pkgs[i].Description
	}
	return pkgs, nil
}

func (p *pnpm) Install(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "pnpm", "add", "-g", pkg).Run()
}
func (p *pnpm) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "pnpm", "remove", "-g", pkg).Run()
}

func (p *pnpm) Outdated(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "pnpm", "outdated", "--global", "--json").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseNpmJSON(out, "pnpm")
}

func (p *pnpm) Upgrade(ctx context.Context, pkg string) error {
	if pkg != "" {
		return exec.CommandContext(ctx, "pnpm", "update", "-g", pkg).Run()
	}
	return exec.CommandContext(ctx, "pnpm", "update", "-g").Run()
}

func (p *pnpm) InstallCmd(pkg string) (string, []string) {
	return "pnpm", []string{"add", "-g", pkg}
}

func (p *pnpm) RemoveCmd(pkg string) (string, []string) {
	return "pnpm", []string{"remove", "-g", pkg}
}

func (p *pnpm) UpgradeCmd(pkg string) (string, []string) {
	if pkg != "" {
		return "pnpm", []string{"add", "-g", pkg + "@latest"}
	}
	return "pnpm", []string{"update", "-g"}
}
