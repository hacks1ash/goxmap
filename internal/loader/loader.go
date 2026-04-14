// Package loader provides type loading and analysis using golang.org/x/tools/go/packages.
package loader

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

// StructField represents a single field extracted from a struct type.
type StructField struct {
	Name     string // Go field name
	TypeStr  string // Full type string (e.g. "*string", "int", "time.Time")
	JSONName string // Value from `json` tag, empty if absent
	MapperFn string // Value from `mapper:"func:Xxx"` tag, empty if absent
	IsPtr    bool   // Whether the field type is a pointer
	ElemType string // Underlying type if pointer, same as TypeStr otherwise

	// Enhanced type info for nested struct and slice support.
	IsNamedStruct     bool   // Whether the (dereferenced) type is a named struct
	StructName        string // The simple name of the struct type (e.g. "Address")
	IsSlice           bool   // Whether the field is a slice ([]T or []*T)
	SliceElemType     string // Type string of the slice element (e.g. "Email", "*Email")
	SliceElemIsPtr    bool   // Whether the slice element is a pointer
	SliceElemTypeName string // Named struct name of slice element, empty if primitive
	IsSliceElemStruct bool   // Whether the slice element is a named struct

	// Cross-package binding support.
	BindName string // Value from `mapper:"bind:Xxx"` tag - direct field name binding
	BindJSON string // Value from `mapper:"bind_json:xxx"` tag - external json key binding
}

// StructInfo holds metadata about a loaded struct.
type StructInfo struct {
	PackageName string
	PackagePath string
	Name        string
	Fields      []StructField
	MapperFunc  string // Struct-level custom mapper from `mapper:"struct_func:Xxx"` tag
}

// GetterInfo describes a getter method on a struct type.
type GetterInfo struct {
	MethodName string // e.g. "GetName"
	FieldName  string // e.g. "Name" (derived by stripping "Get" prefix)
	ReturnType string // Return type string
}

// PackageContext holds loaded package information for reuse across multiple
// struct loads and mapper discovery.
type PackageContext struct {
	Pkg *packages.Package
}

// LoadPackage loads and returns the package at the given directory.
func LoadPackage(dir string) (*PackageContext, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax |
			packages.NeedImports,
		Dir: dir,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("loading package in %s: %w", dir, err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in %s", dir)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		var msgs []string
		for _, e := range pkg.Errors {
			msgs = append(msgs, e.Error())
		}
		return nil, fmt.Errorf("package errors: %s", strings.Join(msgs, "; "))
	}

	return &PackageContext{Pkg: pkg}, nil
}

// LoadExternalPackage loads a package by its import path (e.g.,
// "github.com/org/repo/proto"). The dir parameter is the working directory
// from which module resolution happens.
func LoadExternalPackage(dir, importPath string) (*PackageContext, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax |
			packages.NeedImports,
		Dir: dir,
	}

	pkgs, err := packages.Load(cfg, importPath)
	if err != nil {
		return nil, fmt.Errorf("loading external package %s: %w", importPath, err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for %s", importPath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		var msgs []string
		for _, e := range pkg.Errors {
			msgs = append(msgs, e.Error())
		}
		return nil, fmt.Errorf("external package errors: %s", strings.Join(msgs, "; "))
	}

	return &PackageContext{Pkg: pkg}, nil
}

// LoadStructFromPkg loads the named struct from an already-loaded package context.
func LoadStructFromPkg(pctx *PackageContext, typeName string) (*StructInfo, error) {
	pkg := pctx.Pkg

	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		return nil, fmt.Errorf("type %s not found in package %s", typeName, pkg.PkgPath)
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("%s is not a named type", typeName)
	}

	underlying, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%s is not a struct type", typeName)
	}

	info := &StructInfo{
		PackageName: pkg.Name,
		PackagePath: pkg.PkgPath,
		Name:        typeName,
	}

	q := qualifier(pkg.Types)

	for i := 0; i < underlying.NumFields(); i++ {
		field := underlying.Field(i)
		if !field.Exported() {
			continue
		}

		sf := StructField{
			Name:    field.Name(),
			TypeStr: types.TypeString(field.Type(), q),
		}

		tag := underlying.Tag(i)
		sf.JSONName = parseJSONTag(tag)
		sf.MapperFn = parseMapperFuncTag(tag)
		sf.BindName = parseMapperBindTag(tag)
		sf.BindJSON = parseMapperBindJSONTag(tag)

		analyzeFieldType(field.Type(), &sf, q)

		info.Fields = append(info.Fields, sf)
	}

	info.MapperFunc = parseStructLevelMapper(underlying, pkg.Types)

	return info, nil
}

