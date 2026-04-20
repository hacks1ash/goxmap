package main_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hacks1ash/goxmap/internal/generator"
	"github.com/hacks1ash/goxmap/internal/loader"
	"github.com/hacks1ash/goxmap/internal/matcher"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// projectRoot returns the absolute path to the project root directory.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Dir(filename)
}

// integrationDir returns the path to testdata/integration.
func integrationDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(projectRoot(t), "testdata", "integration")
}

// externalDir returns the path to testdata/external.
func externalDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(projectRoot(t), "testdata", "external")
}

// assertContains fails if haystack does not contain needle.
func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("output does not contain %q\n\nFull output:\n%s", needle, haystack)
	}
}

// assertNotContains fails if haystack contains needle.
func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("output should not contain %q\n\nFull output:\n%s", needle, haystack)
	}
}

// parseAndVerify writes code to a temp file, parses it with go/parser, and
// returns the AST file. It fails the test if parsing fails.
func parseAndVerify(t *testing.T, code []byte) *ast.File {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "generated.go")
	if err := os.WriteFile(path, code, 0644); err != nil {
		t.Fatalf("writing generated file: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("generated code does not parse as valid Go:\n%v\n\nGenerated code:\n%s", err, string(code))
	}
	return f
}

// funcNames returns a set of top-level function declarations in the AST.
func funcNames(f *ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
			names[fn.Name.Name] = true
		}
	}
	return names
}

// ---------------------------------------------------------------------------
// Suite 1: Basic Generator Execution with Pointer Scenarios
// ---------------------------------------------------------------------------

func TestSuite1_PointerConversions(t *testing.T) {
	dir := integrationDir(t)
	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	src, err := loader.LoadStructFromPkg(pctx, "PtrSource")
	if err != nil {
		t.Fatalf("load PtrSource: %v", err)
	}

	dst, err := loader.LoadStructFromPkg(pctx, "PtrDest")
	if err != nil {
		t.Fatalf("load PtrDest: %v", err)
	}

	result := matcher.Match(src, dst, matcher.MatchOptions{})

	mcfg := generator.MultiConfig{
		PackageName: "integration",
		RootFunc: generator.Config{
			PackageName: "integration",
			FuncName:    "MapPtrSourceToPtrDest",
			SrcType:     "PtrSource",
			DstType:     "PtrDest",
			Pairs:       result.Pairs,
		},
		ExistingMappers: make(map[string]bool),
		PkgContext:      pctx,
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		t.Fatalf("GenerateMulti: %v", err)
	}

	output := string(code)

	t.Run("parses_as_valid_go", func(t *testing.T) {
		f := parseAndVerify(t, code)
		fns := funcNames(f)
		if !fns["MapPtrSourceToPtrDest"] {
			t.Error("expected MapPtrSourceToPtrDest function declaration in AST")
		}
	})

	t.Run("ptr_string_to_string_nil_check", func(t *testing.T) {
		// *string -> string must have a nil check to avoid nil pointer dereference.
		assertContains(t, output, "src.Name != nil")
		assertContains(t, output, "*src.Name")
	})

	t.Run("int_to_ptr_int_address_of", func(t *testing.T) {
		// int -> *int must take the address.
		assertContains(t, output, "v := src.Age")
		assertContains(t, output, "return &v")
	})

	t.Run("ptr_string_to_ptr_string_direct", func(t *testing.T) {
		// *string -> *string should be a direct assignment.
		assertContains(t, output, "dst.Email = src.Email")
	})

	t.Run("bool_to_bool_direct", func(t *testing.T) {
		// bool -> bool is a simple direct assignment.
		assertContains(t, output, "dst.Active = src.Active")
	})
}

func TestSuite1_CustomMapperFunction(t *testing.T) {
	dir := integrationDir(t)
	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	src, err := loader.LoadStructFromPkg(pctx, "CustomFuncSource")
	if err != nil {
		t.Fatalf("load CustomFuncSource: %v", err)
	}

	dst, err := loader.LoadStructFromPkg(pctx, "CustomFuncDest")
	if err != nil {
		t.Fatalf("load CustomFuncDest: %v", err)
	}

	result := matcher.Match(src, dst, matcher.MatchOptions{})

	mcfg := generator.MultiConfig{
		PackageName: "integration",
		RootFunc: generator.Config{
			PackageName: "integration",
			FuncName:    "MapCustomFuncSourceToCustomFuncDest",
			SrcType:     "CustomFuncSource",
			DstType:     "CustomFuncDest",
			Pairs:       result.Pairs,
		},
		ExistingMappers: make(map[string]bool),
		PkgContext:      pctx,
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		t.Fatalf("GenerateMulti: %v", err)
	}

	output := string(code)

	t.Run("parses_as_valid_go", func(t *testing.T) {
		parseAndVerify(t, code)
	})

	t.Run("references_custom_function", func(t *testing.T) {
		assertContains(t, output, "ParseTimestamp(src.CreatedAt)")
	})

	t.Run("direct_assignment_for_id", func(t *testing.T) {
		assertContains(t, output, "dst.ID = src.ID")
	})
}

// ---------------------------------------------------------------------------
// Suite 2: Recursive & Slice Mapping
// ---------------------------------------------------------------------------

