package testdata

// Source is a basic source struct for testing.
type Source struct {
	ID        int     `json:"id"`
	FirstName string  `json:"first_name"`
	Email     *string `json:"email"`
	Age       int
	hidden    string //nolint:unused // unexported, should be skipped
}

// Destination is a basic destination struct with mapper tags.
type Destination struct {
	_         struct{} `mapper:"struct_func:CustomMap"`
	ID        int      `json:"id"`
	FirstName string   `json:"first_name"`
	Email     string   `json:"email"`
	Age       *int
	FullName  string `mapper:"func:BuildFullName"`
}

// --- Nested struct types for testing ---

// Address is a source address struct.
type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Zip    string `json:"zip"`
}

// AddressDTO is a destination address struct.
type AddressDTO struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Zip    string `json:"zip"`
}

// EmailInfo is a source email struct.
type EmailInfo struct {
	Address string `json:"address"`
	Primary bool   `json:"primary"`
}

// EmailInfoDTO is a destination email struct.
type EmailInfoDTO struct {
	Address string `json:"address"`
	Primary bool   `json:"primary"`
}

// User is a source struct with nested struct and slice fields.
type User struct {
	ID      int         `json:"id"`
	Name    string      `json:"name"`
	Address Address     `json:"address"`
	Emails  []EmailInfo `json:"emails"`
}

// UserDTO is a destination struct with nested struct and slice fields.
type UserDTO struct {
	ID      int            `json:"id"`
	Name    string         `json:"name"`
	Address AddressDTO     `json:"address"`
	Emails  []EmailInfoDTO `json:"emails"`
}

// UserWithPtr has a pointer to nested struct.
type UserWithPtr struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Address *Address `json:"address"`
}

// UserWithPtrDTO has a non-pointer nested struct.
type UserWithPtrDTO struct {
	ID      int        `json:"id"`
	Name    string     `json:"name"`
	Address AddressDTO `json:"address"`
}

// --- Circular reference types for testing ---

// Parent references children.
type Parent struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Children []Child `json:"children"`
}

// Child references parent.
type Child struct {
	ID     int     `json:"id"`
	Name   string  `json:"name"`
	Parent *Parent `json:"parent"`
}

// ParentDTO is the destination for Parent.
type ParentDTO struct {
	ID       int        `json:"id"`
	Name     string     `json:"name"`
	Children []ChildDTO `json:"children"`
}

// ChildDTO is the destination for Child.
type ChildDTO struct {
	ID     int        `json:"id"`
	Name   string     `json:"name"`
	Parent *ParentDTO `json:"parent"`
}

// --- Existing mapper function for discovery testing ---

// MapAddressToAddressDTO is a manually written mapper that should be discovered.
func MapAddressToAddressDTO(src Address) AddressDTO {
	return AddressDTO{
		Street: src.Street,
		City:   src.City,
		Zip:    src.Zip,
	}
}

// --- Cross-package binding types for testing ---

// InternalUser uses bind and bind_json tags to map to an external struct.
type InternalUser struct {
	ID       int    `json:"id" mapper:"bind:UserId"`
	FullName string `json:"full_name" mapper:"bind_json:display_name"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
}

// ExternalUser simulates an external (e.g., Protobuf-generated) struct.
type ExternalUser struct {
	UserId      int    `json:"user_id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Age         int    `json:"age"`
}

// GetUserId is a Protobuf-style getter for ExternalUser.
func (e *ExternalUser) GetUserId() int {
	if e != nil {
		return e.UserId
	}
	return 0
}

// GetDisplayName is a Protobuf-style getter for ExternalUser.
func (e *ExternalUser) GetDisplayName() string {
	if e != nil {
		return e.DisplayName
	}
	return ""
}

// GetEmail is a Protobuf-style getter for ExternalUser.
func (e *ExternalUser) GetEmail() string {
	if e != nil {
		return e.Email
	}
	return ""
}

// GetAge is a Protobuf-style getter for ExternalUser.
func (e *ExternalUser) GetAge() int {
	if e != nil {
		return e.Age
	}
	return 0
}

// --- Bind tag only types ---

