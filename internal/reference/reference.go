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

	// No slash — either bare ("User") or short module-relative ("sub.Type").
	if !strings.ContainsRune(s, '/') {
		dot := strings.Index(s, ".")
		if dot < 0 {
			// Plain bare reference: no package qualifier.
			return Ref{Kind: KindBare, TypeName: s}, nil
		}
		// Short form: single-segment sub-package, e.g. "b.UserDTO".
		pkg := s[:dot]
		typeName := s[dot+1:]
		if pkg == "" {
			return Ref{}, fmt.Errorf("reference %q has empty package path", s)
		}
		if typeName == "" {
			return Ref{}, fmt.Errorf("reference %q has empty type name", s)
		}
		if strings.ContainsRune(typeName, '.') {
			return Ref{}, fmt.Errorf("reference %q has invalid type name %q", s, typeName)
		}
		return Ref{Kind: KindModuleRelative, PackagePath: pkg, TypeName: typeName}, nil
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

// ImportPath returns the absolute import path for the reference, or "" for
// bare references (which resolve via the working directory's package).
// modulePath is the module path of the working directory's module.
func (r Ref) ImportPath(modulePath string) string {
	switch r.Kind {
	case KindBare:
		return ""
	case KindFullPath:
		return r.PackagePath
	case KindModuleRelative:
		if modulePath == "" {
			return r.PackagePath
		}
		return modulePath + "/" + r.PackagePath
	}
	return ""
}