func TestSuite2_DeeplyNestedMapping(t *testing.T) {
	dir := integrationDir(t)
	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	src, err := loader.LoadStructFromPkg(pctx, "Company")
	if err != nil {
		t.Fatalf("load Company: %v", err)
	}

	dst, err := loader.LoadStructFromPkg(pctx, "CompanyDTO")
	if err != nil {
		t.Fatalf("load CompanyDTO: %v", err)
	}

	result := matcher.Match(src, dst, matcher.MatchOptions{})

	mcfg := generator.MultiConfig{
		PackageName: "integration",
		RootFunc: generator.Config{
			PackageName: "integration",
			FuncName:    "MapCompanyToCompanyDTO",
			SrcType:     "Company",
			DstType:     "CompanyDTO",
			Pairs:       result.Pairs,
		},
		ExistingMappers: make(map[string]bool),
		PkgContext:      pctx,
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		t.Fatalf("GenerateMulti: %v", err)
	}

	output := string(code)

	t.Run("parses_as_valid_go", func(t *testing.T) {
		f := parseAndVerify(t, code)
		fns := funcNames(f)

		// All four mapper functions should be generated.
		for _, name := range []string{
			"MapCompanyToCompanyDTO",
			"MapDepartmentToDepartmentDTO",
			"MapEmployeeToEmployeeDTO",
			"MapAddressToAddressDTO",
		} {
			if !fns[name] {
				t.Errorf("expected function %s in generated code", name)
			}
		}
	})

	t.Run("root_mapper_calls_sub_mappers", func(t *testing.T) {
		assertContains(t, output, "MapDepartmentToDepartmentDTO")
	})

	t.Run("department_mapper_maps_employee_slice", func(t *testing.T) {
		assertContains(t, output, "for i, v := range src.Employees")
		assertContains(t, output, "MapEmployeeToEmployeeDTO(&item)")
	})

	t.Run("employee_mapper_calls_address_mapper", func(t *testing.T) {
		assertContains(t, output, "MapAddressToAddressDTO(&src.Address)")
	})

	t.Run("nil_slice_check", func(t *testing.T) {
		// Slice mappings should have nil checks before iterating.
		assertContains(t, output, "if src.Departments != nil")
		assertContains(t, output, "if src.Employees != nil")
	})
}

func TestSuite2_SliceOfPointersToValues(t *testing.T) {
	dir := integrationDir(t)
	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	src, err := loader.LoadStructFromPkg(pctx, "TeamWithPtrSlice")
	if err != nil {
		t.Fatalf("load TeamWithPtrSlice: %v", err)
	}

	dst, err := loader.LoadStructFromPkg(pctx, "TeamWithValSlice")
	if err != nil {
		t.Fatalf("load TeamWithValSlice: %v", err)
	}

	result := matcher.Match(src, dst, matcher.MatchOptions{})

	mcfg := generator.MultiConfig{
		PackageName: "integration",
		RootFunc: generator.Config{
			PackageName: "integration",
			FuncName:    "MapTeamWithPtrSliceToTeamWithValSlice",
			SrcType:     "TeamWithPtrSlice",
			DstType:     "TeamWithValSlice",
			Pairs:       result.Pairs,
		},
		ExistingMappers: make(map[string]bool),
		PkgContext:      pctx,
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		t.Fatalf("GenerateMulti: %v", err)
	}

	output := string(code)

	t.Run("parses_as_valid_go", func(t *testing.T) {
		parseAndVerify(t, code)
	})

	t.Run("ptr_element_nil_check_in_loop", func(t *testing.T) {
		// []*UserRef -> []UserRefDTO should check v != nil before dereferencing.
		assertContains(t, output, "if v != nil")
		assertContains(t, output, "MapUserRefToUserRefDTO(v)")
	})

	t.Run("generates_element_mapper", func(t *testing.T) {
		assertContains(t, output, "func MapUserRefToUserRefDTO(src *UserRef) *UserRefDTO")
	})
}

func TestSuite2_ExistingMapperReuse(t *testing.T) {
	dir := integrationDir(t)
	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	src, err := loader.LoadStructFromPkg(pctx, "Employee")
	if err != nil {
		t.Fatalf("load Employee: %v", err)
	}

	dst, err := loader.LoadStructFromPkg(pctx, "EmployeeDTO")
	if err != nil {
		t.Fatalf("load EmployeeDTO: %v", err)
	}

	result := matcher.Match(src, dst, matcher.MatchOptions{})

	// Simulate that MapAddressToAddressDTO already exists (discovered via DiscoverMapperFuncs).
	existing := loader.DiscoverMapperFuncs(pctx)
	if !existing["MapAddressToAddressDTO"] {
		t.Fatal("expected MapAddressToAddressDTO to be discovered in integration testdata")
	}

	mcfg := generator.MultiConfig{
		PackageName: "integration",
		RootFunc: generator.Config{
			PackageName: "integration",
			FuncName:    "MapEmployeeToEmployeeDTO",
			SrcType:     "Employee",
			DstType:     "EmployeeDTO",
			Pairs:       result.Pairs,
		},
		ExistingMappers: existing,
		PkgContext:      pctx,
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		t.Fatalf("GenerateMulti: %v", err)
	}

	output := string(code)

	t.Run("calls_existing_mapper", func(t *testing.T) {
		// The generated code should still call MapAddressToAddressDTO.
		assertContains(t, output, "MapAddressToAddressDTO")
	})

	t.Run("does_not_regenerate_existing_mapper", func(t *testing.T) {
		// Should NOT generate a new definition of MapAddressToAddressDTO.
		assertNotContains(t, output, "func MapAddressToAddressDTO")
	})
}

