package loader

import (
	"path/filepath"
	"runtime"
	"testing"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func TestLoadStruct_Source(t *testing.T) {
	dir := testdataDir(t)
	info, err := LoadStruct(dir, "Source")
	if err != nil {
		t.Fatalf("LoadStruct: %v", err)
	}

	if info.Name != "Source" {
		t.Errorf("got name %q, want %q", info.Name, "Source")
	}
	if info.PackageName != "testdata" {
		t.Errorf("got package %q, want %q", info.PackageName, "testdata")
	}

	// Should only have exported fields: ID, FirstName, Email, Age (4 fields).
	if got := len(info.Fields); got != 4 {
		t.Fatalf("got %d fields, want 4", got)
	}

	byName := make(map[string]StructField)
	for _, f := range info.Fields {
		byName[f.Name] = f
	}

	// ID: int, json:"id"
	if f := byName["ID"]; f.JSONName != "id" || f.IsPtr {
		t.Errorf("ID field: json=%q, isPtr=%v", f.JSONName, f.IsPtr)
	}

	// Email: *string, json:"email"
	if f := byName["Email"]; f.JSONName != "email" || !f.IsPtr || f.ElemType != "string" {
		t.Errorf("Email field: json=%q, isPtr=%v, elemType=%q", f.JSONName, f.IsPtr, f.ElemType)
	}

	// Age: int, no json tag
	if f := byName["Age"]; f.JSONName != "" || f.IsPtr {
		t.Errorf("Age field: json=%q, isPtr=%v", f.JSONName, f.IsPtr)
	}
}

func TestLoadStruct_Destination(t *testing.T) {
	dir := testdataDir(t)
	info, err := LoadStruct(dir, "Destination")
	if err != nil {
		t.Fatalf("LoadStruct: %v", err)
	}

	if info.MapperFunc != "CustomMap" {
		t.Errorf("got struct mapper %q, want %q", info.MapperFunc, "CustomMap")
	}

	byName := make(map[string]StructField)
	for _, f := range info.Fields {
		byName[f.Name] = f
	}

	// FullName should have custom mapper func.
	if f := byName["FullName"]; f.MapperFn != "BuildFullName" {
		t.Errorf("FullName mapper func: got %q, want %q", f.MapperFn, "BuildFullName")
	}

	// Age should be a pointer.
	if f := byName["Age"]; !f.IsPtr || f.ElemType != "int" {
		t.Errorf("Age field: isPtr=%v, elemType=%q", f.IsPtr, f.ElemType)
	}
}

func TestLoadStruct_NotFound(t *testing.T) {
	dir := testdataDir(t)
	_, err := LoadStruct(dir, "NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent type")
	}
}

func TestLoadStruct_NestedStruct(t *testing.T) {
	dir := testdataDir(t)
	info, err := LoadStruct(dir, "User")
	if err != nil {
		t.Fatalf("LoadStruct: %v", err)
	}

	byName := make(map[string]StructField)
	for _, f := range info.Fields {
		byName[f.Name] = f
	}

	// Address should be detected as a named struct.
	addr := byName["Address"]
	if !addr.IsNamedStruct {
		t.Error("Address field: expected IsNamedStruct=true")
	}
	if addr.StructName != "Address" {
		t.Errorf("Address field: got StructName=%q, want %q", addr.StructName, "Address")
	}
	if addr.IsPtr {
		t.Error("Address field: expected IsPtr=false")
	}

	// Emails should be detected as a slice of named structs.
	emails := byName["Emails"]
	if !emails.IsSlice {
		t.Error("Emails field: expected IsSlice=true")
	}
	if !emails.IsSliceElemStruct {
		t.Error("Emails field: expected IsSliceElemStruct=true")
	}
	if emails.SliceElemTypeName != "EmailInfo" {
		t.Errorf("Emails field: got SliceElemTypeName=%q, want %q", emails.SliceElemTypeName, "EmailInfo")
	}
}

