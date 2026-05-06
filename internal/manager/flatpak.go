package manager

import (
	"bufio"
	"context"
	"os/exec"
	"strings"

	"github.com/calendar/yuman/internal/model"
)

type flatpak struct{}

func NewFlatpak() model.Manager { return &flatpak{} }

func (f *flatpak) Name() string      { return "flatpak" }
func (f *flatpak) Available() bool    { _, err := exec.LookPath("flatpak"); return err == nil }

func (f *flatpak) List(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "flatpak", "list", "--columns=application,version").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseFlatpakList(out), nil
}

func (f *flatpak) Search(ctx context.Context, query string) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "flatpak", "search", query).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parseFlatpakSearch(out), nil
}

func (f *flatpak) Install(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "flatpak", "install", "-y", pkg).Run()
}
func (f *flatpak) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "flatpak", "uninstall", "-y", pkg).Run()
}

func (f *flatpak) Outdated(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "flatpak", "remote-ls", "--updates", "--columns=application,version").CombinedOutput()
	if err != nil {
		return nil, nil
	}
	return parseFlatpakList(out), nil
}

func (f *flatpak) Upgrade(ctx context.Context, pkg string) error {
	if pkg != "" {
		return exec.CommandContext(ctx, "flatpak", "update", "-y", pkg).Run()
	}
	return exec.CommandContext(ctx, "flatpak", "update", "-y").Run()
}

// parseFlatpakList parses tab-separated appid\tversion lines.
func parseFlatpakList(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) >= 1 {
			version := ""
			if len(parts) >= 2 {
				version = strings.TrimSpace(parts[1])
			}
			pkgs = append(pkgs, model.Package{
				Name:      strings.TrimSpace(parts[0]),
				Version:   version,
				Manager:   "flatpak",
				Installed: true,
			})
		}
	}
	return pkgs
}

// parseFlatpakSearch parses `flatpak search` output (tab-separated).
func parseFlatpakSearch(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) >= 1 {
			name := strings.TrimSpace(parts[0])
			version := ""
			desc := ""
			if len(parts) >= 2 {
				version = strings.TrimSpace(parts[1])
			}
			if len(parts) >= 4 {
				desc = strings.TrimSpace(parts[3])
			}
			pkgs = append(pkgs, model.Package{
				Name:        name,
				Version:     version,
				Description: desc,
				Manager:     "flatpak",
			})
		}
	}
	return pkgs
}
