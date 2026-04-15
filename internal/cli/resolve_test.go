package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// ResolveDir
// ---------------------------------------------------------------------------

func TestResolveDir_ExplicitAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	got := ResolveDir(dir)
	if got != dir {
		t.Errorf("ResolveDir(%q) = %q, want %q", dir, got, dir)
	}
}

func TestResolveDir_ExplicitRelativePath(t *testing.T) {
	// "." should resolve to the absolute CWD.
	got := ResolveDir(".")
	abs, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if got != abs {
		t.Errorf("ResolveDir(%q) = %q, want %q", ".", got, abs)
	}
}

func TestResolveDir_EmptyWithGOFILE(t *testing.T) {
	dir := t.TempDir()
	gofile := filepath.Join(dir, "models.go")
	t.Setenv("GOFILE", gofile)

	got := ResolveDir("")
	if got != dir {
		t.Errorf("ResolveDir(\"\") with GOFILE=%q = %q, want %q", gofile, got, dir)
	}
}

func TestResolveDir_EmptyNoGOFILE(t *testing.T) {
	// Clear GOFILE so fallback to CWD applies.
	t.Setenv("GOFILE", "")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	got := ResolveDir("")
	if got != cwd {
		t.Errorf("ResolveDir(\"\") without GOFILE = %q, want %q", got, cwd)
	}
}

func TestResolveDir_GOFILEWithSubdirFile(t *testing.T) {
	// GOFILE with nested path — dir should be the parent directory.
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	gofile := filepath.Join(sub, "gen.go")
	t.Setenv("GOFILE", gofile)

	got := ResolveDir("")
	if got != sub {
		t.Errorf("ResolveDir(\"\") with nested GOFILE = %q, want %q", got, sub)
	}
}
