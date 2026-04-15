package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createCrossPackageModules sets up two local Go modules:
//   - "local" module with an internal struct
//   - "external" module with an external struct
//
// The local go.mod uses a replace directive so loader.LoadExternalPackage can
// resolve the external package without a network call.
func createCrossPackageModules(t *testing.T, localMod, extMod string,
	localFiles, extFiles map[string]string) (localDir, extDir string) {
	t.Helper()

	extDir = t.TempDir()
	extGomod := "module " + extMod + "\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(extDir, "go.mod"), []byte(extGomod), 0644); err != nil {
		t.Fatalf("writing ext go.mod: %v", err)
	}
	for name, content := range extFiles {
		if err := os.WriteFile(filepath.Join(extDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("writing ext %s: %v", name, err)
		}
	}

	localDir = t.TempDir()
	// go.mod with a replace directive pointing to the local ext module path.
	localGomod := "module " + localMod + "\n\ngo 1.21\n\nrequire " + extMod + " v0.0.0\n\nreplace " + extMod + " => " + extDir + "\n"
	if err := os.WriteFile(filepath.Join(localDir, "go.mod"), []byte(localGomod), 0644); err != nil {
		t.Fatalf("writing local go.mod: %v", err)
	}
	for name, content := range localFiles {
		if err := os.WriteFile(filepath.Join(localDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("writing local %s: %v", name, err)
		}
	}

	return localDir, extDir
}

// ---------------------------------------------------------------------------
// RunCrossPackage — happy paths (in-process for coverage instrumentation)
// ---------------------------------------------------------------------------

func TestRunCrossPackage_BasicMapping(t *testing.T) {
	const extMod = "example.com/extpkg"
	const localMod = "example.com/localpkg"

	localDir, _ := createCrossPackageModules(t, localMod, extMod,
		map[string]string{
			"models.go": `package localpkg

type Internal struct {
	ID   int
	Name string
}
`,
		},
		map[string]string{
			"models.go": `package extpkg

type External struct {
	ID   int
	Name string
}
`,
		},
	)

	RunCrossPackage(CrossPackageOptions{
		WorkDir:      localDir,
		InternalType: "Internal",
		ExternalType: "External",
		Output:       "external_mapper_gen.go",
		ExtPkgPath:   extMod,
	})

	content := mustReadFile(t, filepath.Join(localDir, "external_mapper_gen.go"))
	if !strings.Contains(content, "MapInternalToExternal") {
		t.Errorf("output missing MapInternalToExternal:\n%s", content)
	}
}

func TestRunCrossPackage_CustomFuncName(t *testing.T) {
	const extMod = "example.com/extpkg2"
	const localMod = "example.com/localpkg2"

	localDir, _ := createCrossPackageModules(t, localMod, extMod,
		map[string]string{
			"models.go": `package localpkg2

type Local struct{ Val int }
`,
		},
		map[string]string{
			"models.go": `package extpkg2

type Remote struct{ Val int }
`,
		},
	)

	RunCrossPackage(CrossPackageOptions{
		WorkDir:      localDir,
		InternalType: "Local",
		ExternalType: "Remote",
		FuncName:     "ConvertLocalToRemote",
		Output:       "remote_mapper_gen.go",
		ExtPkgPath:   extMod,
	})

	content := mustReadFile(t, filepath.Join(localDir, "remote_mapper_gen.go"))
	if !strings.Contains(content, "ConvertLocalToRemote") {
		t.Errorf("output missing ConvertLocalToRemote:\n%s", content)
	}
}

func TestRunCrossPackage_BidirectionalMapping(t *testing.T) {
	const extMod = "example.com/extpkg3"
	const localMod = "example.com/localpkg3"

	localDir, _ := createCrossPackageModules(t, localMod, extMod,
		map[string]string{
			"models.go": `package localpkg3

type Inner struct {
	ID   int
	Name string
}
`,
		},
		map[string]string{
			"models.go": `package extpkg3

type Outer struct {
	ID   int
	Name string
}
`,
		},
	)

	RunCrossPackage(CrossPackageOptions{
		WorkDir:      localDir,
		InternalType: "Inner",
		ExternalType: "Outer",
		Output:       "outer_mapper_gen.go",
		ExtPkgPath:   extMod,
		Bidi:         true,
	})

	content := mustReadFile(t, filepath.Join(localDir, "outer_mapper_gen.go"))
	if !strings.Contains(content, "MapInnerToOuter") {
		t.Errorf("output missing MapInnerToOuter:\n%s", content)
	}
}

func TestRunCrossPackage_UnmatchedFieldWarning(t *testing.T) {
	// Internal has an extra field not present in External → warning on stderr.
	const extMod = "example.com/extpkg4"
	const localMod = "example.com/localpkg4"

	localDir, _ := createCrossPackageModules(t, localMod, extMod,
		map[string]string{
			"models.go": `package localpkg4

type Inner struct {
	ID    int
	Extra string
}
`,
		},
		map[string]string{
			"models.go": `package extpkg4

type Outer struct {
	ID int
}
`,
		},
	)

	// Should complete without panicking.
	RunCrossPackage(CrossPackageOptions{
		WorkDir:      localDir,
		InternalType: "Inner",
		ExternalType: "Outer",
		Output:       "outer_mapper_gen.go",
		ExtPkgPath:   extMod,
	})

	if _, err := os.Stat(filepath.Join(localDir, "outer_mapper_gen.go")); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

func TestRunCrossPackage_SameTypeName(t *testing.T) {
	// When internal and external types share the same name, function names
	// should be disambiguated with the external package name.
	const extMod = "example.com/extpkg6"
	const localMod = "example.com/localpkg6"

	localDir, _ := createCrossPackageModules(t, localMod, extMod,
		map[string]string{
			"models.go": `package localpkg6

type User struct {
	ID   int
	Name string
}
`,
		},
		map[string]string{
			"models.go": `package extpkg6

type User struct {
	ID   int
	Name string
}
`,
		},
	)

	RunCrossPackage(CrossPackageOptions{
		WorkDir:      localDir,
		InternalType: "User",
		ExternalType: "User",
		Output:       "user_mapper_gen.go",
		ExtPkgPath:   extMod,
		Bidi:         true,
	})

	content := mustReadFile(t, filepath.Join(localDir, "user_mapper_gen.go"))

	// Function names should include the external package name for disambiguation.
	if !strings.Contains(content, "MapUserToExtpkg6User") {
		t.Errorf("output missing MapUserToExtpkg6User (disambiguated name):\n%s", content)
	}
	if !strings.Contains(content, "MapExtpkg6UserToUser") {
		t.Errorf("output missing MapExtpkg6UserToUser (disambiguated reverse name):\n%s", content)
	}
}

func TestRunCrossPackage_OptionalUnmatchedField(t *testing.T) {
	// Internal has an optional extra field → no warning.
	const extMod = "example.com/extpkg5"
	const localMod = "example.com/localpkg5"

	localDir, _ := createCrossPackageModules(t, localMod, extMod,
		map[string]string{
			"models.go": `package localpkg5

type Inner struct {
	ID    int
	Extra string ` + "`" + `mapper:"optional"` + "`" + `
}
`,
		},
		map[string]string{
			"models.go": `package extpkg5

type Outer struct {
	ID int
}
`,
		},
	)

	RunCrossPackage(CrossPackageOptions{
		WorkDir:      localDir,
		InternalType: "Inner",
		ExternalType: "Outer",
		Output:       "outer_mapper_gen.go",
		ExtPkgPath:   extMod,
	})

	if _, err := os.Stat(filepath.Join(localDir, "outer_mapper_gen.go")); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}
