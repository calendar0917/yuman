package manager

import (
	"testing"

	"github.com/calendar/yuman/internal/model"
)

// --- parseSpaceKV ---

func TestParseSpaceKV(t *testing.T) {
	input := []byte("vim 9.0.1\nfirefox 115.0\npython 3.11.4\n")
	pkgs := parseSpaceKV(input, "pacman")

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	assertPkg(t, pkgs[0], "vim", "9.0.1", "pacman")
	assertPkg(t, pkgs[1], "firefox", "115.0", "pacman")
	assertPkg(t, pkgs[2], "python", "3.11.4", "pacman")
	for _, p := range pkgs {
		if !p.Installed {
			t.Errorf("expected Installed=true for %s", p.Name)
		}
	}
}

func TestParseSpaceKVEmpty(t *testing.T) {
	pkgs := parseSpaceKV([]byte(""), "pacman")
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParseSpaceKVBlankLines(t *testing.T) {
	input := []byte("vim 9.0\n\n\n  \nfirefox 115\n")
	pkgs := parseSpaceKV(input, "pacman")
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
}

func TestParseSpaceKVMalformed(t *testing.T) {
	input := []byte("incomplete\nanotherline\nonly-one-field\n")
	pkgs := parseSpaceKV(input, "pacman")
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages from malformed input, got %d", len(pkgs))
	}
}

// --- parsePacmanSearch ---

func TestParsePacmanSearch(t *testing.T) {
	input := []byte(`extra/firefox 115.0.2-1
    Fast, Private & Safe Web Browser
extra/python 3.11.4-1
    Next generation of the python high-level scripting language
community/git 2.41.0-1 [installed]
    the fast distributed version control system
`)
	pkgs := parsePacmanSearch(input, "pacman")

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}

	assertPkg(t, pkgs[0], "firefox", "115.0.2-1", "pacman")
	if pkgs[0].Description != "Fast, Private & Safe Web Browser" {
		t.Errorf("expected description 'Fast, Private & Safe Web Browser', got %q", pkgs[0].Description)
	}
	if pkgs[0].Installed {
		t.Error("firefox should not be marked installed")
	}

	assertPkg(t, pkgs[2], "git", "2.41.0-1", "pacman")
	if !pkgs[2].Installed {
		t.Error("git should be marked installed")
	}
}

func TestParsePacmanSearchRepoPrefix(t *testing.T) {
	input := []byte("extra/mypackage 1.0\n    a package\n")
	pkgs := parsePacmanSearch(input, "pacman")
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Name != "mypackage" {
		t.Errorf("expected name 'mypackage', got %q", pkgs[0].Name)
	}
}

func TestParsePacmanSearchLocalPrefix(t *testing.T) {
	// pacman -Qs "" output uses local/ prefix (no [installed] flag)
	input := []byte(`local/vim 9.0.1
    Vi Improved, a text editor
local/firefox 115.0
    Fast web browser
`)
	pkgs := parsePacmanSearch(input, "pacman")
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].Name != "vim" {
		t.Errorf("expected name 'vim', got %q", pkgs[0].Name)
	}
	if !pkgs[0].Installed {
		t.Error("local/ prefix should set Installed=true")
	}
	if pkgs[0].Description != "Vi Improved, a text editor" {
		t.Errorf("expected description, got %q", pkgs[0].Description)
	}
}

func TestParsePacmanSearchParuManager(t *testing.T) {
	input := []byte("aur/somepkg 2.0\n    AUR package\n")
	pkgs := parsePacmanSearch(input, "paru")
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	assertPkg(t, pkgs[0], "somepkg", "2.0", "paru")
}

// --- parseCargoList ---

func TestParseCargoList(t *testing.T) {
	input := []byte(`ripgrep v13.0.0:
    /home/user/.cargo/bin/rg
fd-find v8.7.0:
    /home/user/.cargo/bin/fd
`)
	pkgs := parseCargoList(input)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	assertPkg(t, pkgs[0], "ripgrep", "13.0.0", "cargo")
	assertPkg(t, pkgs[1], "fd-find", "8.7.0", "cargo")
}

func TestParseCargoListEmpty(t *testing.T) {
	pkgs := parseCargoList([]byte(""))
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}

// --- parseCargoSearch ---

