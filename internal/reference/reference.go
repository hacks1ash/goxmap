// Package reference parses qualified struct references for goxmap.
package reference

import (
	"fmt"
	"strings"
)

type Kind int

const (
	KindBare Kind = iota
	KindModuleRelative
	KindFullPath
)

type Ref struct {
	Kind        Kind
	PackagePath string // empty when Kind == KindBare
	TypeName    string
}

func Parse(s string) (Ref, error) {
	if s == "" {
		return Ref{}, fmt.Errorf("empty reference")
	}

	// No slash -> bare type in current package.
	if !strings.ContainsRune(s, '/') {
		if strings.ContainsRune(s, '.') {
			return Ref{}, fmt.Errorf("bare reference %q must not contain '.'", s)
		}
		return Ref{Kind: KindBare, TypeName: s}, nil
	}

	// Qualified: split on the final '.'.
	dot := strings.LastIndex(s, ".")
	if dot < 0 {
		return Ref{}, fmt.Errorf("qualified reference %q must contain '.' separating type from package", s)
	}
	pkg := s[:dot]
	typeName := s[dot+1:]
	if pkg == "" {
		return Ref{}, fmt.Errorf("reference %q has empty package path", s)
	}
	if typeName == "" {
		return Ref{}, fmt.Errorf("reference %q has empty type name", s)
	}
	if strings.ContainsRune(typeName, '/') || strings.ContainsRune(typeName, '.') {
		return Ref{}, fmt.Errorf("reference %q has invalid type name %q", s, typeName)
	}

	firstSeg := pkg
	if i := strings.IndexRune(pkg, '/'); i >= 0 {
		firstSeg = pkg[:i]
	}
	if strings.ContainsRune(firstSeg, '.') {
		return Ref{Kind: KindFullPath, PackagePath: pkg, TypeName: typeName}, nil
	}
	return Ref{Kind: KindModuleRelative, PackagePath: pkg, TypeName: typeName}, nil
}
