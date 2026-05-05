package backend

import (
	"context"
	"sort"
	"strings"

	"github.com/calendar/yuman/internal/model"
)

// FindDuplicates returns groups of packages installed from multiple managers.
func (b *Backend) FindDuplicates(ctx context.Context) []model.DuplicateGroup {
	loaded := b.LoadAll(ctx)
	if len(loaded) == 0 {
		return nil
	}

	// Build normalized-name → entries map
	groups := make(map[string][]model.DuplicateEntry)
	for mgr, pkgs := range loaded {
		if pkgs == nil {
			continue
		}
		for _, p := range pkgs {
			if ignoreName(p.Name) {
				continue
			}
			norm := normalizeName(p.Name)
			groups[norm] = append(groups[norm], model.DuplicateEntry{
				Manager: mgr,
				Package: p,
			})
		}
	}

	// Filter to groups with 2+ distinct manager families.
	// pacman/yay/paru share the same DB — treat as one family.
	result := make([]model.DuplicateGroup, 0)
	for norm, entries := range groups {
		families := make(map[string]bool)
		for _, e := range entries {
			families[managerFamily(e.Manager)] = true
		}
		if len(families) < 2 {
			continue
		}
		// Use the first real package name as display name
		display := norm
		if len(entries) > 0 {
			display = entries[0].Package.Name
		}
		result = append(result, model.DuplicateGroup{Name: display, Entries: entries})
	}

	sort.Slice(result, func(i, j int) bool {
		return len(result[i].Entries) > len(result[j].Entries)
	})
	return result
}

// normalizeName normalizes a package name for dedup comparison.
func normalizeName(name string) string {
	n := strings.ToLower(name)
	n = strings.ReplaceAll(n, "_", "-")
	n = strings.TrimPrefix(n, "python-")
	n = strings.TrimPrefix(n, "python3-")
	n = strings.TrimPrefix(n, "node-")
	n = strings.TrimPrefix(n, "ruby-")
	n = strings.TrimPrefix(n, "perl-")
	return n
}

// managerFamily groups managers that share the same package database.
func managerFamily(mgr string) string {
	switch mgr {
	case "pacman", "yay", "paru":
		return "arch"
	case "apt", "dpkg":
		return "debian"
	default:
		return mgr
	}
}

func ignoreName(name string) bool {
	// Common false positives: libs that legitimately exist in multiple managers
	// but are not "duplicates" in the user's concern.
	small := []string{"pip", "npm", "yarn", "pnpm", "setuptools", "wheel"}
	for _, s := range small {
		if name == s {
			return true
		}
	}
	return false
}