func TestSuite2_CircularDependency(t *testing.T) {
	dir := integrationDir(t)
	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	src, err := loader.LoadStructFromPkg(pctx, "Node")
	if err != nil {
		t.Fatalf("load Node: %v", err)
	}

	dst, err := loader.LoadStructFromPkg(pctx, "NodeDTO")
	if err != nil {
		t.Fatalf("load NodeDTO: %v", err)
	}

	result := matcher.Match(src, dst, matcher.MatchOptions{})

	mcfg := generator.MultiConfig{
		PackageName: "integration",
		RootFunc: generator.Config{
			PackageName: "integration",
			FuncName:    "MapNodeToNodeDTO",
			SrcType:     "Node",
			DstType:     "NodeDTO",
			Pairs:       result.Pairs,
		},
		ExistingMappers: make(map[string]bool),
		PkgContext:      pctx,
	}

	// This must NOT hang or panic due to circular Node -> Node reference.
	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		t.Fatalf("GenerateMulti: %v", err)
	}

	output := string(code)

	t.Run("parses_as_valid_go", func(t *testing.T) {
		parseAndVerify(t, code)
	})

	t.Run("generates_mapper_function", func(t *testing.T) {
		assertContains(t, output, "func MapNodeToNodeDTO(src *Node) *NodeDTO")
	})

	t.Run("handles_self_reference_in_children_slice", func(t *testing.T) {
		// Children []Node -> []NodeDTO should call itself recursively.
		assertContains(t, output, "MapNodeToNodeDTO")
	})

	t.Run("handles_ptr_self_reference", func(t *testing.T) {
		// Parent *Node -> *NodeDTO: pointer passed directly, nil handled by nil guard.
		assertContains(t, output, "MapNodeToNodeDTO(src.Parent)")
	})
}

func TestSuite2_GoldStandardComparison(t *testing.T) {
	dir := integrationDir(t)
	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	src, err := loader.LoadStructFromPkg(pctx, "Company")
	if err != nil {
		t.Fatalf("load Company: %v", err)
	}

	dst, err := loader.LoadStructFromPkg(pctx, "CompanyDTO")
	if err != nil {
		t.Fatalf("load CompanyDTO: %v", err)
	}

	result := matcher.Match(src, dst, matcher.MatchOptions{})

	mcfg := generator.MultiConfig{
		PackageName: "integration",
		RootFunc: generator.Config{
			PackageName: "integration",
			FuncName:    "MapCompanyToCompanyDTO",
			SrcType:     "Company",
			DstType:     "CompanyDTO",
			Pairs:       result.Pairs,
		},
		ExistingMappers: make(map[string]bool),
		PkgContext:      pctx,
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		t.Fatalf("GenerateMulti: %v", err)
	}

	output := string(code)

	// Gold standard patterns - every generated file must contain these patterns.
	goldPatterns := []struct {
		name    string
		pattern string
	}{
		{"header", "// Code generated by goxmap; DO NOT EDIT."},
		{"package", "package integration"},
		{"root_func_sig", "func MapCompanyToCompanyDTO(src *Company) *CompanyDTO"},
		{"department_func_sig", "func MapDepartmentToDepartmentDTO(src *Department) *DepartmentDTO"},
		{"employee_func_sig", "func MapEmployeeToEmployeeDTO(src *Employee) *EmployeeDTO"},
		{"address_func_sig", "func MapAddressToAddressDTO(src *Address) *AddressDTO"},
		{"var_dst_pattern", "dst := &CompanyDTO{}"},
		{"return_dst", "return dst"},
		{"direct_name_assign", "dst.Name = src.Name"},
		{"slice_nil_guard", "if src.Departments != nil"},
		{"slice_make", "make([]DepartmentDTO, len(src.Departments))"},
		{"slice_iteration", "for i, v := range src.Departments"},
		{"nested_street", "dst.Street = src.Street"},
		{"nested_city", "dst.City = src.City"},
	}

	for _, gp := range goldPatterns {
		t.Run("gold_"+gp.name, func(t *testing.T) {
			assertContains(t, output, gp.pattern)
		})
	}
}

// ---------------------------------------------------------------------------
// Suite 3: Cross-Package / External Dependency Mapping
// ---------------------------------------------------------------------------

