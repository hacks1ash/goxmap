package cli

// Tests for the fatal paths in ResolveTypeMismatches.
//
// Because log.Fatalf calls os.Exit(1), these paths cannot be covered by
// normal in-process tests. We use the subprocess self-test pattern: when the
// env var TEST_FATAL is set, the test body runs the fatal code directly; the
// outer test spawns itself as a subprocess and asserts a non-zero exit.
//
// This pattern exercises the lines leading up to log.Fatalf (expectedFn,
// fieldRef assignment, IsNumericType check) and contributes to coverage
// because the subprocess runs with -test.coverprofile too — but even without
// that, the in-process path covers the branch condition evaluations.

import (
	"os"
	"os/exec"
	"testing"

	"github.com/hacks1ash/goxmap/internal/loader"
	"github.com/hacks1ash/goxmap/internal/matcher"
)

// TestResolveTypeMismatches_FatalNarrowingConversion tests the narrowing-cast
// fatal path (int64 → int32 with no converter).
func TestResolveTypeMismatches_FatalNarrowingConversion(t *testing.T) {
	if os.Getenv("TEST_FATAL_NARROWING") == "1" {
		// Running in subprocess: set up and call the fatal path directly.
		dir := writeTempModule(t, "example.com/fatal1", map[string]string{
			"models.go": `package fatal1

type A struct{ X int }
`,
		})
		pctx, err := loader.LoadPackage(dir)
		if err != nil {
			t.Fatalf("LoadPackage: %v", err)
		}
		pair := matcher.FieldPair{
			TypeMismatch: true,
			Src:          loader.StructField{Name: "Value", ElemType: "int64"},
			Dst:          loader.StructField{Name: "Value", ElemType: "int32"},
		}
		pairs := []matcher.FieldPair{pair}
		ResolveTypeMismatches(pctx, pairs, "Src") // must call log.Fatalf → os.Exit(1)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestResolveTypeMismatches_FatalNarrowingConversion", "-test.v")
	cmd.Env = append(os.Environ(), "TEST_FATAL_NARROWING=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected subprocess to exit non-zero for narrowing conversion")
	}
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit; got: %v", err)
	}
}

// TestResolveTypeMismatches_FatalTypeMismatch tests the general type-mismatch
// fatal path (non-numeric incompatible types with no converter).
func TestResolveTypeMismatches_FatalTypeMismatch(t *testing.T) {
	if os.Getenv("TEST_FATAL_MISMATCH") == "1" {
		dir := writeTempModule(t, "example.com/fatal2", map[string]string{
			"models.go": `package fatal2

type A struct{ X int }
`,
		})
		pctx, err := loader.LoadPackage(dir)
		if err != nil {
			t.Fatalf("LoadPackage: %v", err)
		}
		pair := matcher.FieldPair{
			TypeMismatch: true,
			Src:          loader.StructField{Name: "Val", ElemType: "string"},
			Dst:          loader.StructField{Name: "Val", ElemType: "int64"},
		}
		pairs := []matcher.FieldPair{pair}
		ResolveTypeMismatches(pctx, pairs, "Src") // must call log.Fatalf → os.Exit(1)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestResolveTypeMismatches_FatalTypeMismatch", "-test.v")
	cmd.Env = append(os.Environ(), "TEST_FATAL_MISMATCH=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected subprocess to exit non-zero for type mismatch")
	}
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit; got: %v", err)
	}
}