func TestLoadStruct_PtrNestedStruct(t *testing.T) {
	dir := testdataDir(t)
	info, err := LoadStruct(dir, "UserWithPtr")
	if err != nil {
		t.Fatalf("LoadStruct: %v", err)
	}

	byName := make(map[string]StructField)
	for _, f := range info.Fields {
		byName[f.Name] = f
	}

	addr := byName["Address"]
	if !addr.IsPtr {
		t.Error("Address field: expected IsPtr=true")
	}
	if !addr.IsNamedStruct {
		t.Error("Address field: expected IsNamedStruct=true")
	}
	if addr.StructName != "Address" {
		t.Errorf("Address field: got StructName=%q, want %q", addr.StructName, "Address")
	}
}

func TestLoadPackage_AndLoadStructFromPkg(t *testing.T) {
	dir := testdataDir(t)
	pctx, err := LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	// Load multiple structs from the same package context.
	for _, name := range []string{"Source", "Destination", "User", "UserDTO"} {
		info, err := LoadStructFromPkg(pctx, name)
		if err != nil {
			t.Errorf("LoadStructFromPkg(%s): %v", name, err)
			continue
		}
		if info.Name != name {
			t.Errorf("got name %q, want %q", info.Name, name)
		}
	}
}

func TestDiscoverMapperFuncs(t *testing.T) {
	dir := testdataDir(t)
	pctx, err := LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	funcs := DiscoverMapperFuncs(pctx)

	if !funcs["MapAddressToAddressDTO"] {
		t.Error("expected MapAddressToAddressDTO to be discovered")
	}
}

// --- New tests for bind, bind_json, getters, and external package loading ---

func TestLoadStruct_BindTag(t *testing.T) {
	dir := testdataDir(t)
	info, err := LoadStruct(dir, "InternalUser")
	if err != nil {
		t.Fatalf("LoadStruct: %v", err)
	}

	byName := make(map[string]StructField)
	for _, f := range info.Fields {
		byName[f.Name] = f
	}

	// ID should have bind:UserId.
	idField := byName["ID"]
	if idField.BindName != "UserId" {
		t.Errorf("ID field: got BindName=%q, want %q", idField.BindName, "UserId")
	}

	// FullName should have bind_json:display_name.
	fnField := byName["FullName"]
	if fnField.BindJSON != "display_name" {
		t.Errorf("FullName field: got BindJSON=%q, want %q", fnField.BindJSON, "display_name")
	}

	// Email should have no bind tags.
	emailField := byName["Email"]
	if emailField.BindName != "" {
		t.Errorf("Email field: got unexpected BindName=%q", emailField.BindName)
	}
	if emailField.BindJSON != "" {
		t.Errorf("Email field: got unexpected BindJSON=%q", emailField.BindJSON)
	}
}

func TestLoadStruct_BindOnlyInternal(t *testing.T) {
	dir := testdataDir(t)
	info, err := LoadStruct(dir, "BindOnlyInternal")
	if err != nil {
		t.Fatalf("LoadStruct: %v", err)
	}

	byName := make(map[string]StructField)
	for _, f := range info.Fields {
		byName[f.Name] = f
	}

	if f := byName["MyID"]; f.BindName != "ExternalID" {
		t.Errorf("MyID.BindName = %q, want %q", f.BindName, "ExternalID")
	}
	if f := byName["MyName"]; f.BindName != "ExternalName" {
		t.Errorf("MyName.BindName = %q, want %q", f.BindName, "ExternalName")
	}
}

func TestLoadStruct_BindJSONInternal(t *testing.T) {
	dir := testdataDir(t)
	info, err := LoadStruct(dir, "BindJSONInternal")
	if err != nil {
		t.Fatalf("LoadStruct: %v", err)
	}

	byName := make(map[string]StructField)
	for _, f := range info.Fields {
		byName[f.Name] = f
	}

	if f := byName["LocalTitle"]; f.BindJSON != "ext_title" {
		t.Errorf("LocalTitle.BindJSON = %q, want %q", f.BindJSON, "ext_title")
	}
	if f := byName["LocalCount"]; f.BindJSON != "ext_count" {
		t.Errorf("LocalCount.BindJSON = %q, want %q", f.BindJSON, "ext_count")
	}
}