func TestSuite3_ProtobufStyleMapping(t *testing.T) {
	// Load the external package to get struct info and getters.
	extDir := externalDir(t)
	extPctx, err := loader.LoadPackage(extDir)
	if err != nil {
		t.Fatalf("LoadPackage(external): %v", err)
	}

	extInfo, err := loader.LoadStructFromPkg(extPctx, "ExternalUser")
	if err != nil {
		t.Fatalf("load ExternalUser: %v", err)
	}

	getters := loader.DiscoverGetters(extPctx, "ExternalUser")

	// Build the internal struct info manually to simulate bind tags.
	internal := &loader.StructInfo{
		PackageName: "myapp",
		Name:        "User",
		Fields: []loader.StructField{
			{Name: "Name", TypeStr: "string", ElemType: "string", BindName: "FullName"},
			{Name: "Email", TypeStr: "string", ElemType: "string", BindName: "UserEmail"},
			{Name: "UserAge", TypeStr: "int", ElemType: "int"},
		},
	}

	crossResult := matcher.MatchCross(internal, extInfo, getters)

	t.Run("all_fields_matched", func(t *testing.T) {
		if got := len(crossResult.ToExternal.Pairs); got != 3 {
			t.Errorf("ToExternal: got %d pairs, want 3", got)
		}
		if got := len(crossResult.ToExternal.Unmatched); got != 0 {
			t.Errorf("ToExternal: got %d unmatched, want 0", got)
		}
	})

	ccfg := generator.CrossConfig{
		PackageName:          "myapp",
		InternalType:         "User",
		ExternalType:         "ExternalUser",
		ExternalPkgName:      "external",
		ExternalPkgPath:      "github.com/hacks1ash/goxmap/testdata/external",
		ToExternalFuncName:   "MapUserToExternalUser",
		FromExternalFuncName: "MapExternalUserToUser",
		ToExternalPairs:      crossResult.ToExternal.Pairs,
		FromExternalPairs:    crossResult.FromExternal.Pairs,
		Bidirectional:        true,
	}

	code, err := generator.GenerateCross(ccfg)
	if err != nil {
		t.Fatalf("GenerateCross: %v", err)
	}

	output := string(code)

	t.Run("parses_as_valid_go", func(t *testing.T) {
		parseAndVerify(t, code)
	})

	t.Run("to_external_assigns_to_fields", func(t *testing.T) {
		assertContains(t, output, "func MapUserToExternalUser(src *User) *external.ExternalUser")
		assertContains(t, output, "dst.FullName = src.Name")
		assertContains(t, output, "dst.UserEmail = src.Email")
	})

	t.Run("from_external_uses_getters", func(t *testing.T) {
		assertContains(t, output, "func MapExternalUserToUser(src *external.ExternalUser) *User")
		assertContains(t, output, "src.GetFullName()")
		assertContains(t, output, "src.GetUserEmail()")
	})

	t.Run("import_statement", func(t *testing.T) {
		assertContains(t, output, `"github.com/hacks1ash/goxmap/testdata/external"`)
	})
}

func TestSuite3_JSONKeyAlignment(t *testing.T) {
	// Load the external package.
	extDir := externalDir(t)
	extPctx, err := loader.LoadPackage(extDir)
	if err != nil {
		t.Fatalf("LoadPackage(external): %v", err)
	}

	extInfo, err := loader.LoadStructFromPkg(extPctx, "RemoteRecord")
	if err != nil {
		t.Fatalf("load RemoteRecord: %v", err)
	}

	// Build internal struct with bind_json tags matching the external json keys.
	internal := &loader.StructInfo{
		PackageName: "myapp",
		Name:        "LocalRecord",
		Fields: []loader.StructField{
			{Name: "ID", TypeStr: "string", ElemType: "string", BindJSON: "remote_id_key"},
			{Name: "Data", TypeStr: "string", ElemType: "string", BindJSON: "remote_data_key"},
			{Name: "Status", TypeStr: "int", ElemType: "int"},
		},
	}

	crossResult := matcher.MatchCross(internal, extInfo, nil)

	t.Run("bind_json_matches_all_fields", func(t *testing.T) {
		if got := len(crossResult.ToExternal.Pairs); got != 3 {
			t.Errorf("ToExternal: got %d pairs, want 3", got)
		}
	})

	t.Run("bind_json_links_correctly", func(t *testing.T) {
		for _, p := range crossResult.ToExternal.Pairs {
			switch p.Src.Name {
			case "ID":
				if p.Dst.Name != "RemoteID" {
					t.Errorf("ID should map to RemoteID via bind_json, got %s", p.Dst.Name)
				}
			case "Data":
				if p.Dst.Name != "RemoteData" {
					t.Errorf("Data should map to RemoteData via bind_json, got %s", p.Dst.Name)
				}
			case "Status":
				if p.Dst.Name != "Status" {
					t.Errorf("Status should map to Status via field name, got %s", p.Dst.Name)
				}
			}
		}
	})

	ccfg := generator.CrossConfig{
		PackageName:        "myapp",
		InternalType:       "LocalRecord",
		ExternalType:       "RemoteRecord",
		ExternalPkgName:    "external",
		ExternalPkgPath:    "github.com/hacks1ash/goxmap/testdata/external",
		ToExternalFuncName: "MapLocalRecordToRemoteRecord",
		ToExternalPairs:    crossResult.ToExternal.Pairs,
		Bidirectional:      false,
	}

	code, err := generator.GenerateCross(ccfg)
	if err != nil {
		t.Fatalf("GenerateCross: %v", err)
	}

	output := string(code)

	t.Run("parses_as_valid_go", func(t *testing.T) {
		parseAndVerify(t, code)
	})

	t.Run("assigns_to_correct_fields", func(t *testing.T) {
		assertContains(t, output, "dst.RemoteID = src.ID")
		assertContains(t, output, "dst.RemoteData = src.Data")
		assertContains(t, output, "dst.Status = src.Status")
	})
}

