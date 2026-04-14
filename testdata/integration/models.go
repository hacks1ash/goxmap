// Package integration contains test structs for integration testing of
// the model-mapper code generator.
package integration

// --- Suite 1: Pointer Scenarios ---

// PtrSource has various pointer combinations for testing.
type PtrSource struct {
	Name   *string `json:"name"`
	Age    int     `json:"age"`
	Email  *string `json:"email"`
	Score  float64 `json:"score"`
	Active bool    `json:"active"`
}

// PtrDest tests pointer conversions: *string->string, int->*int, *string->*string.
type PtrDest struct {
	Name   string   `json:"name"`
	Age    *int     `json:"age"`
	Email  *string  `json:"email"`
	Score  *float64 `json:"score"`
	Active bool     `json:"active"`
}

// CustomFuncSource has a field that requires a custom mapper function.
type CustomFuncSource struct {
	ID        int    `json:"id"`
	CreatedAt string `json:"created_at"`
}

// CustomFuncDest uses a mapper func tag on a field.
type CustomFuncDest struct {
	ID        int   `json:"id"`
	CreatedAt int64 `json:"created_at" mapper:"func:ParseTimestamp"`
}

// ParseTimestamp is a custom converter function for testing.
func ParseTimestamp(s string) int64 {
	return 0
}

// --- Suite 2: Deeply Nested / Recursive Mapping ---

// Address is a leaf-level struct.
type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Zip    string `json:"zip"`
}

// AddressDTO is the destination for Address.
type AddressDTO struct {
	Street string `json:"street"`
	City   string `json:"city"`
	Zip    string `json:"zip"`
}

// Employee has an Address.
type Employee struct {
	Name    string  `json:"name"`
	Title   string  `json:"title"`
	Address Address `json:"address"`
}

// EmployeeDTO is the destination for Employee.
type EmployeeDTO struct {
	Name    string     `json:"name"`
	Title   string     `json:"title"`
	Address AddressDTO `json:"address"`
}

// Department has a slice of Employees.
type Department struct {
	Name      string     `json:"name"`
	Employees []Employee `json:"employees"`
}

// DepartmentDTO is the destination for Department.
type DepartmentDTO struct {
	Name      string        `json:"name"`
	Employees []EmployeeDTO `json:"employees"`
}

// Company is the top-level struct with deeply nested hierarchy.
type Company struct {
	Name        string       `json:"name"`
	Departments []Department `json:"departments"`
}

// CompanyDTO is the top-level destination struct.
type CompanyDTO struct {
	Name        string          `json:"name"`
	Departments []DepartmentDTO `json:"departments"`
}

// --- Slice of pointers to non-pointers ---

// UserRef is a simple user struct.
type UserRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// UserRefDTO is a simple user DTO.
type UserRefDTO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TeamWithPtrSlice has a slice of pointer elements.
type TeamWithPtrSlice struct {
	Name    string     `json:"name"`
	Members []*UserRef `json:"members"`
}

// TeamWithValSlice has a slice of value elements.
type TeamWithValSlice struct {
	Name    string       `json:"name"`
	Members []UserRefDTO `json:"members"`
}

// --- Circular Reference ---

// Node is a self-referencing struct for circular dependency testing.
type Node struct {
	ID       int    `json:"id"`
	Value    string `json:"value"`
	Children []Node `json:"children"`
	Parent   *Node  `json:"parent"`
}

// NodeDTO is the destination for the circular Node type.
type NodeDTO struct {
	ID       int       `json:"id"`
	Value    string    `json:"value"`
	Children []NodeDTO `json:"children"`
	Parent   *NodeDTO  `json:"parent"`
}

// --- Existing mapper for reuse testing ---

// MapAddressToAddressDTO is a pre-existing mapper function that should be
// discovered and reused, not re-generated.
func MapAddressToAddressDTO(src Address) AddressDTO {
	return AddressDTO{
		Street: src.Street,
		City:   src.City,
		Zip:    src.Zip,
	}
}
