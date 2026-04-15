package cli

import (
	"strings"
	"unicode"
)

// ToSnakeCase converts a CamelCase string to snake_case.
// Example: "MyStruct" → "my_struct"
func ToSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// pascalCase converts a lowercase package name to PascalCase for use in
// generated function names. Example: "authv1" → "Authv1"
func pascalCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