func TestSuite3_ComplexCollections(t *testing.T) {
	// Test mapping []InternalRole to []*ExternalRole.
	// We simulate this with manually constructed StructInfos since the slice
	// element pointer direction conversion is the key behavior under test.
	internal := &loader.StructInfo{
		PackageName: "myapp",
		Name:        "InternalContainer",
		Fields: []loader.StructField{
			{
				Name:              "Roles",
				TypeStr:           "[]InternalRole",
				ElemType:          "[]InternalRole",
				IsSlice:           true,
				SliceElemType:     "InternalRole",
				SliceElemIsPtr:    false,
				SliceElemTypeName: "InternalRole",
				IsSliceElemStruct: true,
			},
		},
	}

	external := &loader.StructInfo{
		PackageName: "myapp",
		Name:        "ExternalContainer",
		Fields: []loader.StructField{
			{
				Name:              "Roles",
				TypeStr:           "[]*ExternalRole",
				ElemType:          "[]*ExternalRole",
				IsSlice:           true,
				SliceElemType:     "ExternalRole",
				SliceElemIsPtr:    true,
				SliceElemTypeName: "ExternalRole",
				IsSliceElemStruct: true,
			},
		},
	}

	result := matcher.Match(internal, external, matcher.MatchOptions{})

	if len(result.Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(result.Pairs))
	}

	pair := result.Pairs[0]

	t.Run("detects_slice_mapper_needed", func(t *testing.T) {
		if !pair.NeedsSliceMapper() {
			t.Error("expected NeedsSliceMapper() to be true")
		}
	})

	// Generate code to verify the element pointer conversion in slices.
	mcfg := generator.MultiConfig{
		PackageName: "myapp",
		RootFunc: generator.Config{
			PackageName: "myapp",
			FuncName:    "MapInternalContainerToExternalContainer",
			SrcType:     "InternalContainer",
			DstType:     "ExternalContainer",
			Pairs:       result.Pairs,
		},
		ExistingMappers: make(map[string]bool),
		// No PkgContext here since these are manually constructed types.
		// The generator will skip recursive discovery but still generate the root.
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		t.Fatalf("GenerateMulti: %v", err)
	}

	output := string(code)

	t.Run("generates_ptr_element_conversion", func(t *testing.T) {
		// []T -> []*T: take address of v, pass to pointer-based mapper.
		assertContains(t, output, "item := v")
		assertContains(t, output, "MapInternalRoleToExternalRole(&item)")
	})

	t.Run("generates_make_with_ptr_elem", func(t *testing.T) {
		assertContains(t, output, "make([]*ExternalRole, len(src.Roles))")
	})
}

func TestSuite3_UnmatchedBindField(t *testing.T) {
	internal := &loader.StructInfo{
		PackageName: "myapp",
		Name:        "BadBind",
		Fields: []loader.StructField{
			{Name: "ID", TypeStr: "int", ElemType: "int"},
			{Name: "Ghost", TypeStr: "string", ElemType: "string", BindName: "NonExistentField"},
		},
	}

	external := &loader.StructInfo{
		PackageName: "ext",
		Name:        "ExtStruct",
		Fields: []loader.StructField{
			{Name: "ID", TypeStr: "int", ElemType: "int"},
		},
	}

	result := matcher.MatchCross(internal, external, nil)

	t.Run("matched_count", func(t *testing.T) {
		if got := len(result.ToExternal.Pairs); got != 1 {
			t.Errorf("expected 1 matched pair, got %d", got)
		}
	})

	t.Run("unmatched_bind_field", func(t *testing.T) {
		if got := len(result.ToExternal.Unmatched); got != 1 {
			t.Fatalf("expected 1 unmatched field, got %d", got)
		}
		if result.ToExternal.Unmatched[0].Name != "Ghost" {
			t.Errorf("expected Ghost to be unmatched, got %s", result.ToExternal.Unmatched[0].Name)
		}
	})

	t.Run("unmatched_in_both_directions", func(t *testing.T) {
		if got := len(result.FromExternal.Unmatched); got != 1 {
			t.Fatalf("expected 1 unmatched in FromExternal, got %d", got)
		}
		if result.FromExternal.Unmatched[0].Name != "Ghost" {
			t.Errorf("expected Ghost in FromExternal.Unmatched, got %s", result.FromExternal.Unmatched[0].Name)
		}
	})
}

// ---------------------------------------------------------------------------
// Suite 4: End-to-End Integration
// ---------------------------------------------------------------------------