func TestDiscoverGetters(t *testing.T) {
	dir := testdataDir(t)
	pctx, err := LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	getters := DiscoverGetters(pctx, "ExternalUser")

	expectedGetters := []struct {
		fieldName  string
		methodName string
	}{
		{"UserId", "GetUserId"},
		{"DisplayName", "GetDisplayName"},
		{"Email", "GetEmail"},
		{"Age", "GetAge"},
	}

	for _, exp := range expectedGetters {
		gi, ok := getters[exp.fieldName]
		if !ok {
			t.Errorf("expected getter for field %q, but not found", exp.fieldName)
			continue
		}
		if gi.MethodName != exp.methodName {
			t.Errorf("getter for %q: got MethodName=%q, want %q", exp.fieldName, gi.MethodName, exp.methodName)
		}
	}

	if len(getters) != len(expectedGetters) {
		t.Errorf("got %d getters, want %d", len(getters), len(expectedGetters))
	}
}

func TestDiscoverGetters_NoGetters(t *testing.T) {
	dir := testdataDir(t)
	pctx, err := LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	// Address has no getter methods.
	getters := DiscoverGetters(pctx, "Address")
	if len(getters) != 0 {
		t.Errorf("expected 0 getters for Address, got %d", len(getters))
	}
}

func TestDiscoverGetters_NonExistentType(t *testing.T) {
	dir := testdataDir(t)
	pctx, err := LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	getters := DiscoverGetters(pctx, "NonExistent")
	if len(getters) != 0 {
		t.Errorf("expected 0 getters for non-existent type, got %d", len(getters))
	}
}

func TestBaseTypeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"time.Time", "Time"},
		{"string", "String"},
		{"int32", "Int32"},
		{"*string", "String"},
		{"float64", "Float64"},
		{"int", "Int"},
		{"CustomType", "CustomType"},
		{"pkg.CustomType", "CustomType"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := BaseTypeName(tt.input)
			if got != tt.want {
				t.Errorf("BaseTypeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConverterFuncName(t *testing.T) {
	tests := []struct {
		srcType, dstType string
		want             string
	}{
		{"time.Time", "string", "MapTimeToString"},
		{"string", "time.Time", "MapStringToTime"},
		{"int64", "string", "MapInt64ToString"},
		{"string", "int64", "MapStringToInt64"},
	}

	for _, tt := range tests {
		t.Run(tt.srcType+"->"+tt.dstType, func(t *testing.T) {
			got := ConverterFuncName(tt.srcType, tt.dstType)
			if got != tt.want {
				t.Errorf("ConverterFuncName(%q, %q) = %q, want %q", tt.srcType, tt.dstType, got, tt.want)
			}
		})
	}
}

func TestFindConverterFunc(t *testing.T) {
	dir := testdataDir(t)
	pctx, err := LoadPackage(dir)
	if err != nil {
		t.Fatalf("LoadPackage: %v", err)
	}

	t.Run("finds_MapStringToInt64", func(t *testing.T) {
		fn := FindConverterFunc(pctx, "string", "int64")
		if fn != "MapStringToInt64" {
			t.Errorf("got %q, want %q", fn, "MapStringToInt64")
		}
	})

	t.Run("returns_empty_for_nonexistent", func(t *testing.T) {
		fn := FindConverterFunc(pctx, "time.Time", "string")
		if fn != "" {
			t.Errorf("expected empty string, got %q", fn)
		}
	})
}

func TestLoadExternalPackage(t *testing.T) {
	dir := testdataDir(t)
	// Load the testdata package itself as an "external" package by import path.
	pctx, err := LoadExternalPackage(dir, "github.com/hacks1ash/goxmap/internal/loader/testdata")
	if err != nil {
		t.Fatalf("LoadExternalPackage: %v", err)
	}

	info, err := LoadStructFromPkg(pctx, "ExternalUser")
	if err != nil {
		t.Fatalf("LoadStructFromPkg: %v", err)
	}

	if info.Name != "ExternalUser" {
		t.Errorf("got name %q, want %q", info.Name, "ExternalUser")
	}
	if info.PackageName != "testdata" {
		t.Errorf("got package %q, want %q", info.PackageName, "testdata")
	}
}