// BindOnlyInternal uses only bind tags.
type BindOnlyInternal struct {
	MyID   int    `mapper:"bind:ExternalID"`
	MyName string `mapper:"bind:ExternalName"`
}

// BindOnlyExternal is the target for bind-only matching.
type BindOnlyExternal struct {
	ExternalID   int
	ExternalName string
}

// --- Bind JSON only types ---

// BindJSONInternal uses only bind_json tags.
type BindJSONInternal struct {
	LocalTitle string `mapper:"bind_json:ext_title"`
	LocalCount int    `mapper:"bind_json:ext_count"`
}

// BindJSONExternal is the target for bind_json matching.
type BindJSONExternal struct {
	Title string `json:"ext_title"`
	Count int    `json:"ext_count"`
}

// --- Pointer conversion cross-package types ---

// InternalWithPtr has pointer fields for cross-package pointer conversion testing.
type InternalWithPtr struct {
	Name  *string `mapper:"bind:FullName"`
	Value int     `mapper:"bind:Amount"`
}

// ExternalWithPtr has different pointer-ness for testing conversions.
type ExternalWithPtr struct {
	FullName string
	Amount   *int
}

// --- Numeric type mismatch types for testing ---

// NumericSource has various numeric types.
type NumericSource struct {
	Age      int     `json:"age"`
	Score    float32 `json:"score"`
	SmallID  int64   `json:"small_id"`
	PtrCount *int    `json:"ptr_count"`
}

// NumericDest has different numeric types to test coercion.
type NumericDest struct {
	Age      int32   `json:"age"`
	Score    float64 `json:"score"`
	SmallID  int32   `json:"small_id"`
	PtrCount int32   `json:"ptr_count"`
}

// --- Converter function auto-discovery types for testing ---

// ConverterSource has a custom type field.
type ConverterSource struct {
	ID        int    `json:"id"`
	Timestamp string `json:"timestamp"`
}

// ConverterDest expects a custom converter for Timestamp.
type ConverterDest struct {
	ID        int   `json:"id"`
	Timestamp int64 `json:"timestamp"`
}

// MapStringToInt64 is a converter function following the naming convention.
func MapStringToInt64(v string) int64 {
	return 0
}

// --- Bidirectional same-package types for testing ---

// BidiSource is a source struct for bidirectional mapping tests.
type BidiSource struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// BidiDest is a destination struct for bidirectional mapping tests.
type BidiDest struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// --- Ignore/optional tag types for testing ---

// IgnoreOptSource for testing ignore/optional tags.
type IgnoreOptSource struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// IgnoreOptDest has ignore and optional fields.
type IgnoreOptDest struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Internal string `mapper:"ignore"`
	Extra    string `mapper:"optional"`
}

// --- Enum/named type mapping test types ---

// StatusA is a string enum type.
type StatusA string

// StatusB is a different string enum type with same underlying.
type StatusB string

// RoleA is an int enum type.
type RoleA int

// RoleB is a different int enum type with same underlying.
type RoleB int

// EnumSource has named-type fields.
type EnumSource struct {
	Status StatusA `json:"status"`
	Role   RoleA   `json:"role"`
}

// EnumDest has different named types with same underlying.
type EnumDest struct {
	Status StatusB `json:"status"`
	Role   RoleB   `json:"role"`
}

// --- Slice with pointer-to-struct elements for coverage ---

// SlicePtrElemSource has a slice of pointer-to-struct elements.
type SlicePtrElemSource struct {
	Items []*EmailInfo `json:"items"`
}

// --- JSON tag edge case ---

// JSONDashSource has a field with json:"-" tag (should be ignored by json).
type JSONDashSource struct {
	ID       int    `json:"id"`
	Internal string `json:"-"`
}

// --- Bad converter function signatures for negative testing ---

// NotAFunc_MapBadToString is intentionally a variable, not a function.
var NotAFunc_MapBadToString = "not a func"

// MapTwoParamToString has wrong signature (2 params instead of 1).
func MapTwoParamToString(a, b string) string { return a }

// MapTwoReturnToString has wrong signature (2 returns instead of 1).
func MapTwoReturnToString(a string) (string, error) { return a, nil }
