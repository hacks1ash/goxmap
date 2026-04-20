package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hacks1ash/goxmap/internal/matcher"
	"github.com/hacks1ash/goxmap/internal/reference"
)

func TestRun_SamePackage(t *testing.T) {
	root := writeMod(t) // shared helper from resolve_refs_test.go
	workDir := filepath.Join(root, "a")
	// Replace a.go to add a second struct in the same package so we can map
	// within "a".
	if err := os.WriteFile(filepath.Join(workDir, "a.go"),
		[]byte("package a\n\ntype User struct { Name string }\ntype UserDTO struct { Name string }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	src, _ := reference.Parse("User")
	dst, _ := reference.Parse("UserDTO")

	if err := Run(RunOptions{
		WorkDir:    workDir,
		Src:        src,
		Dst:        dst,
		Output:     "user_dto_mapper_gen.go",
		GetterMode: matcher.GetterModeAuto,
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := filepath.Join(workDir, "user_dto_mapper_gen.go")
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file: %v", err)
	}
	if !strings.Contains(string(data), "func MapUserToUserDTO") {
		t.Fatalf("expected MapUserToUserDTO in output, got:\n%s", data)
	}
}

func TestRun_QualifiedDst_GeneratesFile(t *testing.T) {
	root := writeMod(t) // helper from resolve_refs_test.go
	workDir := filepath.Join(root, "a")

	src, _ := reference.Parse("User")
	dst, _ := reference.Parse("b.UserDTO") // module-relative short form

	if err := Run(RunOptions{
		WorkDir:    workDir,
		Src:        src,
		Dst:        dst,
		Output:     "user_dto_mapper_gen.go",
		GetterMode: matcher.GetterModeAuto,
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := filepath.Join(workDir, "user_dto_mapper_gen.go")
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file: %v", err)
	}
	// Cross-package: file should import package b.
	if !strings.Contains(string(data), "example.com/m/b") {
		t.Fatalf("expected import of example.com/m/b in cross-package output:\n%s", data)
	}
}
