package matcher

import (
	"testing"

	"github.com/hacks1ash/goxmap/internal/loader"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		name          string
		src           *loader.StructInfo
		dst           *loader.StructInfo
		wantPairs     int
		wantUnmatched int
	}{
		{
			name: "exact name match",
			src: &loader.StructInfo{
				Fields: []loader.StructField{
					{Name: "ID", TypeStr: "int", ElemType: "int"},
					{Name: "Name", TypeStr: "string", ElemType: "string"},
				},
			},
			dst: &loader.StructInfo{
				Fields: []loader.StructField{
					{Name: "ID", TypeStr: "int", ElemType: "int"},
					{Name: "Name", TypeStr: "string", ElemType: "string"},
				},
			},
			wantPairs:     2,
			wantUnmatched: 0,
		},
		{
			name: "json tag match",
			src: &loader.StructInfo{
				Fields: []loader.StructField{
					{Name: "UserID", TypeStr: "int", ElemType: "int", JSONName: "user_id"},
				},
			},
			dst: &loader.StructInfo{
				Fields: []loader.StructField{
					{Name: "ID", TypeStr: "int", ElemType: "int", JSONName: "user_id"},
				},
			},
			wantPairs:     1,
			wantUnmatched: 0,
		},
		{
			name: "unmatched destination field",
			src: &loader.StructInfo{
				Fields: []loader.StructField{
					{Name: "ID", TypeStr: "int", ElemType: "int"},
				},
			},
			dst: &loader.StructInfo{
				Fields: []loader.StructField{
					{Name: "ID", TypeStr: "int", ElemType: "int"},
					{Name: "Extra", TypeStr: "string", ElemType: "string"},
				},
			},
			wantPairs:     1,
			wantUnmatched: 1,
		},
		{
			name: "json tag takes precedence over name",
			src: &loader.StructInfo{
				Fields: []loader.StructField{
					{Name: "Foo", TypeStr: "string", ElemType: "string", JSONName: "bar"},
					{Name: "Bar", TypeStr: "string", ElemType: "string"},
				},
			},
			dst: &loader.StructInfo{
				Fields: []loader.StructField{
					{Name: "Baz", TypeStr: "string", ElemType: "string", JSONName: "bar"},
				},
			},
			wantPairs:     1,
			wantUnmatched: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Match(tt.src, tt.dst)

			if got := len(result.Pairs); got != tt.wantPairs {
				t.Errorf("got %d pairs, want %d", got, tt.wantPairs)
			}
			if got := len(result.Unmatched); got != tt.wantUnmatched {
				t.Errorf("got %d unmatched, want %d", got, tt.wantUnmatched)
			}
		})
	}
}

