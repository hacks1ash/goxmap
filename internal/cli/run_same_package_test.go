package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hacks1ash/goxmap/internal/loader"
	"github.com/hacks1ash/goxmap/internal/matcher"
)

// writeTempModule creates a temporary directory with a go.mod and the given source files.
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

// ---------------------------------------------------------------------------
// RunSamePackage — happy paths (called in-process for coverage instrumentation)
// ---------------------------------------------------------------------------

func TestRunSamePackage_BasicMapping(t *testing.T) {
	dir := writeTempModule(t, "example.com/sp", map[string]string{
		"models.go": `package sp

type Src struct {
	ID   int
	Name string
}

type Dst struct {
	ID   int
	Name string
}
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir: dir,
		SrcType: "Src",
		DstType: "Dst",
		Output:  "dst_mapper_gen.go",
	})

	content := mustReadFile(t, filepath.Join(dir, "dst_mapper_gen.go"))
	if !strings.Contains(content, "MapSrcToDst") {
		t.Errorf("output missing MapSrcToDst:\n%s", content)
	}
	if !strings.Contains(content, "dst.ID = src.ID") {
		t.Errorf("output missing dst.ID = src.ID:\n%s", content)
	}
}

func TestRunSamePackage_CustomFuncName(t *testing.T) {
	dir := writeTempModule(t, "example.com/sp2", map[string]string{
		"models.go": `package sp2

type A struct{ Val int }
type B struct{ Val int }
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir:  dir,
		SrcType:  "A",
		DstType:  "B",
		FuncName: "ConvertAToB",
		Output:   "b_mapper_gen.go",
	})

	content := mustReadFile(t, filepath.Join(dir, "b_mapper_gen.go"))
	if !strings.Contains(content, "ConvertAToB") {
		t.Errorf("output missing ConvertAToB:\n%s", content)
	}
}

func TestRunSamePackage_BidirectionalMapping(t *testing.T) {
	dir := writeTempModule(t, "example.com/sp3", map[string]string{
		"models.go": `package sp3

type P struct{ X int }
type Q struct{ X int }
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir: dir,
		SrcType: "P",
		DstType: "Q",
		Output:  "q_mapper_gen.go",
		Bidi:    true,
	})

	content := mustReadFile(t, filepath.Join(dir, "q_mapper_gen.go"))
	if !strings.Contains(content, "MapPToQ") {
		t.Errorf("output missing MapPToQ:\n%s", content)
	}
	if !strings.Contains(content, "MapQToP") {
		t.Errorf("output missing MapQToP:\n%s", content)
	}
}

func TestRunSamePackage_IgnoreTag(t *testing.T) {
	dir := writeTempModule(t, "example.com/sp4", map[string]string{
		"models.go": `package sp4

type Src struct {
	ID     int
	Secret string
}

type Dst struct {
	ID     int
	Secret string ` + "`" + `mapper:"ignore"` + "`" + `
}
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir: dir,
		SrcType: "Src",
		DstType: "Dst",
		Output:  "dst_mapper_gen.go",
	})

	content := mustReadFile(t, filepath.Join(dir, "dst_mapper_gen.go"))
	if strings.Contains(content, "dst.Secret") {
		t.Errorf("output should not assign ignored field Secret:\n%s", content)
	}
}

