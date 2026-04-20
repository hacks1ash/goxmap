package reference

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// ModuleRoot walks up from dir looking for a go.mod file.
// Returns the module's import path and the absolute directory containing go.mod.
// The returned moduleDir preserves the same path form as the input dir so that
// callers can compare it with paths constructed from the same base (e.g. on
// macOS, t.TempDir returns /var/... while filepath.EvalSymlinks resolves to
// /private/var/...; we intentionally avoid EvalSymlinks to keep the two
// consistent).
func ModuleRoot(dir string) (modulePath, moduleDir string, err error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	cur := abs
	for {
		candidate := filepath.Join(cur, "go.mod")
		if data, readErr := os.ReadFile(candidate); readErr == nil {
			mp := modfile.ModulePath(data)
			if mp == "" {
				return "", "", fmt.Errorf("go.mod at %s has no module directive", candidate)
			}
			return mp, cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", "", fmt.Errorf("no go.mod found above %s", abs)
		}
		cur = parent
	}
}
