package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binaryPath holds the path to the compiled goxmap binary built in TestMain.
var binaryPath string

// TestMain builds the goxmap binary once, runs all tests, then cleans up.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "goxmap-test")
	if err != nil {
		panic("creating temp dir: " + err.Error())
	}

	binaryPath = filepath.Join(tmpDir, "goxmap")

	// Build the binary from the project root.
	root := getProjectRoot()
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

// getProjectRoot navigates from internal/cli/ up two levels to the module root.
func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// filename is .../internal/cli/cli_integration_test.go
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// createTempModule creates a temporary Go module with the given files.
// gomod is written automatically; files is a map of relative path → content.
func createTempModule(t *testing.T, modName string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	gomod := "module " + modName + "\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	return dir
}

// runMapperGen runs the goxmap binary with the given args and returns
// stdout, stderr and the process exit code.
func runMapperGen(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			exitCode = e.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// readFile reads the entire content of a file, failing the test on error.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// Happy path integration tests
// ---------------------------------------------------------------------------

func TestCLI_SamePackageMapping(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type A struct {
	ID   int
	Name string
}

type B struct {
	ID   int
	Name string
}
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-dir", dir)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}

	outPath := filepath.Join(dir, "b_mapper_gen.go")
	content := readFile(t, outPath)

	if !strings.Contains(content, "MapAToB") {
		t.Errorf("output missing MapAToB function:\n%s", content)
	}
	if !strings.Contains(content, "dst.ID = src.ID") {
		t.Errorf("output missing dst.ID = src.ID:\n%s", content)
	}
	if !strings.Contains(content, "dst.Name = src.Name") {
		t.Errorf("output missing dst.Name = src.Name:\n%s", content)
	}
}

func TestCLI_IgnoreTag(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type A struct {
	ID     int
	Secret string
}

type B struct {
	ID     int
	Secret string ` + "`" + `mapper:"ignore"` + "`" + `
}
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-dir", dir)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}

	content := readFile(t, filepath.Join(dir, "b_mapper_gen.go"))
	if strings.Contains(content, "dst.Secret") {
		t.Errorf("output should not contain dst.Secret (ignored field):\n%s", content)
	}
	if !strings.Contains(content, "dst.ID = src.ID") {
		t.Errorf("output missing dst.ID = src.ID:\n%s", content)
	}
}

func TestCLI_OptionalTag(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type A struct {
	ID int
}

type B struct {
	ID    int
	Extra string ` + "`" + `mapper:"optional"` + "`" + `
}
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-dir", dir)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}
	// Optional unmatched field must NOT produce a warning.
	if strings.Contains(stderr, "warning") && strings.Contains(stderr, "Extra") {
		t.Errorf("optional field should not produce warning; stderr: %s", stderr)
	}
}

func TestCLI_NumericWidening(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type A struct {
	Value int32
}

type B struct {
	Value int64
}
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-dir", dir)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}

	content := readFile(t, filepath.Join(dir, "b_mapper_gen.go"))
	if !strings.Contains(content, "int64(src.Value)") {
		t.Errorf("output missing int64 cast; content:\n%s", content)
	}
}

func TestCLI_EnumTypeCast(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type StatusA string
type StatusB string

type A struct {
	Status StatusA
}

type B struct {
	Status StatusB
}
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-dir", dir)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}

	content := readFile(t, filepath.Join(dir, "b_mapper_gen.go"))
	if !strings.Contains(content, "StatusB(src.Status)") {
		t.Errorf("output missing StatusB cast; content:\n%s", content)
	}
}

func TestCLI_BidirectionalMapping(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type A struct {
	ID   int
	Name string
}

type B struct {
	ID   int
	Name string
}
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-bidi", "-dir", dir)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}

	content := readFile(t, filepath.Join(dir, "b_mapper_gen.go"))
	if !strings.Contains(content, "MapAToB") {
		t.Errorf("output missing MapAToB; content:\n%s", content)
	}
	if !strings.Contains(content, "MapBToA") {
		t.Errorf("output missing MapBToA; content:\n%s", content)
	}
}

func TestCLI_CustomOutput(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type A struct{ ID int }
type B struct{ ID int }
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-dir", dir, "-output", "custom_name.go")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}

	outPath := filepath.Join(dir, "custom_name.go")
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("expected custom_name.go to exist: %v", err)
	}
}

