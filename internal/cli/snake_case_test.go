package cli

import "testing"

func TestPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"authv1", "Authv1"},
		{"extpkg", "Extpkg"},
		{"proto", "Proto"},
		{"a", "A"},
		{"", ""},
		{"Alreadyupper", "Alreadyupper"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := pascalCase(tc.input)
			if got != tc.want {
				t.Errorf("pascalCase(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple camel case", "MyStruct", "my_struct"},
		{"single word", "User", "user"},
		{"acronym DTO", "UserDTO", "user_d_t_o"},
		{"single uppercase char", "A", "a"},
		{"single lowercase char", "a", "a"},
		{"empty string", "", ""},
		{"mixed lower with uppercase", "alreadyLower", "already_lower"},
		{"consecutive uppercase", "ABCDef", "a_b_c_def"},
		{"with numbers", "myField123", "my_field123"},
		{"acronym prefix", "XMLParser", "x_m_l_parser"},
		{"single uppercase X", "X", "x"},
		{"three consecutive uppercase", "ABCd", "a_b_cd"},
		{"lowercase only", "foo", "foo"},
		{"two words", "FooBar", "foo_bar"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ToSnakeCase(tc.input)
			if got != tc.want {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
