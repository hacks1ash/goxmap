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
