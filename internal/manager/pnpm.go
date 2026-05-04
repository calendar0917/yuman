package manager

import (
	"context"
	"os/exec"

	"github.com/calendar/yuman/internal/model"
)

type pnpm struct{}

func NewPnpm() Manager { return &pnpm{} }

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
	// pnpm search was removed; delegate to npm
	n := &npm{}
	return n.Search(ctx, query)
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