func TestRunSamePackage_UnmatchedFieldWarning(t *testing.T) {
	// Dst has an extra non-optional field with no source counterpart — warning emitted.
	dir := writeTempModule(t, "example.com/sp5", map[string]string{
		"models.go": `package sp5

type Src struct {
	ID int
}

type Dst struct {
	ID    int
	Extra string
}
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir: dir,
		SrcType: "Src",
		DstType: "Dst",
		Output:  "dst_mapper_gen.go",
	})

	if _, err := os.Stat(filepath.Join(dir, "dst_mapper_gen.go")); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

func TestRunSamePackage_UnmatchedOptionalDstField(t *testing.T) {
	// Dst has an optional unmatched field — the `continue` branch in the
	// forward unmatched loop must be exercised and no warning emitted.
	dir := writeTempModule(t, "example.com/sp5opt", map[string]string{
		"models.go": `package sp5opt

type Src struct {
	ID int
}

type Dst struct {
	ID    int
	Extra string ` + "`" + `mapper:"optional"` + "`" + `
}
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir: dir,
		SrcType: "Src",
		DstType: "Dst",
		Output:  "dst_mapper_gen.go",
	})

	if _, err := os.Stat(filepath.Join(dir, "dst_mapper_gen.go")); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

func TestRunSamePackage_NumericWidening(t *testing.T) {
	dir := writeTempModule(t, "example.com/sp6", map[string]string{
		"models.go": `package sp6

type Src struct{ Count int32 }
type Dst struct{ Count int64 }
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir: dir,
		SrcType: "Src",
		DstType: "Dst",
		Output:  "dst_mapper_gen.go",
	})

	content := mustReadFile(t, filepath.Join(dir, "dst_mapper_gen.go"))
	if !strings.Contains(content, "int64(src.Count)") {
		t.Errorf("expected int64 cast in output:\n%s", content)
	}
}

func TestRunSamePackage_ConverterFunc(t *testing.T) {
	dir := writeTempModule(t, "example.com/sp7", map[string]string{
		"models.go": `package sp7

type MyInt int
type MyStr string

type Src struct{ Val MyInt }
type Dst struct{ Val MyStr }

func MapMyIntToMyStr(v MyInt) MyStr { return MyStr("x") }
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir: dir,
		SrcType: "Src",
		DstType: "Dst",
		Output:  "dst_mapper_gen.go",
	})

	content := mustReadFile(t, filepath.Join(dir, "dst_mapper_gen.go"))
	if !strings.Contains(content, "MapMyIntToMyStr") {
		t.Errorf("expected converter func call in output:\n%s", content)
	}
}

func TestRunSamePackage_MapperFnTag(t *testing.T) {
	dir := writeTempModule(t, "example.com/sp8", map[string]string{
		"models.go": `package sp8

type MyA string
type MyB int

func ToMyB(v MyA) MyB { return 0 }

type Src struct{ Val MyA }
type Dst struct {
	Val MyB ` + "`" + `mapper:"func:ToMyB"` + "`" + `
}
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir: dir,
		SrcType: "Src",
		DstType: "Dst",
		Output:  "dst_mapper_gen.go",
	})

	content := mustReadFile(t, filepath.Join(dir, "dst_mapper_gen.go"))
	if !strings.Contains(content, "ToMyB") {
		t.Errorf("expected ToMyB in output:\n%s", content)
	}
}

func TestRunSamePackage_BidiUnmatchedReverseField(t *testing.T) {
	// When -bidi is set and the reverse direction has an unmatched non-optional
	// field, a warning is printed but generation succeeds.
	dir := writeTempModule(t, "example.com/sp10", map[string]string{
		"models.go": `package sp10

type A struct {
	ID   int
	Name string
}

type B struct {
	ID    int
	Extra string
}
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir: dir,
		SrcType: "A",
		DstType: "B",
		Output:  "b_mapper_gen.go",
		Bidi:    true,
	})

	content := mustReadFile(t, filepath.Join(dir, "b_mapper_gen.go"))
	if !strings.Contains(content, "MapAToB") {
		t.Errorf("output missing MapAToB:\n%s", content)
	}
	if !strings.Contains(content, "MapBToA") {
		t.Errorf("output missing MapBToA:\n%s", content)
	}
}

func TestRunSamePackage_BidiOptionalReverseField(t *testing.T) {
	// In bidi mode A→B / B→A: A has an optional field not present in B.
	// The reverse pass (B→A) will see A.Extra as unmatched; the optional tag
	// must suppress the warning and hit the `continue` branch.
	dir := writeTempModule(t, "example.com/sp11", map[string]string{
		"models.go": `package sp11

type A struct {
	ID    int
	Extra string ` + "`" + `mapper:"optional"` + "`" + `
}

type B struct {
	ID int
}
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir: dir,
		SrcType: "A",
		DstType: "B",
		Output:  "b_mapper_gen.go",
		Bidi:    true,
	})

	if _, err := os.Stat(filepath.Join(dir, "b_mapper_gen.go")); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

func TestRunSamePackage_StructFuncOption(t *testing.T) {
	dir := writeTempModule(t, "example.com/sp9", map[string]string{
		"models.go": `package sp9

type Src struct{ ID int }
type Dst struct{ ID int }
`,
	})

	RunSamePackage(SamePackageOptions{
		WorkDir:    dir,
		SrcType:    "Src",
		DstType:    "Dst",
		Output:     "dst_mapper_gen.go",
		StructFunc: "CustomMap",
	})

	if _, err := os.Stat(filepath.Join(dir, "dst_mapper_gen.go")); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ResolveTypeMismatches — in-process tests on non-fatal paths
// ---------------------------------------------------------------------------

func TestResolveTypeMismatches_NoMismatch(t *testing.T) {
	// Pairs with TypeMismatch=false are skipped without any side effects.
	dir := writeTempModule(t, "example.com/rtm", map[string]string{
		"models.go": `package rtm

type A struct{ X int }
`,
	})

	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	pairs := []matcher.FieldPair{{}} // TypeMismatch defaults to false
	ResolveTypeMismatches(pctx, pairs, "A")
	// No panic, no change.
	if pairs[0].TypeMismatch {
		t.Error("TypeMismatch should remain false")
	}
}

func TestResolveTypeMismatches_ResolvedByMapperFnTag(t *testing.T) {
	dir := writeTempModule(t, "example.com/rtm2", map[string]string{
		"models.go": `package rtm2

type MyA string
type MyB int

func ToMyB(v MyA) MyB { return 0 }
`,
	})

	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	pair := matcher.FieldPair{
		TypeMismatch: true,
		Src:          loader.StructField{ElemType: "MyA"},
		Dst:          loader.StructField{Name: "Val", ElemType: "MyB", MapperFn: "ToMyB"},
	}
	pairs := []matcher.FieldPair{pair}

	ResolveTypeMismatches(pctx, pairs, "Src")

	if pairs[0].TypeMismatch {
		t.Error("expected TypeMismatch cleared by MapperFn tag")
	}
	if pairs[0].ConverterFunc != "ToMyB" {
		t.Errorf("expected ConverterFunc=ToMyB, got %q", pairs[0].ConverterFunc)
	}
}

func TestResolveTypeMismatches_ResolvedByAutoDiscovery(t *testing.T) {
	dir := writeTempModule(t, "example.com/rtm3", map[string]string{
		"models.go": `package rtm3

type MyInt int
type MyStr string

func MapMyIntToMyStr(v MyInt) MyStr { return "" }
`,
	})

	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	pair := matcher.FieldPair{
		TypeMismatch: true,
		Src:          loader.StructField{ElemType: "MyInt"},
		Dst:          loader.StructField{Name: "Val", ElemType: "MyStr"},
	}
	pairs := []matcher.FieldPair{pair}

	ResolveTypeMismatches(pctx, pairs, "Src")

	if pairs[0].TypeMismatch {
		t.Error("expected TypeMismatch cleared by auto-discovery")
	}
	if pairs[0].ConverterFunc != "MapMyIntToMyStr" {
		t.Errorf("expected ConverterFunc=MapMyIntToMyStr, got %q", pairs[0].ConverterFunc)
	}
}

// mustReadFile reads file content, failing the test on error.
func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}
