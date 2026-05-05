package manager

import "github.com/calendar/yuman/internal/model"

// All returns every registered manager (instantiated with defaults).
// Callers should filter by Available() before use.
func All() []model.Manager {
	return []model.Manager{
		NewPacman(),
		NewApt(),
		NewDnf(),
		NewZypper(),
		NewBrew(),
		NewParu(),
		NewPip(),
		NewNpm(),
		NewPnpm(),
		NewCargo(),
		NewFlatpak(),
	}
}