func TestFieldPair_Conversion(t *testing.T) {
	tests := []struct {
		name string
		src  loader.StructField
		dst  loader.StructField
		want PointerConversion
	}{
		{
			name: "T to T",
			src:  loader.StructField{IsPtr: false},
			dst:  loader.StructField{IsPtr: false},
			want: NoneConversion,
		},
		{
			name: "*T to *T",
			src:  loader.StructField{IsPtr: true},
			dst:  loader.StructField{IsPtr: true},
			want: NoneConversion,
		},
		{
			name: "*T to T (deref)",
			src:  loader.StructField{IsPtr: true},
			dst:  loader.StructField{IsPtr: false},
			want: DerefConversion,
		},
		{
			name: "T to *T (addr)",
			src:  loader.StructField{IsPtr: false},
			dst:  loader.StructField{IsPtr: true},
			want: AddrConversion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pair := FieldPair{Src: tt.src, Dst: tt.dst}
			if got := pair.Conversion(); got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFieldPair_NeedsNestedMapper(t *testing.T) {
	tests := []struct {
		name string
		pair FieldPair
		want bool
	}{
		{
			name: "both named structs",
			pair: FieldPair{
				Src: loader.StructField{Name: "Address", IsNamedStruct: true, StructName: "Address"},
				Dst: loader.StructField{Name: "Address", IsNamedStruct: true, StructName: "AddressDTO"},
			},
			want: true,
		},
		{
			name: "primitive types",
			pair: FieldPair{
				Src: loader.StructField{Name: "ID", TypeStr: "int", ElemType: "int"},
				Dst: loader.StructField{Name: "ID", TypeStr: "int", ElemType: "int"},
			},
			want: false,
		},
		{
			name: "custom mapper overrides nested",
			pair: FieldPair{
				Src: loader.StructField{Name: "Address", IsNamedStruct: true, StructName: "Address"},
				Dst: loader.StructField{Name: "Address", IsNamedStruct: true, StructName: "AddressDTO", MapperFn: "CustomMapAddr"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pair.NeedsNestedMapper(); got != tt.want {
				t.Errorf("NeedsNestedMapper() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFieldPair_NeedsSliceMapper(t *testing.T) {
	tests := []struct {
		name string
		pair FieldPair
		want bool
	}{
		{
			name: "slice of named structs",
			pair: FieldPair{
				Src: loader.StructField{
					Name: "Emails", IsSlice: true,
					IsSliceElemStruct: true, SliceElemTypeName: "EmailInfo",
				},
				Dst: loader.StructField{
					Name: "Emails", IsSlice: true,
					IsSliceElemStruct: true, SliceElemTypeName: "EmailInfoDTO",
				},
			},
			want: true,
		},
		{
			name: "slice of primitives",
			pair: FieldPair{
				Src: loader.StructField{Name: "Tags", IsSlice: true, IsSliceElemStruct: false},
				Dst: loader.StructField{Name: "Tags", IsSlice: true, IsSliceElemStruct: false},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pair.NeedsSliceMapper(); got != tt.want {
				t.Errorf("NeedsSliceMapper() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- Numeric cast and type mismatch tests ---

func TestMatch_NumericCast_Widening(t *testing.T) {
	// Widening casts should still auto-cast.
	src := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "SmallAge", TypeStr: "int32", ElemType: "int32"},
			{Name: "Score", TypeStr: "float32", ElemType: "float32"},
		},
	}
	dst := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "SmallAge", TypeStr: "int64", ElemType: "int64"},
			{Name: "Score", TypeStr: "float64", ElemType: "float64"},
		},
	}

	result := Match(src, dst)
	if got := len(result.Pairs); got != 2 {
		t.Fatalf("got %d pairs, want 2", got)
	}

	for _, pair := range result.Pairs {
		if !pair.NumericCast {
			t.Errorf("field %s: expected NumericCast=true (widening)", pair.Dst.Name)
		}
		if pair.TypeMismatch {
			t.Errorf("field %s: expected TypeMismatch=false (widening)", pair.Dst.Name)
		}
	}

	if result.Pairs[0].CastType != "int64" {
		t.Errorf("SmallAge CastType: got %q, want %q", result.Pairs[0].CastType, "int64")
	}
	if result.Pairs[1].CastType != "float64" {
		t.Errorf("Score CastType: got %q, want %q", result.Pairs[1].CastType, "float64")
	}
}

func TestMatch_NumericCast_Narrowing(t *testing.T) {
	// Narrowing casts should be marked as type mismatches.
	src := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Age", TypeStr: "int64", ElemType: "int64"},
			{Name: "Score", TypeStr: "float64", ElemType: "float64"},
		},
	}
	dst := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Age", TypeStr: "int32", ElemType: "int32"},
			{Name: "Score", TypeStr: "float32", ElemType: "float32"},
		},
	}

	result := Match(src, dst)
	if got := len(result.Pairs); got != 2 {
		t.Fatalf("got %d pairs, want 2", got)
	}

	for _, pair := range result.Pairs {
		if pair.NumericCast {
			t.Errorf("field %s: expected NumericCast=false (narrowing)", pair.Dst.Name)
		}
		if !pair.TypeMismatch {
			t.Errorf("field %s: expected TypeMismatch=true (narrowing)", pair.Dst.Name)
		}
	}
}

func TestMatch_TypeMismatch(t *testing.T) {
	src := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "CreatedAt", TypeStr: "time.Time", ElemType: "time.Time"},
		},
	}
	dst := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "CreatedAt", TypeStr: "string", ElemType: "string"},
		},
	}

	result := Match(src, dst)
	if got := len(result.Pairs); got != 1 {
		t.Fatalf("got %d pairs, want 1", got)
	}

	pair := result.Pairs[0]
	if !pair.TypeMismatch {
		t.Error("expected TypeMismatch=true for time.Time -> string")
	}
	if pair.NumericCast {
		t.Error("expected NumericCast=false for time.Time -> string")
	}
}