// LoadStruct loads the named struct from the package at the given directory.
func LoadStruct(dir, typeName string) (*StructInfo, error) {
	pctx, err := LoadPackage(dir)
	if err != nil {
		return nil, err
	}
	return LoadStructFromPkg(pctx, typeName)
}

// DiscoverMapperFuncs scans the package for existing functions matching the
// naming convention Map[Source]To[Dest]. It returns a set of function names.
func DiscoverMapperFuncs(pctx *PackageContext) map[string]bool {
	result := make(map[string]bool)
	scope := pctx.Pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if _, ok := obj.(*types.Func); ok {
			if strings.HasPrefix(name, "Map") && strings.Contains(name, "To") {
				result[name] = true
			}
		}
	}
	return result
}

// DiscoverGetters finds all exported methods on the named type that follow the
// GetXxx() pattern (single return value, no parameters beyond the receiver).
// This is used for Protobuf-style getter detection.
func DiscoverGetters(pctx *PackageContext, typeName string) map[string]GetterInfo {
	result := make(map[string]GetterInfo)

	obj := pctx.Pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		return result
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		return result
	}

	q := qualifier(pctx.Pkg.Types)

	// Check methods on *T (pointer receiver) and T (value receiver).
	// For protobuf, getters are typically on *T.
	ptrType := types.NewPointer(named)
	mset := types.NewMethodSet(ptrType)

	for i := 0; i < mset.Len(); i++ {
		sel := mset.At(i)
		fn, ok := sel.Obj().(*types.Func)
		if !ok || !fn.Exported() {
			continue
		}

		name := fn.Name()
		if !strings.HasPrefix(name, "Get") || len(name) <= 3 {
			continue
		}

		sig, ok := fn.Type().(*types.Signature)
		if !ok {
			continue
		}

		// Must take no parameters (beyond receiver) and return exactly one value.
		if sig.Params().Len() != 0 || sig.Results().Len() != 1 {
			continue
		}

		fieldName := name[3:] // Strip "Get" prefix
		retType := types.TypeString(sig.Results().At(0).Type(), q)

		result[fieldName] = GetterInfo{
			MethodName: name,
			FieldName:  fieldName,
			ReturnType: retType,
		}
	}

	return result
}

// BaseTypeName extracts a capitalized base type name from a full Go type string.
// Examples:
//
//	"time.Time" -> "Time"
//	"string" -> "String"
//	"int32" -> "Int32"
//	"*string" -> "String"
//	"[]byte" -> "Bytes"
func BaseTypeName(typeStr string) string {
	// Strip pointer prefix.
	t := strings.TrimPrefix(typeStr, "*")

	// Handle slice types.
	if strings.HasPrefix(t, "[]") {
		inner := strings.TrimPrefix(t, "[]")
		return capitalize(inner) + "s"
	}

	// Strip package qualifier.
	if idx := strings.LastIndex(t, "."); idx >= 0 {
		t = t[idx+1:]
	}

	return capitalize(t)
}

// capitalize returns s with its first rune uppercased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = toUpper(r[0])
	return string(r)
}

func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 'a' + 'A'
	}
	return r
}

// ConverterFuncName computes the expected converter function name for a
// src -> dst type mapping following the Map<SrcBase>To<DstBase> convention.
func ConverterFuncName(srcType, dstType string) string {
	return "Map" + BaseTypeName(srcType) + "To" + BaseTypeName(dstType)
}

