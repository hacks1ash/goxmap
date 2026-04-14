// Package matcher provides field matching logic between source and destination structs.
//
// Fields are matched using strategies applied in priority order:
//  1. bind tag: `mapper:"bind:ExternalFieldName"` - direct field name binding
//  2. bind_json tag: `mapper:"bind_json:external_json_key"` - external json key binding
//  3. JSON tag name matching
//  4. Field name matching
package matcher

import (
	"github.com/hacks1ash/goxmap/internal/loader"
)

// FieldPair represents a matched pair of fields between source and destination structs.
type FieldPair struct {
	Src loader.StructField
	Dst loader.StructField

	// UseGetter indicates that when reading from Src, a getter method should
	// be used instead of direct field access. This is set during cross-package
	// matching when the source struct has Protobuf-style getters.
	UseGetter  bool
	GetterName string // e.g. "GetName"

	// TypeMismatch is true when the source and destination element types differ
	// (after pointer peeling) and no built-in numeric coercion applies.
	TypeMismatch bool

	// ConverterFunc is the name of a converter function to use for type-mismatched
	// fields. It can be set via auto-discovery (Map<SrcType>To<DstType>) or
	// explicitly via `mapper:"func:..."` tag.
	ConverterFunc string

	// NumericCast is true when the source and destination are both numeric types
	// and a simple Go type cast can be used.
	NumericCast bool

	// CastType is the destination type name to use for numeric casts (e.g. "int32").
	CastType string
}

// PointerConversion describes how to handle pointer differences between matched fields.
type PointerConversion int

const (
	// NoneConversion means both fields are the same pointer-ness (T->T or *T->*T).
	NoneConversion PointerConversion = iota
	// DerefConversion means source is *T and destination is T.
	DerefConversion
	// AddrConversion means source is T and destination is *T.
	AddrConversion
)

// Conversion returns the pointer conversion needed for this field pair.
func (fp FieldPair) Conversion() PointerConversion {
	if fp.Src.IsPtr && !fp.Dst.IsPtr {
		return DerefConversion
	}
	if !fp.Src.IsPtr && fp.Dst.IsPtr {
		return AddrConversion
	}
	return NoneConversion
}

// HasCustomMapper reports whether the destination field has a custom mapper function.
func (fp FieldPair) HasCustomMapper() bool {
	return fp.Dst.MapperFn != ""
}

// NeedsNestedMapper reports whether this field pair involves a named struct
// that requires a sub-mapper function (and no custom mapper is set).
func (fp FieldPair) NeedsNestedMapper() bool {
	if fp.HasCustomMapper() {
		return false
	}
	return fp.Src.IsNamedStruct && fp.Dst.IsNamedStruct
}

// NeedsSliceMapper reports whether this field pair involves a slice of named
// structs that requires element-level mapping.
func (fp FieldPair) NeedsSliceMapper() bool {
	if fp.HasCustomMapper() {
		return false
	}
	return fp.Src.IsSlice && fp.Dst.IsSlice &&
		fp.Src.IsSliceElemStruct && fp.Dst.IsSliceElemStruct
}

// MatchResult holds the complete matching result between two structs.
type MatchResult struct {
	Pairs     []FieldPair
	Unmatched []loader.StructField // destination fields with no source match
}

// numericTypes is the set of Go numeric type names that support type casting.
var numericTypes = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true,
	"byte": true, "rune": true,
}

// IsNumericType reports whether the given type name is a Go numeric type.
func IsNumericType(t string) bool {
	return numericTypes[t]
}

// classifyPair sets the NumericCast/TypeMismatch fields on a FieldPair based
// on the element types of the source and destination fields.
func classifyPair(pair *FieldPair) {
	srcElem := pair.Src.ElemType
	dstElem := pair.Dst.ElemType

	// If element types match, nothing to do.
	if srcElem == dstElem {
		return
	}

	// Skip classification for nested structs and slices -- those are handled
	// by their own mapper discovery logic.
	if pair.NeedsNestedMapper() || pair.NeedsSliceMapper() {
		return
	}

	// If a custom mapper func tag is already set, use it as the converter.
	if pair.Dst.MapperFn != "" {
		pair.ConverterFunc = pair.Dst.MapperFn
		return
	}

	// Check for numeric-to-numeric coercion.
	if IsNumericType(srcElem) && IsNumericType(dstElem) {
		pair.NumericCast = true
		pair.CastType = dstElem
		return
	}

	// Otherwise, mark as a type mismatch for later resolution.
	pair.TypeMismatch = true
}

