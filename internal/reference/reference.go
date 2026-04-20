// Package reference parses qualified struct references for goxmap.
package reference

import "fmt"

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
	return Ref{Kind: KindBare, TypeName: s}, nil
}