func TestMatch_CustomMapperPreventsTypeMismatch(t *testing.T) {
	src := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "CreatedAt", TypeStr: "time.Time", ElemType: "time.Time"},
		},
	}
	dst := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "CreatedAt", TypeStr: "string", ElemType: "string", MapperFn: "FormatTime"},
		},
	}

	result := Match(src, dst)
	if got := len(result.Pairs); got != 1 {
		t.Fatalf("got %d pairs, want 1", got)
	}

	pair := result.Pairs[0]
	if pair.TypeMismatch {
		t.Error("expected TypeMismatch=false when MapperFn is set")
	}
	if pair.ConverterFunc != "FormatTime" {
		t.Errorf("expected ConverterFunc=%q, got %q", "FormatTime", pair.ConverterFunc)
	}
}

func TestMatch_PtrNumericCast_Widening(t *testing.T) {
	// *int32 -> int64 is widening, should auto-cast.
	src := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Count", TypeStr: "*int32", ElemType: "int32", IsPtr: true},
		},
	}
	dst := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Count", TypeStr: "int64", ElemType: "int64", IsPtr: false},
		},
	}

	result := Match(src, dst)
	if got := len(result.Pairs); got != 1 {
		t.Fatalf("got %d pairs, want 1", got)
	}

	pair := result.Pairs[0]
	if !pair.NumericCast {
		t.Error("expected NumericCast=true for *int32 -> int64 (widening)")
	}
	if pair.CastType != "int64" {
		t.Errorf("CastType: got %q, want %q", pair.CastType, "int64")
	}
}

func TestMatch_PtrNumericCast_Narrowing(t *testing.T) {
	// *int -> int32 is narrowing (int is 64-bit on most platforms).
	src := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Count", TypeStr: "*int", ElemType: "int", IsPtr: true},
		},
	}
	dst := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Count", TypeStr: "int32", ElemType: "int32", IsPtr: false},
		},
	}

	result := Match(src, dst)
	if got := len(result.Pairs); got != 1 {
		t.Fatalf("got %d pairs, want 1", got)
	}

	pair := result.Pairs[0]
	if pair.NumericCast {
		t.Error("expected NumericCast=false for *int -> int32 (narrowing)")
	}
	if !pair.TypeMismatch {
		t.Error("expected TypeMismatch=true for *int -> int32 (narrowing)")
	}
}

// --- Narrowing cast detection tests ---

