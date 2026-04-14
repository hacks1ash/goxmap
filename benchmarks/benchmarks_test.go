package benchmarks

import (
	"testing"

	"github.com/jinzhu/copier"
	"github.com/mitchellh/mapstructure"
)

// sink variables prevent the compiler from eliminating benchmark work.
var (
	sinkSimple      SimpleDest
	sinkNested      NestedDest
	sinkComplex     ComplexDest
	sinkNumericCast NumericCastDest
)

// ---------------------------------------------------------------------------
// Test data constructors
// ---------------------------------------------------------------------------

func newSimpleSource() SimpleSource {
	phone := "+1-555-0199"
	balance := 1234.56
	return SimpleSource{
		ID:      42,
		Name:    "Jane Doe",
		Email:   "jane@example.com",
		Age:     30,
		Active:  true,
		Score:   98.6,
		Phone:   &phone,
		Balance: &balance,
		Country: "US",
		ZipCode: "90210",
	}
}

func newNestedSource() NestedSource {
	return NestedSource{
		ID:    1,
		Name:  "John Smith",
		Email: "john@example.com",
		Home: Address{
			Street:  "123 Main St",
			City:    "Springfield",
			State:   "IL",
			Zip:     "62701",
			Country: "US",
		},
		Work: Address{
			Street:  "456 Corporate Blvd",
			City:    "Chicago",
			State:   "IL",
			Zip:     "60601",
			Country: "US",
		},
		Tags:    []string{"admin", "power-user", "beta-tester"},
		Aliases: []string{"jsmith", "john.smith"},
	}
}

func newComplexSource() ComplexSource {
	email := "alice@example.com"
	age := 35
	score := 92.5
	phone1 := "+1-555-0100"
	phone2 := "+1-555-0101"
	mgr1 := "Bob Manager"
	mgr2 := "Carol Director"

	return ComplexSource{
		ID:        100,
		FirstName: "Alice",
		LastName:  "Wonderland",
		Email:     &email,
		Age:       &age,
		Active:    true,
		Contacts: []ContactInfo{
			{Email: "alice.personal@example.com", Phone: &phone1, Primary: true},
			{Email: "alice.work@example.com", Phone: &phone2, Primary: false},
			{Email: "alice.other@example.com", Phone: nil, Primary: false},
		},
		Departments: []Department{
			{
				ID:      10,
				Name:    "Engineering",
				Manager: &mgr1,
				Roles: []Role{
					{ID: 1, Name: "Developer", Permissions: []string{"read", "write", "deploy"}},
					{ID: 2, Name: "Reviewer", Permissions: []string{"read", "review"}},
				},
			},
			{
				ID:      20,
				Name:    "Product",
				Manager: &mgr2,
				Roles: []Role{
					{ID: 3, Name: "PM", Permissions: []string{"read", "write", "plan"}},
				},
			},
		},
		Address: &Address{
			Street:  "789 Wonderland Ave",
			City:    "Fantasy",
			State:   "CA",
			Zip:     "99999",
			Country: "US",
		},
		Metadata: map[string]string{
			"theme":    "dark",
			"lang":     "en",
			"timezone": "America/Los_Angeles",
		},
		Tags:  []string{"vip", "early-adopter", "internal"},
		Score: &score,
	}
}

// ---------------------------------------------------------------------------
// Generated mapper functions (simulate code-gen output: direct field assignment)
// ---------------------------------------------------------------------------

// MapSimple performs direct field-by-field assignment for flat structs.
func MapSimple(src SimpleSource) SimpleDest {
	var dst SimpleDest
	dst.ID = src.ID
	dst.Name = src.Name
	dst.Email = src.Email
	dst.Age = src.Age
	dst.Active = src.Active
	dst.Score = src.Score
	if src.Phone != nil {
		v := *src.Phone
		dst.Phone = &v
	}
	if src.Balance != nil {
		v := *src.Balance
		dst.Balance = &v
	}
	dst.Country = src.Country
	dst.ZipCode = src.ZipCode
	return dst
}

