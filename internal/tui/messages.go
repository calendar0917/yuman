package tui

import "github.com/calendar/yuman/internal/model"

// viewState tracks which view is active.
type viewState int

const (
	viewDashboard viewState = iota
	viewInstalled
	viewSearch
	viewDetail
	viewHelp
	viewOutdated
	viewDuplicates
	viewEnvironment
)

// colSpec describes a table column's sizing constraints.
type colSpec struct {
	min  int
	flex int
	max  int
}

var installedColSpecs = []colSpec{
	{min: 15, flex: 3, max: 40}, // Name
	{min: 10, flex: 2, max: 25}, // Version
	{min: 10, flex: 5, max: 60}, // Description
}

var searchColSpecs = []colSpec{
	{min: 12, flex: 3, max: 35}, // Name
	{min: 8, flex: 2, max: 20},  // Version
	{min: 6, flex: 1, max: 10},  // Manager
	{min: 10, flex: 4, max: 50}, // Description
}

var outdatedColSpecs = []colSpec{
	{min: 15, flex: 3, max: 35}, // Name
	{min: 10, flex: 2, max: 20}, // Current
	{min: 10, flex: 2, max: 20}, // Latest
	{min: 8, flex: 1, max: 12},  // Manager
}

// managerStatus holds runtime info about a manager.
type managerStatus struct {
	name          string
	available     bool
	count         int // installed count, -1 = loading, -2 = error
	outdatedCount int // number of outdated packages, -1 = not checked
	err           error
}

// loadedMsg carries results from a background manager query.
type loadedMsg struct {
	managerName string
	packages    []model.Package
	err         error
}

// searchResultMsg carries partial search results from a single manager.
type searchResultMsg struct {
	managerName string
	packages    []model.Package
	err         error
}

// outdatedCountMsg carries the count of outdated packages for a manager.
type outdatedCountMsg struct {
	managerName string
	count       int
}

// installedWithOutdatedMsg carries installed packages with outdated info.
type installedWithOutdatedMsg struct {
	packages []model.Package
	outdated map[string]string // name -> latest version
}

// actionDoneMsg reports the result of an install/remove/upgrade action.
type actionDoneMsg struct {
	action  string
	pkgName string
	err     error
}

// operationOutputMsg carries a line of streaming output.
type operationOutputMsg string

// outdatedResultMsg carries outdated packages from a single manager.
type outdatedResultMsg struct {
	managerName string
	packages    []model.Package
	err         error
}

// backupMsg reports the result of a backup/restore operation.
type backupMsg struct {
	action string // "export" or "import"
	path   string
	err    error
}