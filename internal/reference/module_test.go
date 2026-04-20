package reference

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModuleRoot_FindsGoMod(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/test\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}

	modPath, modDir, err := ModuleRoot(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modPath != "example.com/test" {
		t.Errorf("module path = %q", modPath)
	}
	if modDir != dir {
		t.Errorf("module dir = %q want %q", modDir, dir)
	}
}

func TestModuleRoot_NotFound(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := ModuleRoot(dir); err == nil {
		t.Fatal("expected error when no go.mod exists")
	}
}
