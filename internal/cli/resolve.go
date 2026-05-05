package cli

import "github.com/calendar/yuman/internal/backend"

// resolveManager finds the manager for a package. If exactly one manager has
// the package installed, returns that manager's name. If zero or multiple
// have it, returns "" (caller must disambiguate).
func resolveManager(b *backend.Backend, pkg string) string {
	if manager != "" {
		return manager
	}
	var found []string
	for _, name := range b.ManagerNames() {
		if b.IsInstalled(name, pkg) {
			found = append(found, name)
		}
	}
	if len(found) == 1 {
		return found[0]
	}
	return ""
}