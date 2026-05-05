package model

import (
	"github.com/pelletier/go-toml/v2"
)

// DuplicateEntry is a single package entry in a duplicate group.
type DuplicateEntry struct {
	Manager string  `json:"manager" toml:"manager"`
	Package Package `json:"package" toml:"package"`
}

// DuplicateGroup represents packages that appear in multiple managers.
type DuplicateGroup struct {
	Name    string           `json:"name" toml:"name"`
	Entries []DuplicateEntry `json:"entries" toml:"entries"`
}

// ToolSpec defines a version-pinned tool in a project environment.
type ToolSpec struct {
	Name    string `toml:"name"`
	Manager string `toml:"manager"`
	Version string `toml:"version"`
}

// Environment is a project-level tool specification loaded from yuman.tools.toml.
type Environment struct {
	Tools []ToolSpec `toml:"tools"`
	Path  string     `toml:"-"` // file path, not serialized
}

// ParseEnvironment parses a yuman.tools.toml byte slice.
func ParseEnvironment(data []byte) (*Environment, error) {
	var env Environment
	if err := toml.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}