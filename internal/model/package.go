package model

// Package represents a package from any manager.
type Package struct {
	Name        string
	Version     string
	Description string
	Manager     string // source manager name (e.g. "pacman", "npm")
	Installed   bool
	Outdated    bool
	Latest      string // latest version, if known
}
