// Package matcher provides field matching logic between source and destination structs.
//
// Fields are matched using strategies applied in priority order:
//  1. bind tag: `mapper:"bind:ExternalFieldName"` - direct field name binding
//  2. bind_json tag: `mapper:"bind_json:external_json_key"` - external json key binding
//  3. JSON tag name matching
//  4. Field name matching
package matcher

import (
	"strings"

	"github.com/hacks1ash/goxmap/internal/loader"
)

// GetterMode controls whether the generated mapper reads fields via getter
// methods or directly.
type GetterMode int

const (
	GetterModeAuto     GetterMode = iota // Use getter when one exists, else direct field access.
	GetterModeForce                       // Always use getter; mark MissingGetterForce when absent.
	GetterModeDisabled                    // Always use direct field access.
)

// MatchOptions configures Match behavior. SrcGetters and DstGetters map field
// names to discovered getter methods on the corresponding struct.
type MatchOptions struct {
	SrcGetters map[string]loader.GetterInfo
	DstGetters map[string]loader.GetterInfo
	GetterMode GetterMode
}

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

	// TypeCast is true when src and dst are different named types with the same
	// underlying type (e.g., type StatusA string -> type StatusB string).
	TypeCast bool
	// CastTypeName is the destination type name for named type casts.
	CastTypeName string

	// GetterReturnsPtr indicates whether the getter method returns a pointer type.
	// For proto optional fields, the field is *T but the getter returns T.
	// This is used to adjust pointer conversion when generating code with getters.
	GetterReturnsPtr bool

	// MissingGetterForce indicates that GetterMode was Force but no getter was
	// discovered for the source field. The caller is expected to surface this as
	// a fatal CLI error.
	MissingGetterForce bool
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

// numericFamily classifies numeric types into signed, unsigned, or float families.
type numericFamily int

const (
	familySigned numericFamily = iota
	familyUnsigned
	familyFloat
)

// numericRank maps numeric types to their bit width for narrowing detection.
// Platform-dependent types (int, uint) use 64 because they may be 64-bit.
var numericRank = map[string]int{
	"int8": 8, "int16": 16, "int32": 32, "int64": 64, "int": 64,
	"uint8": 8, "byte": 8, "uint16": 16, "uint32": 32, "uint64": 64, "uint": 64,
	"float32": 32, "float64": 64,
	"rune": 32,
}

// getFamily returns the numeric family for a given type name.
func getFamily(t string) numericFamily {
	switch t {
	case "uint", "uint8", "uint16", "uint32", "uint64", "byte":
		return familyUnsigned
	case "float32", "float64":
		return familyFloat
	default:
		return familySigned
	}
}

// IsNarrowingCast reports whether a numeric conversion from src to dst may lose data.
// Cross-family conversions (signed<->unsigned, int<->float) are always narrowing.
// Same-family conversions are narrowing when destination has fewer bits than source.
func IsNarrowingCast(src, dst string) bool {
	srcFamily := getFamily(src)
	dstFamily := getFamily(dst)

	// Cross-family is always considered narrowing (potential overflow or precision loss).
	if srcFamily != dstFamily {
		return true
	}

	// Same family: narrowing if destination rank is smaller than source rank.
	return numericRank[dst] < numericRank[src]
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
		if IsNarrowingCast(srcElem, dstElem) {
			// Narrowing casts are treated as type mismatches, requiring an
			// explicit converter function to prevent silent data loss.
			pair.TypeMismatch = true
			return
		}
		pair.NumericCast = true
		pair.CastType = dstElem
		return
	}

	// Same-underlying-type cast (enum support).
	if pair.Src.IsNamedNonStruct && pair.Dst.IsNamedNonStruct &&
		pair.Src.UnderlyingTypeName == pair.Dst.UnderlyingTypeName {
		pair.TypeCast = true
		pair.CastTypeName = pair.Dst.ElemType
		return
	}

	// Otherwise, mark as a type mismatch for later resolution.
	pair.TypeMismatch = true
}

// Match finds corresponding fields between the source and destination structs
// using a unified 4-priority strategy. Bind tags (BindName, BindJSON) on the
// destination field are consulted first, followed by JSON tag matching, then
// field name matching. opts controls getter probing behavior.
//
// Passing MatchOptions{} (zero value) is identical to the old 2-arg Match call.
func Match(src, dst *loader.StructInfo, opts MatchOptions) MatchResult {
	// Build source lookup maps.
	// When a src field carries a BindName, that alias takes precedence over the
	// field's own Go name in the lookup table. This lets a src field declare an
	// alternate identity so dst fields with that name resolve to it.
	srcByName := make(map[string]loader.StructField)
	srcByJSON := make(map[string]loader.StructField)

	for _, f := range src.Fields {
		if f.BindName != "" {
			// Register under the alias only; the original name is shadowed.
			srcByName[f.BindName] = f
		} else {
			srcByName[f.Name] = f
		}
		if f.JSONName != "" {
			srcByJSON[f.JSONName] = f
		}
	}

	var result MatchResult

	for _, df := range dst.Fields {
		// Skip fields marked with mapper:"ignore".
		if df.Ignore {
			continue
		}

		var matched bool
		var srcField loader.StructField

		// Priority 1: bind tag on dst — direct src field name binding.
		if df.BindName != "" {
			if sf, ok := srcByName[df.BindName]; ok {
				srcField = sf
				matched = true
			}
		}

		// Priority 2: bind_json tag on dst — find src field by its json tag.
		if !matched && df.BindJSON != "" {
			if sf, ok := srcByJSON[df.BindJSON]; ok {
				srcField = sf
				matched = true
			}
		}

		// Priority 3: json tag match.
		if !matched && df.JSONName != "" {
			if sf, ok := srcByJSON[df.JSONName]; ok {
				srcField = sf
				matched = true
			}
		}

		// Priority 4: field name match.
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

			// Apply getter rules using opts.SrcGetters.
			switch opts.GetterMode {
			case GetterModeAuto:
				if opts.SrcGetters != nil {
					if gi, ok := opts.SrcGetters[srcField.Name]; ok {
						pair.UseGetter = true
						pair.GetterName = gi.MethodName
						pair.GetterReturnsPtr = strings.HasPrefix(gi.ReturnType, "*")
					}
				}
			case GetterModeForce:
				if opts.SrcGetters != nil {
					if gi, ok := opts.SrcGetters[srcField.Name]; ok {
						pair.UseGetter = true
						pair.GetterName = gi.MethodName
						pair.GetterReturnsPtr = strings.HasPrefix(gi.ReturnType, "*")
					} else {
						pair.MissingGetterForce = true
					}
				} else {
					pair.MissingGetterForce = true
				}
			case GetterModeDisabled:
				// Always use direct field access — nothing to do.
			}

			result.Pairs = append(result.Pairs, pair)
		} else {
			result.Unmatched = append(result.Unmatched, df)
		}
	}

	return result
}

