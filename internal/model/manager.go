package model

import "context"

// Manager defines the interface every package manager adapter must implement.
type Manager interface {
	// Name returns the human-readable name (e.g. "pacman", "npm").
	Name() string
	// Available reports whether this manager is installed on the system.
	Available() bool
	// List returns all locally installed packages.
	List(ctx context.Context) ([]Package, error)
	// Search queries the manager's repository for matching packages.
	Search(ctx context.Context, query string) ([]Package, error)
	// Install installs a package by name.
	Install(ctx context.Context, pkg string) error
	// Remove uninstalls a package by name.
	Remove(ctx context.Context, pkg string) error
	// Outdated returns installed packages that have a newer version available.
	Outdated(ctx context.Context) ([]Package, error)
	// Upgrade upgrades a specific package, or all packages if pkg is empty.
	Upgrade(ctx context.Context, pkg string) error
}