package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hacks1ash/goxmap/internal/reference"
)

func writeMod(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.com/m\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}
	aDir := filepath.Join(root, "a")
	bDir := filepath.Join(root, "b")
	for _, d := range []string{aDir, bDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(aDir, "a.go"),
		[]byte("package a\n\ntype User struct { Name string }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bDir, "b.go"),
		[]byte("package b\n\ntype UserDTO struct { Name string }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestResolveRefs_BareAndQualified(t *testing.T) {
	root := writeMod(t)
	workDir := filepath.Join(root, "a")

	src, _ := reference.Parse("User")
	dst, _ := reference.Parse("b.UserDTO")

	rr, err := ResolveRefs(workDir, src, dst)
	if err != nil {
		t.Fatalf("ResolveRefs: %v", err)
	}
	if rr.Src.Info.Name != "User" || rr.Dst.Info.Name != "UserDTO" {
		t.Fatalf("loaded wrong types: %+v / %+v", rr.Src.Info, rr.Dst.Info)
	}
	if rr.OutputDir != workDir {
		t.Fatalf("OutputDir = %q want %q", rr.OutputDir, workDir)
	}
	if rr.OutputPackageName != "a" {
		t.Fatalf("OutputPackageName = %q", rr.OutputPackageName)
	}
}

func TestResolveRefs_BothQualifiedNeitherCurrent(t *testing.T) {
	root := writeMod(t)
	cDir := filepath.Join(root, "c")
	if err := os.MkdirAll(cDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cDir, "c.go"), []byte("package c\n"), 0644); err != nil {
		t.Fatal(err)
	}

	src, _ := reference.Parse("a.User")
	dst, _ := reference.Parse("b.UserDTO")

	if _, err := ResolveRefs(cDir, src, dst); err == nil {
		t.Fatal("expected error when neither side is current package")
	}
}