func TestIsNarrowingCast(t *testing.T) {
	tests := []struct {
		src  string
		dst  string
		want bool
	}{
		// Widening (safe) — same family, dst rank >= src rank.
		{"int8", "int16", false},
		{"int8", "int32", false},
		{"int8", "int64", false},
		{"int16", "int32", false},
		{"int32", "int64", false},
		{"float32", "float64", false},
		{"uint8", "uint16", false},
		{"uint16", "uint32", false},
		{"uint32", "uint64", false},
		{"byte", "uint16", false},

		// Same type — not narrowing.
		{"int32", "int32", false},
		{"float64", "float64", false},

		// Narrowing — same family, dst rank < src rank.
		{"int64", "int32", true},
		{"int32", "int16", true},
		{"int16", "int8", true},
		{"float64", "float32", true},
		{"uint64", "uint32", true},
		{"uint32", "uint16", true},

		// Platform-dependent: int/uint treated as 64-bit.
		{"int", "int32", true},
		{"uint", "uint32", true},
		{"int32", "int", false}, // 32 -> 64 is widening

		// Cross-family — always narrowing.
		{"int32", "uint32", true},  // signed -> unsigned
		{"uint32", "int32", true},  // unsigned -> signed
		{"int64", "float64", true}, // int -> float
		{"float64", "int64", true}, // float -> int
		{"uint8", "int8", true},    // unsigned -> signed
		{"int8", "uint8", true},    // signed -> unsigned

		// rune = int32 (signed family, rank 32).
		{"rune", "int64", false}, // widening
		{"int64", "rune", true},  // narrowing
	}

	for _, tt := range tests {
		t.Run(tt.src+"_to_"+tt.dst, func(t *testing.T) {
			got := IsNarrowingCast(tt.src, tt.dst)
			if got != tt.want {
				t.Errorf("IsNarrowingCast(%q, %q) = %v, want %v", tt.src, tt.dst, got, tt.want)
			}
		})
	}
}

// --- Cross-package matching tests ---

func TestMatchCross_BindTag(t *testing.T) {
	internal := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "MyID", BindName: "ExternalID", TypeStr: "int", ElemType: "int"},
			{Name: "MyName", BindName: "ExternalName", TypeStr: "string", ElemType: "string"},
		},
	}
	external := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "ExternalID", TypeStr: "int", ElemType: "int"},
			{Name: "ExternalName", TypeStr: "string", ElemType: "string"},
		},
	}

	result := MatchCross(internal, external, nil)

	if got := len(result.ToExternal.Pairs); got != 2 {
		t.Fatalf("ToExternal: got %d pairs, want 2", got)
	}
	if got := len(result.FromExternal.Pairs); got != 2 {
		t.Fatalf("FromExternal: got %d pairs, want 2", got)
	}

	// Verify ToExternal direction: src=internal, dst=external.
	for _, p := range result.ToExternal.Pairs {
		if p.Src.Name == "MyID" && p.Dst.Name != "ExternalID" {
			t.Errorf("ToExternal: MyID should map to ExternalID, got %s", p.Dst.Name)
		}
		if p.Src.Name == "MyName" && p.Dst.Name != "ExternalName" {
			t.Errorf("ToExternal: MyName should map to ExternalName, got %s", p.Dst.Name)
		}
	}

	// Verify FromExternal direction: src=external, dst=internal.
	for _, p := range result.FromExternal.Pairs {
		if p.Dst.Name == "MyID" && p.Src.Name != "ExternalID" {
			t.Errorf("FromExternal: MyID should come from ExternalID, got %s", p.Src.Name)
		}
	}
}

func TestMatchCross_BindJSONTag(t *testing.T) {
	internal := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "LocalTitle", BindJSON: "ext_title", TypeStr: "string", ElemType: "string"},
		},
	}
	external := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Title", JSONName: "ext_title", TypeStr: "string", ElemType: "string"},
		},
	}

	result := MatchCross(internal, external, nil)

	if got := len(result.ToExternal.Pairs); got != 1 {
		t.Fatalf("ToExternal: got %d pairs, want 1", got)
	}

	p := result.ToExternal.Pairs[0]
	if p.Src.Name != "LocalTitle" || p.Dst.Name != "Title" {
		t.Errorf("expected LocalTitle->Title, got %s->%s", p.Src.Name, p.Dst.Name)
	}
}

func TestMatchCross_Priority(t *testing.T) {
	// bind tag should take priority over json match and name match.
	internal := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Name", BindName: "FullName", JSONName: "name", TypeStr: "string", ElemType: "string"},
		},
	}
	external := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Name", JSONName: "name", TypeStr: "string", ElemType: "string"},
			{Name: "FullName", JSONName: "full_name", TypeStr: "string", ElemType: "string"},
		},
	}

	result := MatchCross(internal, external, nil)

	if got := len(result.ToExternal.Pairs); got != 1 {
		t.Fatalf("ToExternal: got %d pairs, want 1", got)
	}

	p := result.ToExternal.Pairs[0]
	if p.Dst.Name != "FullName" {
		t.Errorf("bind tag should take priority: got dst=%s, want FullName", p.Dst.Name)
	}
}