// MapNested performs direct field-by-field assignment for nested structs.
func MapNested(src NestedSource) NestedDest {
	var dst NestedDest
	dst.ID = src.ID
	dst.Name = src.Name
	dst.Email = src.Email
	dst.Home = src.Home
	dst.Work = src.Work
	if src.Tags != nil {
		dst.Tags = make([]string, len(src.Tags))
		copy(dst.Tags, src.Tags)
	}
	if src.Aliases != nil {
		dst.Aliases = make([]string, len(src.Aliases))
		copy(dst.Aliases, src.Aliases)
	}
	return dst
}

// mapContactInfo maps a single ContactInfo value.
func mapContactInfo(src ContactInfo) ContactInfo {
	dst := ContactInfo{
		Email:   src.Email,
		Primary: src.Primary,
	}
	if src.Phone != nil {
		v := *src.Phone
		dst.Phone = &v
	}
	return dst
}

// mapRole maps a single Role value.
func mapRole(src Role) Role {
	dst := Role{
		ID:   src.ID,
		Name: src.Name,
	}
	if src.Permissions != nil {
		dst.Permissions = make([]string, len(src.Permissions))
		copy(dst.Permissions, src.Permissions)
	}
	return dst
}

// mapDepartment maps a single Department value.
func mapDepartment(src Department) Department {
	dst := Department{
		ID:   src.ID,
		Name: src.Name,
	}
	if src.Manager != nil {
		v := *src.Manager
		dst.Manager = &v
	}
	if src.Roles != nil {
		dst.Roles = make([]Role, len(src.Roles))
		for i, r := range src.Roles {
			dst.Roles[i] = mapRole(r)
		}
	}
	return dst
}

// mapAddress maps a single Address value.
func mapAddress(src Address) Address {
	return Address{
		Street:  src.Street,
		City:    src.City,
		State:   src.State,
		Zip:     src.Zip,
		Country: src.Country,
	}
}

// MapComplex performs direct field-by-field assignment for deeply nested structs.
func MapComplex(src ComplexSource) ComplexDest {
	var dst ComplexDest
	dst.ID = src.ID
	dst.FirstName = src.FirstName
	dst.LastName = src.LastName
	if src.Email != nil {
		v := *src.Email
		dst.Email = &v
	}
	if src.Age != nil {
		v := *src.Age
		dst.Age = &v
	}
	dst.Active = src.Active

	if src.Contacts != nil {
		dst.Contacts = make([]ContactInfo, len(src.Contacts))
		for i, c := range src.Contacts {
			dst.Contacts[i] = mapContactInfo(c)
		}
	}

	if src.Departments != nil {
		dst.Departments = make([]Department, len(src.Departments))
		for i, d := range src.Departments {
			dst.Departments[i] = mapDepartment(d)
		}
	}

	if src.Address != nil {
		a := mapAddress(*src.Address)
		dst.Address = &a
	}

	if src.Metadata != nil {
		dst.Metadata = make(map[string]string, len(src.Metadata))
		for k, v := range src.Metadata {
			dst.Metadata[k] = v
		}
	}

	if src.Tags != nil {
		dst.Tags = make([]string, len(src.Tags))
		copy(dst.Tags, src.Tags)
	}

	if src.Score != nil {
		v := *src.Score
		dst.Score = &v
	}

	return dst
}

// ---------------------------------------------------------------------------
// Level 1 - Simple benchmarks
// ---------------------------------------------------------------------------

func BenchmarkSimple_Generated(b *testing.B) {
	b.ReportAllocs()
	src := newSimpleSource()
	b.ResetTimer()
	for b.Loop() {
		sinkSimple = MapSimple(src)
	}
}

func BenchmarkSimple_Copier(b *testing.B) {
	b.ReportAllocs()
	src := newSimpleSource()
	b.ResetTimer()
	for b.Loop() {
		var dst SimpleDest
		_ = copier.Copy(&dst, &src)
		sinkSimple = dst
	}
}

func BenchmarkSimple_Mapstructure(b *testing.B) {
	b.ReportAllocs()
	src := newSimpleSource()
	b.ResetTimer()
	for b.Loop() {
		var dst SimpleDest
		_ = mapstructure.Decode(src, &dst)
		sinkSimple = dst
	}
}

