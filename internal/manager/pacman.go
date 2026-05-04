package manager

import (
	"bufio"
	"context"
	"os/exec"
	"strings"

	"github.com/calendar/yuman/internal/model"
)

type pacman struct{}

func NewPacman() Manager { return &pacman{} }

func (p *pacman) Name() string      { return "pacman" }
func (p *pacman) Available() bool    { _, err := exec.LookPath("pacman"); return err == nil }

func (p *pacman) List(ctx context.Context) ([]model.Package, error) {
	// Use -Qs with empty regex to get descriptions for all installed packages
	out, err := exec.CommandContext(ctx, "pacman", "-Qs", "").CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parsePacmanSearch(out, "pacman"), nil
}

func (p *pacman) Search(ctx context.Context, query string) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "pacman", "-Ss", query).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parsePacmanSearch(out, "pacman"), nil
}

func (p *pacman) Install(ctx context.Context, pkg string) error {
	// Requires root — run yuman with sudo or configure sudoers
	return exec.CommandContext(ctx, "pacman", "-S", "--noconfirm", pkg).Run()
}

func (p *pacman) Remove(ctx context.Context, pkg string) error {
	return exec.CommandContext(ctx, "pacman", "-R", "--noconfirm", pkg).Run()
}

func (p *pacman) Outdated(ctx context.Context) ([]model.Package, error) {
	out, err := exec.CommandContext(ctx, "checkupdates").CombinedOutput()
	if err != nil {
		// checkupdates returns exit 2 when no updates — treat as empty
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			return nil, nil
		}
		return nil, err
	}
	return parseSpaceKV(out, "pacman"), nil
}

func (p *pacman) Upgrade(ctx context.Context, pkg string) error {
	if pkg != "" {
		return exec.CommandContext(ctx, "pacman", "-S", "--noconfirm", pkg).Run()
	}
	return exec.CommandContext(ctx, "pacman", "-Syu", "--noconfirm").Run()
}

// parseSpaceKV parses lines of "name version" into packages.
func parseSpaceKV(data []byte, mgr string) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pkgs = append(pkgs, model.Package{
				Name:      parts[0],
				Version:   parts[1],
				Manager:   mgr,
				Installed: true,
			})
		}
	}
	return pkgs
}

// parsePacmanSearch parses `pacman -Ss` output.
// Format: repo/name version\n    description
func parsePacmanSearch(data []byte, mgr string) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var current *model.Package
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			// description continuation
			if current != nil {
				current.Description = strings.TrimSpace(line)
			}
			continue
		}
		// header line: repo/name version [installed]
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := parts[0]
			installed := false
			// detect repo prefix and check if it's local (installed)
			if idx := strings.LastIndex(name, "/"); idx >= 0 {
				repo := name[:idx]
				name = name[idx+1:]
				if repo == "local" {
					installed = true
				}
			}
			for _, p := range parts {
				if p == "[installed]" {
					installed = true
					break
				}
			}
			pkg := model.Package{
				Name:      name,
				Version:   parts[1],
				Installed: installed,
				Manager:   mgr,
			}
			pkgs = append(pkgs, pkg)
			current = &pkgs[len(pkgs)-1]
		}
	}
	return pkgs
}