func TestMatchCross_BindJSONPriority(t *testing.T) {
	// bind_json should take priority over regular json match.
	internal := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Title", BindJSON: "alt_title", JSONName: "title", TypeStr: "string", ElemType: "string"},
		},
	}
	external := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Title", JSONName: "title", TypeStr: "string", ElemType: "string"},
			{Name: "AltTitle", JSONName: "alt_title", TypeStr: "string", ElemType: "string"},
		},
	}

	result := MatchCross(internal, external, nil)

	if got := len(result.ToExternal.Pairs); got != 1 {
		t.Fatalf("ToExternal: got %d pairs, want 1", got)
	}

	p := result.ToExternal.Pairs[0]
	if p.Dst.Name != "AltTitle" {
		t.Errorf("bind_json should take priority over json: got dst=%s, want AltTitle", p.Dst.Name)
	}
}

func TestMatchCross_FallbackToJSONAndName(t *testing.T) {
	internal := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "Email", JSONName: "email", TypeStr: "string", ElemType: "string"},
			{Name: "Age", TypeStr: "int", ElemType: "int"},
		},
	}
	external := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "ContactEmail", JSONName: "email", TypeStr: "string", ElemType: "string"},
			{Name: "Age", TypeStr: "int", ElemType: "int"},
		},
	}

	result := MatchCross(internal, external, nil)

	if got := len(result.ToExternal.Pairs); got != 2 {
		t.Fatalf("ToExternal: got %d pairs, want 2", got)
	}

	byInternalName := make(map[string]FieldPair)
	for _, p := range result.ToExternal.Pairs {
		byInternalName[p.Src.Name] = p
	}

	// Email should match by JSON tag.
	if p, ok := byInternalName["Email"]; !ok || p.Dst.Name != "ContactEmail" {
		t.Error("Email should match ContactEmail via json tag")
	}

	// Age should match by field name.
	if p, ok := byInternalName["Age"]; !ok || p.Dst.Name != "Age" {
		t.Error("Age should match Age via field name")
	}
}

func TestMatchCross_WithGetters(t *testing.T) {
	internal := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "ID", BindName: "UserId", TypeStr: "int", ElemType: "int"},
			{Name: "Email", JSONName: "email", TypeStr: "string", ElemType: "string"},
		},
	}
	external := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "UserId", TypeStr: "int", ElemType: "int"},
			{Name: "Email", JSONName: "email", TypeStr: "string", ElemType: "string"},
		},
	}

	getters := map[string]loader.GetterInfo{
		"UserId": {MethodName: "GetUserId", FieldName: "UserId", ReturnType: "int"},
		"Email":  {MethodName: "GetEmail", FieldName: "Email", ReturnType: "string"},
	}

	result := MatchCross(internal, external, getters)

	// ToExternal should NOT use getters (writing to external).
	for _, p := range result.ToExternal.Pairs {
		if p.UseGetter {
			t.Errorf("ToExternal pair %s should not use getter", p.Src.Name)
		}
	}

	// FromExternal SHOULD use getters (reading from external).
	for _, p := range result.FromExternal.Pairs {
		if !p.UseGetter {
			t.Errorf("FromExternal pair %s->%s should use getter", p.Src.Name, p.Dst.Name)
		}
	}

	// Verify specific getter names.
	for _, p := range result.FromExternal.Pairs {
		if p.Src.Name == "UserId" && p.GetterName != "GetUserId" {
			t.Errorf("expected GetterName=GetUserId, got %q", p.GetterName)
		}
		if p.Src.Name == "Email" && p.GetterName != "GetEmail" {
			t.Errorf("expected GetterName=GetEmail, got %q", p.GetterName)
		}
	}
}