func TestParseCargoSearch(t *testing.T) {
	input := []byte(`ripgrep = "13.0.0"    # A line-oriented search tool
fd-find = "8.7.0"      # A simple, fast alternative to find
`)
	pkgs := parseCargoSearch(input)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	assertPkg(t, pkgs[0], "ripgrep", "13.0.0", "cargo")
	if pkgs[0].Description != "A line-oriented search tool" {
		t.Errorf("expected description 'A line-oriented search tool', got %q", pkgs[0].Description)
	}
	assertPkg(t, pkgs[1], "fd-find", "8.7.0", "cargo")
}

// --- parseFlatpakList ---

func TestParseFlatpakList(t *testing.T) {
	input := []byte("org.mozilla.firefox\t115.0\ncom.visualstudio.code\t1.80.0\n")
	pkgs := parseFlatpakList(input)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	assertPkg(t, pkgs[0], "org.mozilla.firefox", "115.0", "flatpak")
	assertPkg(t, pkgs[1], "com.visualstudio.code", "1.80.0", "flatpak")
}

func TestParseFlatpakListNoVersion(t *testing.T) {
	input := []byte("org.example.app\n")
	pkgs := parseFlatpakList(input)

	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	assertPkg(t, pkgs[0], "org.example.app", "", "flatpak")
}

// --- parseFlatpakSearch ---

func TestParseFlatpakSearch(t *testing.T) {
	input := []byte("org.mozilla.firefox\t115.0\tflathub\tFirefox Web Browser\n")
	pkgs := parseFlatpakSearch(input)

	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	assertPkg(t, pkgs[0], "org.mozilla.firefox", "115.0", "flatpak")
	if pkgs[0].Description != "Firefox Web Browser" {
		t.Errorf("expected description 'Firefox Web Browser', got %q", pkgs[0].Description)
	}
}

// --- parseNpmJSON ---

func TestParseNpmJSON(t *testing.T) {
	input := []byte(`{
		"dependencies": {
			"typescript": { "version": "5.1.6" },
			"prettier": { "version": "3.0.0" }
		}
	}`)
	pkgs, err := parseNpmJSON(input, "npm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	// Map for order-independent checking
	pkgMap := map[string]string{}
	for _, p := range pkgs {
		pkgMap[p.Name] = p.Version
	}

	if pkgMap["typescript"] != "5.1.6" {
		t.Errorf("expected typescript 5.1.6, got %s", pkgMap["typescript"])
	}
	if pkgMap["prettier"] != "3.0.0" {
		t.Errorf("expected prettier 3.0.0, got %s", pkgMap["prettier"])
	}
	for _, p := range pkgs {
		if !p.Installed {
			t.Errorf("expected Installed=true for %s", p.Name)
		}
	}
}

func TestParseNpmJSONEmpty(t *testing.T) {
	input := []byte(`{"dependencies": {}}`)
	pkgs, err := parseNpmJSON(input, "npm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParsePnpmJSONArray(t *testing.T) {
	input := []byte(`[
		{"name": "typescript", "version": "5.1.6"},
		{"name": "prettier", "version": "3.0.0"}
	]`)
	pkgs, err := parseNpmJSON(input, "pnpm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	assertPkg(t, pkgs[0], "typescript", "5.1.6", "pnpm")
	assertPkg(t, pkgs[1], "prettier", "3.0.0", "pnpm")
	if !pkgs[0].Installed {
		t.Error("expected Installed=true for list results")
	}
}

func TestParseNpmJSONInvalid(t *testing.T) {
	input := []byte(`not json`)
	pkgs, err := parseNpmJSON(input, "npm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages for invalid JSON, got %d", len(pkgs))
	}
}

// --- parseLines ---

func TestParseLines(t *testing.T) {
	input := []byte("package-one\npackage-two\n  \npackage-three\n")
	pkgs := parseLines(input, "pip")

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	assertPkg(t, pkgs[0], "package-one", "", "pip")
	assertPkg(t, pkgs[1], "package-two", "", "pip")
	assertPkg(t, pkgs[2], "package-three", "", "pip")
}

// --- helpers ---

func assertPkg(t *testing.T, pkg model.Package, name, version, mgr string) {
	t.Helper()
	if pkg.Name != name {
		t.Errorf("expected name %q, got %q", name, pkg.Name)
	}
	if pkg.Version != version {
		t.Errorf("expected version %q, got %q", version, pkg.Version)
	}
	if pkg.Manager != mgr {
		t.Errorf("expected manager %q, got %q", mgr, pkg.Manager)
	}
}
