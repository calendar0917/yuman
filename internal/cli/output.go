package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/calendar/yuman/internal/backend"
	"github.com/calendar/yuman/internal/config"
	"github.com/calendar/yuman/internal/model"
)

// OutputFormat is the output format for CLI commands.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatPlain OutputFormat = "plain"
)

// Ctx returns a background context for CLI operations.
func Ctx() context.Context {
	return context.Background()
}

func getBackend() *backend.Backend {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: config error: %v\n", err)
		cfg = config.DefaultConfig()
	}
	return backend.New(cfg)
}

type outputWriter struct {
	format OutputFormat
}

func (w *outputWriter) write(pkgs []model.Package, headers []string, rowFn func(model.Package) []string) {
	switch w.format {
	case FormatJSON:
		w.writeJSON(pkgs)
	case FormatTable:
		w.writeTable(pkgs, headers, rowFn)
	case FormatPlain:
		w.writePlain(pkgs, rowFn)
	}
}

func (w *outputWriter) writeJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func (w *outputWriter) writeTable(pkgs []model.Package, headers []string, rowFn func(model.Package) []string) {
	cols := make([]int, len(headers))
	for i, h := range headers {
		cols[i] = len(h)
	}
	rows := make([][]string, len(pkgs))
	for i, p := range pkgs {
		row := rowFn(p)
		rows[i] = row
		for j, c := range row {
			if len(c) > cols[j] {
				cols[j] = len(c)
			}
		}
	}
	// Header
	for i, h := range headers {
		fmt.Printf("%-*s  ", cols[i], h)
	}
	fmt.Println()
	// Separator
	for i, c := range cols {
		fmt.Print(strings.Repeat("-", c))
		if i < len(cols)-1 {
			fmt.Print("  ")
		}
	}
	fmt.Println()
	// Rows
	for _, row := range rows {
		for i, c := range row {
			fmt.Printf("%-*s  ", cols[i], c)
		}
		fmt.Println()
	}
}

func (w *outputWriter) writePlain(pkgs []model.Package, rowFn func(model.Package) []string) {
	for _, p := range pkgs {
		fmt.Println(strings.Join(rowFn(p), " "))
	}
}

func (w *outputWriter) writeMap(results map[string][]model.Package) {
	switch w.format {
	case FormatJSON:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(results)
	default:
		for _, name := range sortedKeys(results) {
			pkgs := results[name]
			fmt.Printf("%s (%d packages):\n", name, len(pkgs))
			for _, p := range pkgs {
				fmt.Printf("  %s %s\n", p.Name, p.Version)
			}
			fmt.Println()
		}
	}
}

func sortedKeys(m map[string][]model.Package) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}