// --- Ignore and optional tag tests ---

func TestMatch_IgnoreDstField(t *testing.T) {
	src := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "ID", TypeStr: "int", ElemType: "int"},
		},
	}
	dst := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "ID", TypeStr: "int", ElemType: "int"},
			{Name: "Internal", TypeStr: "string", ElemType: "string", Ignore: true},
		},
	}

	result := Match(src, dst)

	// Ignored field must not appear in pairs.
	for _, p := range result.Pairs {
		if p.Dst.Name == "Internal" {
			t.Error("ignored field Internal should not appear in pairs")
		}
	}
	// Ignored field must not appear in unmatched.
	for _, f := range result.Unmatched {
		if f.Name == "Internal" {
			t.Error("ignored field Internal should not appear in unmatched")
		}
	}
	if got := len(result.Pairs); got != 1 {
		t.Errorf("got %d pairs, want 1", got)
	}
	if got := len(result.Unmatched); got != 0 {
		t.Errorf("got %d unmatched, want 0", got)
	}
}

func TestMatch_OptionalUnmatchedDstField(t *testing.T) {
	src := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "ID", TypeStr: "int", ElemType: "int"},
		},
	}
	dst := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "ID", TypeStr: "int", ElemType: "int"},
			{Name: "Extra", TypeStr: "string", ElemType: "string", Optional: true},
		},
	}

	result := Match(src, dst)

	if got := len(result.Pairs); got != 1 {
		t.Errorf("got %d pairs, want 1", got)
	}
	// Optional unmatched field must still appear in unmatched (with Optional=true).
	if got := len(result.Unmatched); got != 1 {
		t.Fatalf("got %d unmatched, want 1", got)
	}
	if !result.Unmatched[0].Optional {
		t.Error("unmatched Extra field should have Optional=true")
	}
	if result.Unmatched[0].Name != "Extra" {
		t.Errorf("expected unmatched field name Extra, got %q", result.Unmatched[0].Name)
	}
}

func TestMatchCross_Unmatched(t *testing.T) {
	internal := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "ID", TypeStr: "int", ElemType: "int"},
			{Name: "NonExistent", TypeStr: "string", ElemType: "string"},
		},
	}
	external := &loader.StructInfo{
		Fields: []loader.StructField{
			{Name: "ID", TypeStr: "int", ElemType: "int"},
		},
	}

	result := MatchCross(internal, external, nil)

	if got := len(result.ToExternal.Pairs); got != 1 {
		t.Errorf("ToExternal: got %d pairs, want 1", got)
	}
	if got := len(result.ToExternal.Unmatched); got != 1 {
		t.Errorf("ToExternal: got %d unmatched, want 1", got)
	}
	if result.ToExternal.Unmatched[0].Name != "NonExistent" {
		t.Errorf("expected NonExistent to be unmatched")
	}
}

// --- Enum / same-underlying-type cast tests ---

func TestClassifyPair_TypeCast_SameUnderlying(t *testing.T) {
	// StatusA string -> StatusB string: same underlying, should TypeCast.
	pair := FieldPair{
		Src: loader.StructField{
			Name:               "Status",
			TypeStr:            "StatusA",
			ElemType:           "StatusA",
			IsNamedNonStruct:   true,
			UnderlyingTypeName: "string",
		},
		Dst: loader.StructField{
			Name:               "Status",
			TypeStr:            "StatusB",
			ElemType:           "StatusB",
			IsNamedNonStruct:   true,
			UnderlyingTypeName: "string",
		},
	}

	classifyPair(&pair)

	if !pair.TypeCast {
		t.Error("expected TypeCast=true for same-underlying named types")
	}
	if pair.CastTypeName != "StatusB" {
		t.Errorf("CastTypeName: got %q, want %q", pair.CastTypeName, "StatusB")
	}
	if pair.TypeMismatch {
		t.Error("expected TypeMismatch=false for same-underlying named types")
	}
	if pair.NumericCast {
		t.Error("expected NumericCast=false for same-underlying named types")
	}
}

