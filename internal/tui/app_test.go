package tui

import (
	"testing"

	"github.com/calendar/yuman/internal/config"
)

func TestComputeColumnWidthsWide(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	app.width = 120

	specs := []colSpec{
		{min: 15, flex: 3, max: 40},
		{min: 10, flex: 2, max: 25},
		{min: 10, flex: 5, max: 60},
	}
	widths := app.computeColumnWidths(specs)

	if len(widths) != 3 {
		t.Fatalf("expected 3 widths, got %d", len(widths))
	}
	for i, w := range widths {
		if w < specs[i].min {
			t.Errorf("column %d: width %d < min %d", i, w, specs[i].min)
		}
		if w > specs[i].max {
			t.Errorf("column %d: width %d > max %d", i, w, specs[i].max)
		}
	}
	if widths[0] > widths[1] && widths[2] > widths[0] {
		// flex: Name=3, Version=2, Desc=5 — Desc should be widest
	} else {
		t.Errorf("expected flex-based ordering, got widths=%v", widths)
	}
}

func TestComputeColumnWidthsNarrow(t *testing.T) {
	app := NewApp(config.DefaultConfig())
	app.width = 40

	specs := []colSpec{
		{min: 15, flex: 3, max: 40},
		{min: 10, flex: 2, max: 25},
		{min: 10, flex: 5, max: 60},
	}
	widths := app.computeColumnWidths(specs)

	for i, w := range widths {
		if w < specs[i].min {
			t.Errorf("column %d: width %d < min %d", i, w, specs[i].min)
		}
	}
	if widths[0]+widths[1]+widths[2]+3 > 40 {
		t.Errorf("total width %d exceeds terminal width 40", widths[0]+widths[1]+widths[2]+2)
	}
}

func TestNewApp(t *testing.T) {
	cfg := config.DefaultConfig()
	app := NewApp(cfg)

	if app.state != viewDashboard {
		t.Error("expected initial state to be dashboard")
	}
	if len(app.managers) == 0 {
		t.Error("expected at least one manager")
	}
	if app.installedCache == nil {
		t.Error("expected installed cache to be initialized")
	}
	if app.searchInput.Placeholder == "" {
		t.Error("expected search placeholder")
	}
	if app.DryRun {
		t.Error("expected DryRun to be false by default")
	}
}

func TestNewAppWithDisabledManagers(t *testing.T) {
	cfg := config.DefaultConfig()
	for k := range cfg.Managers {
		cfg.Managers[k] = config.ManagerConfig{Enabled: false}
	}
	app := NewApp(cfg)
	if len(app.managers) != 0 {
		t.Errorf("expected 0 managers when all disabled, got %d", len(app.managers))
	}
}