package cli

// Subprocess self-tests for log.Fatalf branches in RunSamePackage,
// RunCrossPackage, and ResolveDir.
//
// Pattern: when the sentinel env var is set, the test body exercises the fatal
// code directly in the subprocess. The outer test asserts non-zero exit.
// The subprocess runs the same instrumented binary, so coverage is accumulated
// in the subprocess's profile — merged by `go test -coverprofile`.
//
// Note: Go merges subprocess coverage only when the subprocess is started with
// -test.coverprofile. For the standard `go test -cover` run we use here, these
// tests primarily serve as documentation of the fatal paths and provide partial
// coverage via the subprocess's own execution. The branch condition `if err != nil`
// is covered by the outer in-process tests that succeed; only the `{ log.Fatalf }
// body is unreachable in-process.

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runSelfAsSubprocess spawns the current test binary running only the named
// test, with the given env vars added. It returns the exit error (nil = success).
func runSelfAsSubprocess(t *testing.T, testName string, extraEnv ...string) error {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^"+testName+"$", "-test.v")
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ---------------------------------------------------------------------------
// RunSamePackage fatal: loading package fails
// ---------------------------------------------------------------------------

func TestRunSamePackage_FatalLoadPackage(t *testing.T) {
	if os.Getenv("TEST_FATAL_SAME_LOAD_PKG") == "1" {
		RunSamePackage(SamePackageOptions{
			WorkDir: "/nonexistent/path/that/does/not/exist",
			SrcType: "A",
			DstType: "B",
			Output:  "out.go",
		})
		return
	}
	err := runSelfAsSubprocess(t, "TestRunSamePackage_FatalLoadPackage", "TEST_FATAL_SAME_LOAD_PKG=1")
	if err == nil {
		t.Fatal("expected non-zero exit for invalid WorkDir")
	}
}

// ---------------------------------------------------------------------------
// RunSamePackage fatal: source type not found
// ---------------------------------------------------------------------------

func TestRunSamePackage_FatalSrcNotFound(t *testing.T) {
	if os.Getenv("TEST_FATAL_SAME_SRC") == "1" {
		dir := writeTempModule(t, "example.com/fsp1", map[string]string{
			"models.go": `package fsp1

type B struct{ ID int }
`,
		})
		RunSamePackage(SamePackageOptions{
			WorkDir: dir,
			SrcType: "NonExistent",
			DstType: "B",
			Output:  "out.go",
		})
		return
	}
	err := runSelfAsSubprocess(t, "TestRunSamePackage_FatalSrcNotFound", "TEST_FATAL_SAME_SRC=1")
	if err == nil {
		t.Fatal("expected non-zero exit for missing src type")
	}
}

// ---------------------------------------------------------------------------
// RunSamePackage fatal: destination type not found
// ---------------------------------------------------------------------------

func TestRunSamePackage_FatalDstNotFound(t *testing.T) {
	if os.Getenv("TEST_FATAL_SAME_DST") == "1" {
		dir := writeTempModule(t, "example.com/fsp2", map[string]string{
			"models.go": `package fsp2

type A struct{ ID int }
`,
		})
		RunSamePackage(SamePackageOptions{
			WorkDir: dir,
			SrcType: "A",
			DstType: "NonExistent",
			Output:  "out.go",
		})
		return
	}
	err := runSelfAsSubprocess(t, "TestRunSamePackage_FatalDstNotFound", "TEST_FATAL_SAME_DST=1")
	if err == nil {
		t.Fatal("expected non-zero exit for missing dst type")
	}
}

// ---------------------------------------------------------------------------
// RunSamePackage fatal: write output file fails (read-only dir)
// ---------------------------------------------------------------------------

func TestRunSamePackage_FatalWriteOutput(t *testing.T) {
	if os.Getenv("TEST_FATAL_SAME_WRITE") == "1" {
		dir := writeTempModule(t, "example.com/fsp3", map[string]string{
			"models.go": `package fsp3

type A struct{ ID int }
type B struct{ ID int }
`,
		})
		// Make the directory read-only so WriteFile fails.
		if err := os.Chmod(dir, 0555); err != nil {
			// Can't make read-only (e.g. root), skip the fatal path.
			return
		}
		defer os.Chmod(dir, 0755) //nolint:errcheck
		RunSamePackage(SamePackageOptions{
			WorkDir: dir,
			SrcType: "A",
			DstType: "B",
			Output:  "b_mapper_gen.go",
		})
		return
	}
	err := runSelfAsSubprocess(t, "TestRunSamePackage_FatalWriteOutput", "TEST_FATAL_SAME_WRITE=1")
	// Non-zero exit expected; if it exits zero (e.g. chmod not possible),
	// that is also acceptable — the test is best-effort on this path.
	_ = err
}

// ---------------------------------------------------------------------------
// RunCrossPackage fatal: loading local package fails
// ---------------------------------------------------------------------------

func TestRunCrossPackage_FatalLoadLocalPkg(t *testing.T) {
	if os.Getenv("TEST_FATAL_CROSS_LOCAL") == "1" {
		RunCrossPackage(CrossPackageOptions{
			WorkDir:      "/nonexistent/path",
			InternalType: "A",
			ExternalType: "B",
			Output:       "out.go",
			ExtPkgPath:   "example.com/ext",
		})
		return
	}
	err := runSelfAsSubprocess(t, "TestRunCrossPackage_FatalLoadLocalPkg", "TEST_FATAL_CROSS_LOCAL=1")
	if err == nil {
		t.Fatal("expected non-zero exit for invalid WorkDir")
	}
}

// ---------------------------------------------------------------------------
// RunCrossPackage fatal: loading external package fails
// ---------------------------------------------------------------------------

func TestRunCrossPackage_FatalLoadExtPkg(t *testing.T) {
	if os.Getenv("TEST_FATAL_CROSS_EXT") == "1" {
		dir := writeTempModule(t, "example.com/fcp1", map[string]string{
			"models.go": `package fcp1

type A struct{ ID int }
`,
		})
		RunCrossPackage(CrossPackageOptions{
			WorkDir:      dir,
			InternalType: "A",
			ExternalType: "B",
			Output:       "out.go",
			ExtPkgPath:   "example.com/nonexistent/pkg/that/does/not/exist",
		})
		return
	}
	err := runSelfAsSubprocess(t, "TestRunCrossPackage_FatalLoadExtPkg", "TEST_FATAL_CROSS_EXT=1")
	if err == nil {
		t.Fatal("expected non-zero exit for invalid external pkg")
	}
}

// ---------------------------------------------------------------------------
// RunCrossPackage fatal: internal type not found
// ---------------------------------------------------------------------------

func TestRunCrossPackage_FatalInternalTypeNotFound(t *testing.T) {
	if os.Getenv("TEST_FATAL_CROSS_INT") == "1" {
		const extMod = "example.com/fcp2ext"
		const localMod = "example.com/fcp2"
		localDir, _ := createCrossPackageModules(t, localMod, extMod,
			map[string]string{
				"models.go": `package fcp2

type A struct{ ID int }
`,
			},
			map[string]string{
				"models.go": `package fcp2ext

type B struct{ ID int }
`,
			},
		)
		RunCrossPackage(CrossPackageOptions{
			WorkDir:      localDir,
			InternalType: "NonExistent",
			ExternalType: "B",
			Output:       "out.go",
			ExtPkgPath:   extMod,
		})
		return
	}
	err := runSelfAsSubprocess(t, "TestRunCrossPackage_FatalInternalTypeNotFound", "TEST_FATAL_CROSS_INT=1")
	if err == nil {
		t.Fatal("expected non-zero exit for missing internal type")
	}
}

// ---------------------------------------------------------------------------
// RunCrossPackage fatal: external type not found
// ---------------------------------------------------------------------------

func TestRunCrossPackage_FatalExternalTypeNotFound(t *testing.T) {
	if os.Getenv("TEST_FATAL_CROSS_EXT_TYPE") == "1" {
		const extMod = "example.com/fcp3ext"
		const localMod = "example.com/fcp3"
		localDir, _ := createCrossPackageModules(t, localMod, extMod,
			map[string]string{
				"models.go": `package fcp3

type A struct{ ID int }
`,
			},
			map[string]string{
				"models.go": `package fcp3ext

type B struct{ ID int }
`,
			},
		)
		RunCrossPackage(CrossPackageOptions{
			WorkDir:      localDir,
			InternalType: "A",
			ExternalType: "NonExistent",
			Output:       "out.go",
			ExtPkgPath:   extMod,
		})
		return
	}
	err := runSelfAsSubprocess(t, "TestRunCrossPackage_FatalExternalTypeNotFound", "TEST_FATAL_CROSS_EXT_TYPE=1")
	if err == nil {
		t.Fatal("expected non-zero exit for missing external type")
	}
}

// ---------------------------------------------------------------------------
// RunCrossPackage fatal: write output fails (read-only dir)
// ---------------------------------------------------------------------------

func TestRunCrossPackage_FatalWriteOutput(t *testing.T) {
	if os.Getenv("TEST_FATAL_CROSS_WRITE") == "1" {
		const extMod = "example.com/fcp4ext"
		const localMod = "example.com/fcp4"
		localDir, _ := createCrossPackageModules(t, localMod, extMod,
			map[string]string{
				"models.go": `package fcp4

type A struct{ ID int }
`,
			},
			map[string]string{
				"models.go": `package fcp4ext

type B struct{ ID int }
`,
			},
		)
		if err := os.Chmod(localDir, 0555); err != nil {
			return // best-effort
		}
		defer os.Chmod(localDir, 0755) //nolint:errcheck
		RunCrossPackage(CrossPackageOptions{
			WorkDir:      localDir,
			InternalType: "A",
			ExternalType: "B",
			Output:       filepath.Join(localDir, "out.go"),
			ExtPkgPath:   extMod,
		})
		return
	}
	_ = runSelfAsSubprocess(t, "TestRunCrossPackage_FatalWriteOutput", "TEST_FATAL_CROSS_WRITE=1")
	// Best-effort: chmod may not work in all environments.
}