func TestClassifyPair_TypeCast_DifferentUnderlying(t *testing.T) {
	// StatusA string -> RoleB int: different underlying, should TypeMismatch.
	pair := FieldPair{
		Src: loader.StructField{
			Name:               "Status",
			TypeStr:            "StatusA",
			ElemType:           "StatusA",
			IsNamedNonStruct:   true,
			UnderlyingTypeName: "string",
		},
		Dst: loader.StructField{
			Name:               "Role",
			TypeStr:            "RoleB",
			ElemType:           "RoleB",
			IsNamedNonStruct:   true,
			UnderlyingTypeName: "int",
		},
	}

	classifyPair(&pair)

	if pair.TypeCast {
		t.Error("expected TypeCast=false for different-underlying named types")
	}
	if !pair.TypeMismatch {
		t.Error("expected TypeMismatch=true for different-underlying named types")
	}
}

func TestClassifyPair_TypeCast_CustomMapperTakesPriority(t *testing.T) {
	// Even with same underlying, if MapperFn is set, ConverterFunc takes priority.
	pair := FieldPair{
		Src: loader.StructField{
			Name:               "Status",
			ElemType:           "StatusA",
			IsNamedNonStruct:   true,
			UnderlyingTypeName: "string",
		},
		Dst: loader.StructField{
			Name:               "Status",
			ElemType:           "StatusB",
			IsNamedNonStruct:   true,
			UnderlyingTypeName: "string",
			MapperFn:           "ConvertStatus",
		},
	}

	classifyPair(&pair)

	if pair.TypeCast {
		t.Error("expected TypeCast=false when MapperFn is set")
	}
	if pair.ConverterFunc != "ConvertStatus" {
		t.Errorf("expected ConverterFunc=ConvertStatus, got %q", pair.ConverterFunc)
	}
}

func TestMatch_EnumNamedTypes(t *testing.T) {
	src := &loader.StructInfo{
		Fields: []loader.StructField{
			{
				Name:               "Status",
				TypeStr:            "StatusA",
				ElemType:           "StatusA",
				JSONName:           "status",
				IsNamedNonStruct:   true,
				UnderlyingTypeName: "string",
			},
			{
				Name:               "Role",
				TypeStr:            "RoleA",
				ElemType:           "RoleA",
				JSONName:           "role",
				IsNamedNonStruct:   true,
				UnderlyingTypeName: "int",
			},
		},
	}
	dst := &loader.StructInfo{
		Fields: []loader.StructField{
			{
				Name:               "Status",
				TypeStr:            "StatusB",
				ElemType:           "StatusB",
				JSONName:           "status",
				IsNamedNonStruct:   true,
				UnderlyingTypeName: "string",
			},
			{
				Name:               "Role",
				TypeStr:            "RoleB",
				ElemType:           "RoleB",
				JSONName:           "role",
				IsNamedNonStruct:   true,
				UnderlyingTypeName: "int",
			},
		},
	}

	result := Match(src, dst)
	if got := len(result.Pairs); got != 2 {
		t.Fatalf("got %d pairs, want 2", got)
	}

	byName := make(map[string]FieldPair)
	for _, p := range result.Pairs {
		byName[p.Dst.Name] = p
	}

	statusPair := byName["Status"]
	if !statusPair.TypeCast {
		t.Error("Status: expected TypeCast=true")
	}
	if statusPair.CastTypeName != "StatusB" {
		t.Errorf("Status: CastTypeName got %q, want %q", statusPair.CastTypeName, "StatusB")
	}

	rolePair := byName["Role"]
	if !rolePair.TypeCast {
		t.Error("Role: expected TypeCast=true")
	}
	if rolePair.CastTypeName != "RoleB" {
		t.Errorf("Role: CastTypeName got %q, want %q", rolePair.CastTypeName, "RoleB")
	}
}
