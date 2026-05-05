package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calendar/yuman/internal/model"
)

func testPkgs() []model.Package {
	return []model.Package{
		{Name: "vim", Version: "9.0.1", Manager: "pacman", Installed: true},
		{Name: "typescript", Version: "5.1.6", Manager: "npm", Installed: true},
		{Name: "ripgrep", Version: "13.0.0", Manager: "cargo", Installed: true, Outdated: true, Latest: "14.0.0"},
	}
}

func TestExportImportJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")

	pkgs := testPkgs()
	if err := ExportJSON(path, pkgs); err != nil {
		t.Fatalf("export: %v", err)
	}

	snap, err := ImportJSON(path)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	if len(snap.Packages) != len(pkgs) {
		t.Fatalf("expected %d packages, got %d", len(pkgs), len(snap.Packages))
	}
	if time.Since(snap.Generated) > time.Minute {
		t.Error("generated time should be recent")
	}

	for i, p := range pkgs {
		got := snap.Packages[i]
		if got.Name != p.Name || got.Version != p.Version || got.Manager != p.Manager {
			t.Errorf("package %d: expected %+v, got %+v", i, p, got)
		}
	}
}

func TestExportImportTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.toml")

	pkgs := testPkgs()
	if err := ExportTOML(path, pkgs); err != nil {
		t.Fatalf("export: %v", err)
	}

	snap, err := ImportTOML(path)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	if len(snap.Packages) != len(pkgs) {
		t.Fatalf("expected %d packages, got %d", len(pkgs), len(snap.Packages))
	}
}

func TestImportMissingFile(t *testing.T) {
	_, err := ImportJSON("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestExportNestedDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "snap.json")

	pkgs := testPkgs()
	if err := ExportJSON(path, pkgs); err != nil {
		t.Fatalf("export to nested dir: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file should exist: %v", err)
	}
}

func TestExportEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")

	if err := ExportJSON(path, nil); err != nil {
		t.Fatalf("export empty: %v", err)
	}

	snap, err := ImportJSON(path)
	if err != nil {
		t.Fatalf("import empty: %v", err)
	}

	if len(snap.Packages) != 0 {
		t.Errorf("expected 0 packages, got %d", len(snap.Packages))
	}
}