func TestSuite4_EndToEnd_TempModule(t *testing.T) {
	// Create a temporary directory with a go.mod and Go source files.
	tmpDir := t.TempDir()
	modName := "example.com/e2etest"

	// Write go.mod.
	gomod := "module " + modName + "\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	// Write source Go file with structs.
	src := `package e2etest

type Order struct {
	ID         int       ` + "`json:\"id\"`" + `
	CustomerID string    ` + "`json:\"customer_id\"`" + `
	Amount     float64   ` + "`json:\"amount\"`" + `
	Notes      *string   ` + "`json:\"notes\"`" + `
	Items      []Item    ` + "`json:\"items\"`" + `
}

type Item struct {
	SKU   string  ` + "`json:\"sku\"`" + `
	Qty   int     ` + "`json:\"qty\"`" + `
	Price float64 ` + "`json:\"price\"`" + `
}

type OrderDTO struct {
	ID         int       ` + "`json:\"id\"`" + `
	CustomerID string    ` + "`json:\"customer_id\"`" + `
	Amount     *float64  ` + "`json:\"amount\"`" + `
	Notes      string    ` + "`json:\"notes\"`" + `
	Items      []ItemDTO ` + "`json:\"items\"`" + `
}

type ItemDTO struct {
	SKU   string  ` + "`json:\"sku\"`" + `
	Qty   int     ` + "`json:\"qty\"`" + `
	Price float64 ` + "`json:\"price\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "models.go"), []byte(src), 0644); err != nil {
		t.Fatalf("writing models.go: %v", err)
	}

	// Step 1: Load the package.
	pctx, err := loader.LoadPackage(tmpDir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	// Step 2: Load structs from the package.
	orderInfo, err := loader.LoadStructFromPkg(pctx, "Order")
	if err != nil {
		t.Fatalf("LoadStructFromPkg(Order): %v", err)
	}

	orderDTOInfo, err := loader.LoadStructFromPkg(pctx, "OrderDTO")
	if err != nil {
		t.Fatalf("LoadStructFromPkg(OrderDTO): %v", err)
	}

	// Verify basic struct loading.
	if orderInfo.PackageName != "e2etest" {
		t.Errorf("Order package: got %q, want %q", orderInfo.PackageName, "e2etest")
	}

	// Step 3: Match fields.
	matchResult := matcher.Match(orderInfo, orderDTOInfo, matcher.MatchOptions{})

	if got := len(matchResult.Pairs); got != 5 {
		t.Fatalf("expected 5 matched pairs, got %d", got)
	}
	if got := len(matchResult.Unmatched); got != 0 {
		t.Errorf("expected 0 unmatched fields, got %d", got)
	}

	// Step 4: Generate code.
	mcfg := generator.MultiConfig{
		PackageName: "e2etest",
		RootFunc: generator.Config{
			PackageName: "e2etest",
			FuncName:    "MapOrderToOrderDTO",
			SrcType:     "Order",
			DstType:     "OrderDTO",
			Pairs:       matchResult.Pairs,
		},
		ExistingMappers: make(map[string]bool),
		PkgContext:      pctx,
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		t.Fatalf("GenerateMulti: %v", err)
	}

	output := string(code)

	// Step 5: Write the generated file and parse it.
	genPath := filepath.Join(tmpDir, "mapper_gen.go")
	if err := os.WriteFile(genPath, code, 0644); err != nil {
		t.Fatalf("writing generated file: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, genPath, nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("generated code does not parse:\n%v\n\nCode:\n%s", err, output)
	}

	// Step 6: Verify AST contains expected function declarations.
	fns := funcNames(f)

	expectedFuncs := []string{
		"MapOrderToOrderDTO",
		"MapItemToItemDTO",
	}

	for _, name := range expectedFuncs {
		t.Run("has_func_"+name, func(t *testing.T) {
			if !fns[name] {
				t.Errorf("expected function %s in generated AST", name)
			}
		})
	}

	// Step 7: Verify specific code patterns.
	t.Run("ptr_deref_notes", func(t *testing.T) {
		// *string -> string: nil check required.
		assertContains(t, output, "src.Notes != nil")
	})

	t.Run("addr_amount", func(t *testing.T) {
		// float64 -> *float64: address-of required.
		assertContains(t, output, "v := src.Amount")
		assertContains(t, output, "return &v")
	})

	t.Run("slice_mapping_items", func(t *testing.T) {
		assertContains(t, output, "for i, v := range src.Items")
		assertContains(t, output, "MapItemToItemDTO(&item)")
	})

	t.Run("slice_nil_check", func(t *testing.T) {
		assertContains(t, output, "if src.Items != nil")
	})

	t.Run("item_mapper_fields", func(t *testing.T) {
		assertContains(t, output, "dst.SKU = src.SKU")
		assertContains(t, output, "dst.Qty = src.Qty")
		assertContains(t, output, "dst.Price = src.Price")
	})
}

func TestSuite4_EndToEnd_CrossPackage(t *testing.T) {
	// Create a temporary module with two packages: internal and external.
	tmpDir := t.TempDir()
	modName := "example.com/crosstest"

	gomod := "module " + modName + "\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	// Create the "ext" sub-package.
	extPkgDir := filepath.Join(tmpDir, "ext")
	if err := os.MkdirAll(extPkgDir, 0755); err != nil {
		t.Fatalf("creating ext dir: %v", err)
	}

	extSrc := `package ext

type RemoteUser struct {
	RemoteName string ` + "`json:\"remote_name\"`" + `
	RemoteAge  int    ` + "`json:\"remote_age\"`" + `
}

func (r *RemoteUser) GetRemoteName() string {
	if r != nil {
		return r.RemoteName
	}
	return ""
}

func (r *RemoteUser) GetRemoteAge() int {
	if r != nil {
		return r.RemoteAge
	}
	return 0
}
`
	if err := os.WriteFile(filepath.Join(extPkgDir, "remote.go"), []byte(extSrc), 0644); err != nil {
		t.Fatalf("writing ext/remote.go: %v", err)
	}

	// Create the "app" sub-package.
	appPkgDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appPkgDir, 0755); err != nil {
		t.Fatalf("creating app dir: %v", err)
	}

	appSrc := `package app

type LocalUser struct {
	Name string ` + "`json:\"name\" mapper:\"bind:RemoteName\"`" + `
	Age  int    ` + "`json:\"age\" mapper:\"bind:RemoteAge\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(appPkgDir, "local.go"), []byte(appSrc), 0644); err != nil {
		t.Fatalf("writing app/local.go: %v", err)
	}

	// Load both packages.
	appPctx, err := loader.LoadPackage(appPkgDir)
	if err != nil {
		t.Fatalf("LoadPackage(app): %v", err)
	}

	extPctx, err := loader.LoadExternalPackage(tmpDir, modName+"/ext")
	if err != nil {
		t.Fatalf("LoadExternalPackage(ext): %v", err)
	}

	localUser, err := loader.LoadStructFromPkg(appPctx, "LocalUser")
	if err != nil {
		t.Fatalf("load LocalUser: %v", err)
	}

	remoteUser, err := loader.LoadStructFromPkg(extPctx, "RemoteUser")
	if err != nil {
		t.Fatalf("load RemoteUser: %v", err)
	}

	getters := loader.DiscoverGetters(extPctx, "RemoteUser")

	// Verify getters were discovered.
	t.Run("discovers_getters", func(t *testing.T) {
		if _, ok := getters["RemoteName"]; !ok {
			t.Error("expected getter for RemoteName")
		}
		if _, ok := getters["RemoteAge"]; !ok {
			t.Error("expected getter for RemoteAge")
		}
	})

	// Match cross-package.
	crossResult := matcher.MatchCross(localUser, remoteUser, getters)

	t.Run("cross_match_pairs", func(t *testing.T) {
		if got := len(crossResult.ToExternal.Pairs); got != 2 {
			t.Errorf("ToExternal: got %d pairs, want 2", got)
		}
		if got := len(crossResult.FromExternal.Pairs); got != 2 {
			t.Errorf("FromExternal: got %d pairs, want 2", got)
		}
	})

	// Generate cross-package code.
	ccfg := generator.CrossConfig{
		PackageName:          "app",
		InternalType:         "LocalUser",
		ExternalType:         "RemoteUser",
		ExternalPkgName:      "ext",
		ExternalPkgPath:      modName + "/ext",
		ToExternalFuncName:   "MapLocalUserToRemoteUser",
		FromExternalFuncName: "MapRemoteUserToLocalUser",
		ToExternalPairs:      crossResult.ToExternal.Pairs,
		FromExternalPairs:    crossResult.FromExternal.Pairs,
		Bidirectional:        true,
	}

	code, err := generator.GenerateCross(ccfg)
	if err != nil {
		t.Fatalf("GenerateCross: %v", err)
	}

	output := string(code)

	// Write and parse.
	genPath := filepath.Join(appPkgDir, "mapper_gen.go")
	if err := os.WriteFile(genPath, code, 0644); err != nil {
		t.Fatalf("writing generated file: %v", err)
	}

	t.Run("parses_as_valid_go", func(t *testing.T) {
		f := parseAndVerify(t, code)
		fns := funcNames(f)
		if !fns["MapLocalUserToRemoteUser"] {
			t.Error("expected MapLocalUserToRemoteUser")
		}
		if !fns["MapRemoteUserToLocalUser"] {
			t.Error("expected MapRemoteUserToLocalUser")
		}
	})

	t.Run("to_external_field_access", func(t *testing.T) {
		assertContains(t, output, "dst.RemoteName = src.Name")
		assertContains(t, output, "dst.RemoteAge = src.Age")
	})

	t.Run("from_external_uses_getters", func(t *testing.T) {
		assertContains(t, output, "src.GetRemoteName()")
		assertContains(t, output, "src.GetRemoteAge()")
	})

	t.Run("correct_import_path", func(t *testing.T) {
		assertContains(t, output, `"example.com/crosstest/ext"`)
	})

	t.Run("bidirectional_signatures", func(t *testing.T) {
		assertContains(t, output, "func MapLocalUserToRemoteUser(src *LocalUser) *ext.RemoteUser")
		assertContains(t, output, "func MapRemoteUserToLocalUser(src *ext.RemoteUser) *LocalUser")
	})
}