// Match finds corresponding fields between the source and destination structs.
// This is the original same-package matcher.
func Match(src, dst *loader.StructInfo) MatchResult {
	srcByJSON := make(map[string]loader.StructField)
	srcByName := make(map[string]loader.StructField)

	for _, f := range src.Fields {
		srcByName[f.Name] = f
		if f.JSONName != "" {
			srcByJSON[f.JSONName] = f
		}
	}

	var result MatchResult

	for _, df := range dst.Fields {
		var matched bool
		var srcField loader.StructField

		if df.JSONName != "" {
			if sf, ok := srcByJSON[df.JSONName]; ok {
				srcField = sf
				matched = true
			}
		}

		if !matched {
			if sf, ok := srcByName[df.Name]; ok {
				srcField = sf
				matched = true
			}
		}

		if matched {
			pair := FieldPair{
				Src: srcField,
				Dst: df,
			}
			classifyPair(&pair)
			result.Pairs = append(result.Pairs, pair)
		} else {
			result.Unmatched = append(result.Unmatched, df)
		}
	}

	return result
}

// CrossMatchResult holds bidirectional matching results for cross-package mapping.
type CrossMatchResult struct {
	// ToExternal contains pairs for mapping internal -> external.
	// Src is always the internal field, Dst is the external field.
	ToExternal MatchResult

	// FromExternal contains pairs for mapping external -> internal.
	// Src is the external field, Dst is the internal field.
	FromExternal MatchResult
}

// MatchCross performs cross-package field matching using the 4-level priority:
//  1. bind tag on internal field -> direct external field name
//  2. bind_json tag on internal field -> external field by json tag
//  3. json tag match
//  4. field name match
//
// The internal struct is the one with mapper tags (bind/bind_json).
// The external struct comes from a different package (e.g., Protobuf).
// getters maps field names to GetterInfo for the external struct (may be nil).
func MatchCross(internal, external *loader.StructInfo, getters map[string]loader.GetterInfo) CrossMatchResult {
	// Build external lookup maps.
	extByName := make(map[string]loader.StructField)
	extByJSON := make(map[string]loader.StructField)

	for _, f := range external.Fields {
		extByName[f.Name] = f
		if f.JSONName != "" {
			extByJSON[f.JSONName] = f
		}
	}

	var toExtResult MatchResult
	var fromExtResult MatchResult

	for _, inf := range internal.Fields {
		var matched bool
		var extField loader.StructField

		// Priority 1: bind tag - direct field name binding.
		if inf.BindName != "" {
			if ef, ok := extByName[inf.BindName]; ok {
				extField = ef
				matched = true
			}
		}

		// Priority 2: bind_json tag - find external field by its json tag.
		if !matched && inf.BindJSON != "" {
			if ef, ok := extByJSON[inf.BindJSON]; ok {
				extField = ef
				matched = true
			}
		}

		// Priority 3: json tag match.
		if !matched && inf.JSONName != "" {
			if ef, ok := extByJSON[inf.JSONName]; ok {
				extField = ef
				matched = true
			}
		}

		// Priority 4: field name match.
		if !matched {
			if ef, ok := extByName[inf.Name]; ok {
				extField = ef
				matched = true
			}
		}

		if !matched {
			toExtResult.Unmatched = append(toExtResult.Unmatched, inf)
			fromExtResult.Unmatched = append(fromExtResult.Unmatched, inf)
			continue
		}

		// ToExternal: internal (src) -> external (dst).
		toExtResult.Pairs = append(toExtResult.Pairs, FieldPair{
			Src: inf,
			Dst: extField,
		})

		// FromExternal: external (src) -> internal (dst).
		fromExtPair := FieldPair{
			Src: extField,
			Dst: inf,
		}

		// Check for getter on the external struct.
		if getters != nil {
			if gi, ok := getters[extField.Name]; ok {
				fromExtPair.UseGetter = true
				fromExtPair.GetterName = gi.MethodName
			}
		}

		fromExtResult.Pairs = append(fromExtResult.Pairs, fromExtPair)
	}

	return CrossMatchResult{
		ToExternal:   toExtResult,
		FromExternal: fromExtResult,
	}
}
