// Package benchmarks provides benchmark comparisons between generated mappers
// and reflection-based alternatives (jinzhu/copier, mitchellh/mapstructure).
package benchmarks

// ---------------------------------------------------------------------------
// Level 1 - Simple (flat, ~10 fields)
// ---------------------------------------------------------------------------

// SimpleSource is a flat struct with primitive types and pointer fields.
type SimpleSource struct {
	ID      int
	Name    string
	Email   string
	Age     int
	Active  bool
	Score   float64
	Phone   *string
	Balance *float64
	Country string
	ZipCode string
}

// SimpleDest mirrors SimpleSource for mapping benchmarks.
type SimpleDest struct {
	ID      int
	Name    string
	Email   string
	Age     int
	Active  bool
	Score   float64
	Phone   *string
	Balance *float64
	Country string
	ZipCode string
}

// ---------------------------------------------------------------------------
// Level 2 - Nested (nested structs + slices of strings)
// ---------------------------------------------------------------------------

// Address represents a physical mailing address.
type Address struct {
	Street  string
	City    string
	State   string
	Zip     string
	Country string
}

// NestedSource contains nested structs and string slices.
type NestedSource struct {
	ID      int
	Name    string
	Email   string
	Home    Address
	Work    Address
	Tags    []string
	Aliases []string
}

// NestedDest mirrors NestedSource for mapping benchmarks.
type NestedDest struct {
	ID      int
	Name    string
	Email   string
	Home    Address
	Work    Address
	Tags    []string
	Aliases []string
}

// ---------------------------------------------------------------------------
// Level 3 - Deep & Complex (slices of nested pointers, API response shape)
// ---------------------------------------------------------------------------

// ContactInfo holds contact details with a primary flag.
type ContactInfo struct {
	Email   string
	Phone   *string
	Primary bool
}

// Role represents an authorization role with permissions.
type Role struct {
	ID          int
	Name        string
	Permissions []string
}

// Department groups roles under a named organizational unit.
type Department struct {
	ID      int
	Name    string
	Manager *string
	Roles   []Role
}

// ComplexSource simulates a rich API response payload.
type ComplexSource struct {
	ID          int
	FirstName   string
	LastName    string
	Email       *string
	Age         *int
	Active      bool
	Contacts    []ContactInfo
	Departments []Department
	Address     *Address
	Metadata    map[string]string
	Tags        []string
	Score       *float64
}

// ComplexDest mirrors ComplexSource for mapping benchmarks.
type ComplexDest struct {
	ID          int
	FirstName   string
	LastName    string
	Email       *string
	Age         *int
	Active      bool
	Contacts    []ContactInfo
	Departments []Department
	Address     *Address
	Metadata    map[string]string
	Tags        []string
	Score       *float64
}

// ---------------------------------------------------------------------------
// Level 4 - Numeric Cast (type coercion between numeric types)
// ---------------------------------------------------------------------------

// NumericCastSource uses standard Go int/float types.
type NumericCastSource struct {
	ID        int
	Age       int
	Score     float64
	SmallVal  int8
	BigVal    int64
	Precision float64
	Counter   uint
	Ratio     float32
	PtrAge    *int
	PtrScore  *float64
}

// NumericCastDest uses different numeric types requiring casts.
type NumericCastDest struct {
	ID        int64
	Age       int32
	Score     float32
	SmallVal  int32
	BigVal    int
	Precision float32
	Counter   uint64
	Ratio     float64
	PtrAge    int32
	PtrScore  float32
}
