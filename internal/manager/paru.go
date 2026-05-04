package manager

import (
	"context"
	"os/exec"

	"github.com/calendar/yuman/internal/model"
)

// paru is an AUR helper that delegates most operations to pacman.
type paru struct {
	cmd string // "paru" or "yay"
}

func NewParu() Manager {
	// Prefer paru, fall back to yay
	if _, err := exec.LookPath("paru"); err == nil {
		return &paru{cmd: "paru"}
	}
	if _, err := exec.LookPath("yay"); err == nil {
		return &paru{cmd: "yay"}
	}
	return &paru{cmd: "paru"} // default, Available() will be false
}

func (p *paru) Name() string {
	if p.cmd == "yay" {
		return "yay"
	}
	return "paru"
}

func (p *paru) Available() bool {
	_, err := exec.LookPath(p.cmd)
	return err == nil
}

func (p *paru) List(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, p.cmd, "-Qs", "").CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parsePacmanSearch(out, p.Name()), nil
}

func (p *paru) Search(ctx context.Context, query string) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, p.cmd, "-Ss", query).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parsePacmanSearch(out, p.Name()), nil
}

func (p *paru) Install(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, p.cmd, "-S", "--noconfirm", pkg).Run()
}

func (p *paru) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, p.cmd, "-R", "--noconfirm", pkg).Run()
}

func (p *paru) Outdated(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, p.cmd, "-Qu").CombinedOutput()
	if err != nil {
		return nil, nil // no updates or error
	}
	return parseSpaceKV(out, p.Name()), nil
}

func (p *paru) Upgrade(ctx context.Context, pkg string) error {
	if pkg != "" {
		return exec.CommandContext(ctx, p.cmd, "-S", "--noconfirm", pkg).Run()
	}
	return exec.CommandContext(ctx, p.cmd, "-Syu", "--noconfirm").Run()
}