// ---------------------------------------------------------------------------
// Level 2 - Nested benchmarks
// ---------------------------------------------------------------------------

func BenchmarkNested_Generated(b *testing.B) {
	b.ReportAllocs()
	src := newNestedSource()
	b.ResetTimer()
	for b.Loop() {
		sinkNested = MapNested(src)
	}
}

func BenchmarkNested_Copier(b *testing.B) {
	b.ReportAllocs()
	src := newNestedSource()
	b.ResetTimer()
	for b.Loop() {
		var dst NestedDest
		_ = copier.Copy(&dst, &src)
		sinkNested = dst
	}
}

func BenchmarkNested_Mapstructure(b *testing.B) {
	b.ReportAllocs()
	src := newNestedSource()
	b.ResetTimer()
	for b.Loop() {
		var dst NestedDest
		_ = mapstructure.Decode(src, &dst)
		sinkNested = dst
	}
}

// ---------------------------------------------------------------------------
// Level 3 - Complex benchmarks
// ---------------------------------------------------------------------------

func BenchmarkComplex_Generated(b *testing.B) {
	b.ReportAllocs()
	src := newComplexSource()
	b.ResetTimer()
	for b.Loop() {
		sinkComplex = MapComplex(src)
	}
}

func BenchmarkComplex_Copier(b *testing.B) {
	b.ReportAllocs()
	src := newComplexSource()
	b.ResetTimer()
	for b.Loop() {
		var dst ComplexDest
		_ = copier.Copy(&dst, &src)
		sinkComplex = dst
	}
}

func BenchmarkComplex_Mapstructure(b *testing.B) {
	b.ReportAllocs()
	src := newComplexSource()
	b.ResetTimer()
	for b.Loop() {
		var dst ComplexDest
		_ = mapstructure.Decode(src, &dst)
		sinkComplex = dst
	}
}

// ---------------------------------------------------------------------------
// Level 4 - Numeric Cast benchmarks
// ---------------------------------------------------------------------------

func newNumericCastSource() NumericCastSource {
	age := 30
	score := 98.6
	return NumericCastSource{
		ID:        42,
		Age:       30,
		Score:     98.6,
		SmallVal:  127,
		BigVal:    9999999999,
		Precision: 3.141592653589793,
		Counter:   1000,
		Ratio:     0.75,
		PtrAge:    &age,
		PtrScore:  &score,
	}
}

// MapNumericCast simulates generated code with inline numeric type casts.
func MapNumericCast(src NumericCastSource) NumericCastDest {
	var dst NumericCastDest
	dst.ID = int64(src.ID)
	dst.Age = int32(src.Age)
	dst.Score = float32(src.Score)
	dst.SmallVal = int32(src.SmallVal)
	dst.BigVal = int(src.BigVal)
	dst.Precision = float32(src.Precision)
	dst.Counter = uint64(src.Counter)
	dst.Ratio = float64(src.Ratio)
	if src.PtrAge != nil {
		dst.PtrAge = int32(*src.PtrAge)
	}
	if src.PtrScore != nil {
		dst.PtrScore = float32(*src.PtrScore)
	}
	return dst
}

func BenchmarkNumericCast_Generated(b *testing.B) {
	b.ReportAllocs()
	src := newNumericCastSource()
	b.ResetTimer()
	for b.Loop() {
		sinkNumericCast = MapNumericCast(src)
	}
}

func BenchmarkNumericCast_Copier(b *testing.B) {
	b.ReportAllocs()
	src := newNumericCastSource()
	b.ResetTimer()
	for b.Loop() {
		var dst NumericCastDest
		_ = copier.Copy(&dst, &src)
		sinkNumericCast = dst
	}
}

func BenchmarkNumericCast_Mapstructure(b *testing.B) {
	b.ReportAllocs()
	src := newNumericCastSource()
	b.ResetTimer()
	for b.Loop() {
		var dst NumericCastDest
		_ = mapstructure.Decode(src, &dst)
		sinkNumericCast = dst
	}
}
