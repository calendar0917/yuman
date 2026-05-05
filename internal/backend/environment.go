package backend

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/calendar/yuman/internal/model"
)

// ReadEnvironment loads a yuman.tools.toml from the given directory or
// walks up to find one.
func ReadEnvironment(dir string) (*model.Environment, error) {
	for {
		path := dir + "/yuman.tools.toml"
		if data, err := os.ReadFile(path); err == nil {
			return parseEnvironment(data, path)
		}
		parent := dir[:strings.LastIndex(dir, "/")]
		if parent == "" || parent == dir {
			return nil, fmt.Errorf("no yuman.tools.toml found")
		}
		dir = parent
	}
}

func parseEnvironment(data []byte, path string) (*model.Environment, error) {
	var env model.Environment
	if err := toml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	env.Path = path
	return &env, nil
}

// InstallEnvironment installs all tools specified in the environment.
func (b *Backend) InstallEnvironment(ctx context.Context, env *model.Environment) (int, error) {
	installed := 0
	for _, tool := range env.Tools {
		if b.IsInstalled(tool.Manager, tool.Name) {
			continue
		}
		if err := b.Install(ctx, tool.Manager, tool.Name); err != nil {
			return installed, fmt.Errorf("install %s from %s: %w", tool.Name, tool.Manager, err)
		}
		installed++
	}
	return installed, nil
}

// WriteEnvironment writes a yuman.tools.toml from installed packages.
func WriteEnvironment(path string, tools []model.ToolSpec) error {
	env := model.Environment{Tools: tools}
	data, err := toml.Marshal(env)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}