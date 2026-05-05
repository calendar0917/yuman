package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/calendar/yuman/internal/model"
)

// Snapshot represents a point-in-time capture of installed packages.
type Snapshot struct {
	Generated time.Time       `json:"generated" toml:"generated"`
	Packages  []model.Package `json:"packages" toml:"packages"`
}

// ExportJSON writes the snapshot as JSON.
func ExportJSON(path string, pkgs []model.Package) error {
	snap := Snapshot{
		Generated: time.Now(),
		Packages:  pkgs,
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snap); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}

// ExportTOML writes the snapshot as TOML.
func ExportTOML(path string, pkgs []model.Package) error {
	snap := Snapshot{
		Generated: time.Now(),
		Packages:  pkgs,
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	data, err := toml.Marshal(snap)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// ImportJSON reads a JSON snapshot.
func ImportJSON(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &snap, nil
}

// ImportTOML reads a TOML snapshot.
func ImportTOML(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var snap Snapshot
	if err := toml.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &snap, nil
}