// TestSuite4_EndToEnd_CrossPackage_ProtoOptionalFields verifies that proto
// optional fields (pointer field + value getter) generate correct assignments.
func TestSuite4_EndToEnd_CrossPackage_ProtoOptionalFields(t *testing.T) {
	tmpDir := t.TempDir()
	modName := "example.com/prototest"

	gomod := "module " + modName + "\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	// Proto-style package with optional (pointer) fields and value-returning getters.
	extPkgDir := filepath.Join(tmpDir, "pb")
	if err := os.MkdirAll(extPkgDir, 0755); err != nil {
		t.Fatalf("creating pb dir: %v", err)
	}

	pbSrc := `package pb

type ProtoMessage struct {
	Name    *string ` + "`json:\"name\"`" + `
	Country *string ` + "`json:\"country\"`" + `
	Age     *int    ` + "`json:\"age\"`" + `
}

func (x *ProtoMessage) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *ProtoMessage) GetCountry() string {
	if x != nil && x.Country != nil {
		return *x.Country
	}
	return ""
}

func (x *ProtoMessage) GetAge() int {
	if x != nil && x.Age != nil {
		return *x.Age
	}
	return 0
}
`
	if err := os.WriteFile(filepath.Join(extPkgDir, "message.go"), []byte(pbSrc), 0644); err != nil {
		t.Fatalf("writing pb/message.go: %v", err)
	}

	appPkgDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appPkgDir, 0755); err != nil {
		t.Fatalf("creating app dir: %v", err)
	}

	// Local struct with value fields (common case: proto *T -> local T).
	appSrc := `package app

type LocalModel struct {
	Name    string  ` + "`json:\"name\" mapper:\"bind:Name\"`" + `
	Country *string ` + "`json:\"country\" mapper:\"bind:Country\"`" + `
	Age     int     ` + "`json:\"age\" mapper:\"bind:Age\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(appPkgDir, "model.go"), []byte(appSrc), 0644); err != nil {
		t.Fatalf("writing app/model.go: %v", err)
	}

	appPctx, err := loader.LoadPackage(appPkgDir)
	if err != nil {
		t.Fatalf("LoadPackage(app): %v", err)
	}

	extPctx, err := loader.LoadExternalPackage(tmpDir, modName+"/pb")
	if err != nil {
		t.Fatalf("LoadExternalPackage(pb): %v", err)
	}

	localModel, err := loader.LoadStructFromPkg(appPctx, "LocalModel")
	if err != nil {
		t.Fatalf("load LocalModel: %v", err)
	}

	protoMsg, err := loader.LoadStructFromPkg(extPctx, "ProtoMessage")
	if err != nil {
		t.Fatalf("load ProtoMessage: %v", err)
	}

	getters := loader.DiscoverGetters(extPctx, "ProtoMessage")

	crossResult := matcher.MatchCross(localModel, protoMsg, getters)

	ccfg := generator.CrossConfig{
		PackageName:          "app",
		InternalType:         "LocalModel",
		ExternalType:         "ProtoMessage",
		ExternalPkgName:      "pb",
		ExternalPkgPath:      modName + "/pb",
		ToExternalFuncName:   "MapLocalModelToProtoMessage",
		FromExternalFuncName: "MapProtoMessageToLocalModel",
		ToExternalPairs:      crossResult.ToExternal.Pairs,
		FromExternalPairs:    crossResult.FromExternal.Pairs,
		Bidirectional:        true,
	}

	code, err := generator.GenerateCross(ccfg)
	if err != nil {
		t.Fatalf("GenerateCross: %v", err)
	}

	output := string(code)

	t.Run("parses_as_valid_go", func(t *testing.T) {
		parseAndVerify(t, code)
	})

	// Proto *string -> local string: getter returns string, so direct assignment (NoneConversion).
	t.Run("proto_ptr_to_local_value_uses_getter_no_deref", func(t *testing.T) {
		assertContains(t, output, "dst.Name = src.GetName()")
		assertContains(t, output, "dst.Age = src.GetAge()")
	})

	// Proto *string -> local *string: skip getter, use direct field access to preserve nil.
	t.Run("proto_ptr_to_local_ptr_skips_getter_for_nil_safety", func(t *testing.T) {
		assertContains(t, output, "dst.Country = src.Country")
	})

	// Write and verify the file compiles.
	genPath := filepath.Join(appPkgDir, "mapper_gen.go")
	if err := os.WriteFile(genPath, code, 0644); err != nil {
		t.Fatalf("writing generated file: %v", err)
	}

	t.Run("generated_code_compiles", func(t *testing.T) {
		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = tmpDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("generated code does not compile:\n%s\n%s", out, output)
		}
	})
}