// FindConverterFunc looks up a converter function in the package scope that
// matches the Map<SrcType>To<DstType> convention and has the right signature
// (one parameter, one return value). Returns the function name if found.
func FindConverterFunc(pctx *PackageContext, srcType, dstType string) string {
	expectedName := ConverterFuncName(srcType, dstType)
	scope := pctx.Pkg.Types.Scope()
	obj := scope.Lookup(expectedName)
	if obj == nil {
		return ""
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return ""
	}

	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return ""
	}

	// Must have exactly one parameter and one result.
	if sig.Params().Len() != 1 || sig.Results().Len() != 1 {
		return ""
	}

	return expectedName
}

// analyzeFieldType populates the enhanced type information on a StructField.
func analyzeFieldType(t types.Type, sf *StructField, q types.Qualifier) {
	actual := t
	if ptr, ok := t.(*types.Pointer); ok {
		sf.IsPtr = true
		sf.ElemType = types.TypeString(ptr.Elem(), q)
		actual = ptr.Elem()
	} else {
		sf.ElemType = sf.TypeStr
	}

	if named, ok := actual.(*types.Named); ok {
		if _, isStruct := named.Underlying().(*types.Struct); isStruct {
			sf.IsNamedStruct = true
			sf.StructName = named.Obj().Name()
		}
	}

	if sl, ok := actual.(*types.Slice); ok {
		sf.IsSlice = true
		elem := sl.Elem()
		sf.SliceElemType = types.TypeString(elem, q)

		if ptr, ok := elem.(*types.Pointer); ok {
			sf.SliceElemIsPtr = true
			inner := ptr.Elem()
			sf.SliceElemType = types.TypeString(inner, q)
			if named, ok := inner.(*types.Named); ok {
				if _, isStruct := named.Underlying().(*types.Struct); isStruct {
					sf.IsSliceElemStruct = true
					sf.SliceElemTypeName = named.Obj().Name()
				}
			}
		} else if named, ok := elem.(*types.Named); ok {
			if _, isStruct := named.Underlying().(*types.Struct); isStruct {
				sf.IsSliceElemStruct = true
				sf.SliceElemTypeName = named.Obj().Name()
			}
		}
	}
}

func qualifier(pkg *types.Package) types.Qualifier {
	return func(other *types.Package) string {
		if pkg == other {
			return ""
		}
		return other.Name()
	}
}

func parseJSONTag(rawTag string) string {
	tag := reflect.StructTag(rawTag)
	jsonVal, ok := tag.Lookup("json")
	if !ok {
		return ""
	}
	name, _, _ := strings.Cut(jsonVal, ",")
	if name == "-" {
		return ""
	}
	return name
}

func parseMapperFuncTag(rawTag string) string {
	tag := reflect.StructTag(rawTag)
	mapperVal, ok := tag.Lookup("mapper")
	if !ok {
		return ""
	}
	for _, part := range strings.Split(mapperVal, ";") {
		part = strings.TrimSpace(part)
		if after, found := strings.CutPrefix(part, "func:"); found {
			return after
		}
	}
	return ""
}

func parseMapperBindTag(rawTag string) string {
	tag := reflect.StructTag(rawTag)
	mapperVal, ok := tag.Lookup("mapper")
	if !ok {
		return ""
	}
	for _, part := range strings.Split(mapperVal, ";") {
		part = strings.TrimSpace(part)
		if after, found := strings.CutPrefix(part, "bind:"); found {
			return after
		}
	}
	return ""
}

func parseMapperBindJSONTag(rawTag string) string {
	tag := reflect.StructTag(rawTag)
	mapperVal, ok := tag.Lookup("mapper")
	if !ok {
		return ""
	}
	for _, part := range strings.Split(mapperVal, ";") {
		part = strings.TrimSpace(part)
		if after, found := strings.CutPrefix(part, "bind_json:"); found {
			return after
		}
	}
	return ""
}

func parseStructLevelMapper(s *types.Struct, pkg *types.Package) string {
	for i := 0; i < s.NumFields(); i++ {
		field := s.Field(i)
		if field.Name() == "_" {
			tag := reflect.StructTag(s.Tag(i))
			mapperVal, ok := tag.Lookup("mapper")
			if !ok {
				continue
			}
			if after, found := strings.CutPrefix(mapperVal, "struct_func:"); found {
				return after
			}
		}
	}
	return ""
}
