package backend

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/calendar/yuman/internal/config"
	"github.com/calendar/yuman/internal/manager"
	"github.com/calendar/yuman/internal/model"
)

// Backend is the shared orchestration layer used by both CLI and TUI.
type Backend struct {
	cfg      *config.Config
	managers map[string]model.Manager
	mu       sync.RWMutex
	cache    map[string]bool // "manager/name" -> installed
}

// New creates a Backend from config, registering enabled managers.
func New(cfg *config.Config) *Backend {
	all := manager.All()
	reg := make(map[string]model.Manager, len(all))
	for _, m := range all {
		if cfg.IsEnabled(m.Name()) {
			reg[m.Name()] = m
		}
	}
	return &Backend{
		cfg:      cfg,
		managers: reg,
		cache:    make(map[string]bool),
	}
}

// ManagerNames returns sorted names of registered managers.
func (b *Backend) ManagerNames() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	names := make([]string, 0, len(b.managers))
	for name := range b.managers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Manager returns a manager by name, or nil.
func (b *Backend) Manager(name string) model.Manager {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.managers[name]
}

// Cache returns a copy of the installed cache.
func (b *Backend) Cache() map[string]bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make(map[string]bool, len(b.cache))
	for k, v := range b.cache {
		out[k] = v
	}
	return out
}

// IsInstalled checks the cache for a package.
func (b *Backend) IsInstalled(mgrName, pkgName string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.cache[mgrName+"/"+pkgName]
}

// LoadAll loads installed packages from all managers in parallel.
func (b *Backend) LoadAll(ctx context.Context) map[string][]model.Package {
	b.mu.RLock()
	mgrList := make([]model.Manager, 0, len(b.managers))
	for _, m := range b.managers {
		mgrList = append(mgrList, m)
	}
	b.mu.RUnlock()

	var wg sync.WaitGroup
	results := make(map[string][]model.Package)
	var mu sync.Mutex

	for _, m := range mgrList {
		wg.Add(1)
		go func(m model.Manager) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, time.Duration(b.cfg.TimeoutSeconds())*time.Second)
			defer cancel()

			pkgs, err := m.List(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results[m.Name()] = nil
				return
			}
			results[m.Name()] = pkgs

			b.mu.Lock()
			for _, p := range pkgs {
				b.cache[m.Name()+"/"+p.Name] = true
			}
			b.mu.Unlock()
		}(m)
	}

	wg.Wait()
	return results
}

// SearchAll searches across all managers in parallel.
func (b *Backend) SearchAll(ctx context.Context, query string) []model.Package {
	b.mu.RLock()
	mgrList := make([]model.Manager, 0, len(b.managers))
	for _, m := range b.managers {
		mgrList = append(mgrList, m)
	}
	b.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var all []model.Package

	for _, m := range mgrList {
		wg.Add(1)
		go func(m model.Manager) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			pkgs, err := m.Search(ctx, query)
			if err != nil {
				return
			}
			for i := range pkgs {
				pkgs[i].Manager = m.Name()
				if b.IsInstalled(m.Name(), pkgs[i].Name) {
					pkgs[i].Installed = true
				}
			}
			mu.Lock()
			all = append(all, pkgs...)
			mu.Unlock()
		}(m)
	}

	wg.Wait()
	return all
}

// GetOutdated loads outdated packages from all managers in parallel.
func (b *Backend) GetOutdated(ctx context.Context) []model.Package {
	b.mu.RLock()
	mgrList := make([]model.Manager, 0, len(b.managers))
	for _, m := range b.managers {
		mgrList = append(mgrList, m)
	}
	b.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var all []model.Package

	for _, m := range mgrList {
		wg.Add(1)
		go func(m model.Manager) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, time.Duration(b.cfg.TimeoutSeconds())*time.Second)
			defer cancel()

			pkgs, err := m.Outdated(ctx)
			if err != nil {
				return
			}
			mu.Lock()
			all = append(all, pkgs...)
			mu.Unlock()
		}(m)
	}

	wg.Wait()
	return all
}

// Install runs the install action on a manager.
func (b *Backend) Install(ctx context.Context, mgrName, pkg string) error {
	m := b.Manager(mgrName)
	if m == nil {
		return fmt.Errorf("manager %q not found", mgrName)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(b.cfg.TimeoutSeconds())*time.Second)
	defer cancel()
	return m.Install(ctx, pkg)
}

// Remove runs the remove action on a manager.
func (b *Backend) Remove(ctx context.Context, mgrName, pkg string) error {
	m := b.Manager(mgrName)
	if m == nil {
		return fmt.Errorf("manager %q not found", mgrName)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(b.cfg.TimeoutSeconds())*time.Second)
	defer cancel()
	return m.Remove(ctx, pkg)
}

// Upgrade runs the upgrade action on a manager.
func (b *Backend) Upgrade(ctx context.Context, mgrName, pkg string) error {
	m := b.Manager(mgrName)
	if m == nil {
		return fmt.Errorf("manager %q not found", mgrName)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(b.cfg.TimeoutSeconds())*time.Second)
	defer cancel()
	return m.Upgrade(ctx, pkg)
}

// Config returns the backend's config.
func (b *Backend) Config() *config.Config {
	return b.cfg
}