func TestSuite4_EndToEnd_MatcherFieldVerification(t *testing.T) {
	// End-to-end test that uses LoadPackage on testdata/integration
	// and verifies all match behavior using real loaded types.
	dir := integrationDir(t)
	pctx, err := loader.LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	tests := []struct {
		srcType       string
		dstType       string
		wantPairs     int
		wantUnmatched int
	}{
		{"PtrSource", "PtrDest", 5, 0},
		{"Company", "CompanyDTO", 2, 0},
		{"Employee", "EmployeeDTO", 3, 0},
		{"Address", "AddressDTO", 3, 0},
		{"TeamWithPtrSlice", "TeamWithValSlice", 2, 0},
		{"Node", "NodeDTO", 4, 0},
	}

	for _, tt := range tests {
		t.Run(tt.srcType+"_to_"+tt.dstType, func(t *testing.T) {
			srcInfo, err := loader.LoadStructFromPkg(pctx, tt.srcType)
			if err != nil {
				t.Fatalf("load %s: %v", tt.srcType, err)
			}

			dstInfo, err := loader.LoadStructFromPkg(pctx, tt.dstType)
			if err != nil {
				t.Fatalf("load %s: %v", tt.dstType, err)
			}

			result := matcher.Match(srcInfo, dstInfo, matcher.MatchOptions{})

			if got := len(result.Pairs); got != tt.wantPairs {
				t.Errorf("pairs: got %d, want %d", got, tt.wantPairs)
			}
			if got := len(result.Unmatched); got != tt.wantUnmatched {
				t.Errorf("unmatched: got %d, want %d", got, tt.wantUnmatched)
			}
		})
	}
}
