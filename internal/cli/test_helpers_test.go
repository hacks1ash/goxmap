package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempModule creates a temporary directory with a go.mod and the given
// source files. Returns the directory path.
func writeTempModule(t *testing.T, modName string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	gomod := "module " + modName + "\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return dir
}