func TestCLI_CustomFuncName(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type A struct{ ID int }
type B struct{ ID int }
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-dir", dir, "-func", "CustomMapper")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr)
	}

	content := readFile(t, filepath.Join(dir, "b_mapper_gen.go"))
	if !strings.Contains(content, "CustomMapper") {
		t.Errorf("output missing CustomMapper function; content:\n%s", content)
	}
	// Default name must not appear.
	if strings.Contains(content, "MapAToB") {
		t.Errorf("output should not contain default MapAToB when -func is overridden; content:\n%s", content)
	}
}

// ---------------------------------------------------------------------------
// Negative integration tests
// ---------------------------------------------------------------------------

func TestCLI_MissingSrcFlag(t *testing.T) {
	_, stderr, code := runMapperGen(t, "-dst", "B")
	if code == 0 {
		t.Fatal("expected non-zero exit code when -src is missing")
	}
	if !strings.Contains(stderr, "both -src and -dst flags are required") {
		t.Errorf("unexpected stderr: %s", stderr)
	}
}

func TestCLI_MissingDstFlag(t *testing.T) {
	_, stderr, code := runMapperGen(t, "-src", "A")
	if code == 0 {
		t.Fatal("expected non-zero exit code when -dst is missing")
	}
	if !strings.Contains(stderr, "both -src and -dst flags are required") {
		t.Errorf("unexpected stderr: %s", stderr)
	}
}

func TestCLI_NonExistentSourceType(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type B struct{ ID int }
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "NonExist", "-dst", "B", "-dir", dir)
	if code == 0 {
		t.Fatal("expected non-zero exit for missing source type")
	}
	if !strings.Contains(stderr, "NonExist") {
		t.Errorf("expected error mentioning NonExist; stderr: %s", stderr)
	}
}

func TestCLI_NonExistentDestType(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type A struct{ ID int }
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "NonExist", "-dir", dir)
	if code == 0 {
		t.Fatal("expected non-zero exit for missing dest type")
	}
	if !strings.Contains(stderr, "NonExist") {
		t.Errorf("expected error mentioning NonExist; stderr: %s", stderr)
	}
}

func TestCLI_TypeMismatchNoConverter(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type A struct {
	Value string
}

type B struct {
	Value int64
}
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-dir", dir)
	if code == 0 {
		t.Fatal("expected non-zero exit for type mismatch without converter")
	}
	// Should mention the mismatch or missing converter.
	if !strings.Contains(stderr, "mismatch") && !strings.Contains(stderr, "converter") && !strings.Contains(stderr, "No converter") {
		t.Errorf("expected error about type mismatch or converter; stderr: %s", stderr)
	}
}

func TestCLI_NarrowingCastNoConverter(t *testing.T) {
	dir := createTempModule(t, "example.com/test", map[string]string{
		"models.go": `package test

type A struct {
	Value int64
}

type B struct {
	Value int32
}
`,
	})

	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-dir", dir)
	if code == 0 {
		t.Fatal("expected non-zero exit for narrowing conversion without converter")
	}
	if !strings.Contains(stderr, "narrowing") {
		t.Errorf("expected 'narrowing' in stderr; got: %s", stderr)
	}
}

func TestCLI_InvalidDir(t *testing.T) {
	_, stderr, code := runMapperGen(t, "-src", "A", "-dst", "B", "-dir", "/nonexistent/path/that/does/not/exist")
	if code == 0 {
		t.Fatal("expected non-zero exit for invalid directory")
	}
	// Should mention loading or package error.
	if stderr == "" {
		t.Error("expected non-empty stderr for invalid directory")
	}
}
