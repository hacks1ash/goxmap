// Package cli provides helper functions for the goxmap CLI tool.
package cli

import (
	"log"
	"os"
	"path/filepath"

	"github.com/hacks1ash/goxmap/internal/loader"
	"github.com/hacks1ash/goxmap/internal/matcher"
)

// ResolveTypeMismatches iterates over pairs and resolves type mismatches
// by looking for converter functions in the package scope.
func ResolveTypeMismatches(pctx *loader.PackageContext, pairs []matcher.FieldPair, parentType string) {
	for i := range pairs {
		pair := &pairs[i]
		if !pair.TypeMismatch {
			continue
		}

		// mapper:"func:..." tag takes priority over auto-discovery.
		if pair.Dst.MapperFn != "" {
			pair.ConverterFunc = pair.Dst.MapperFn
			pair.TypeMismatch = false
			continue
		}

		// Auto-discover converter function.
		fn := loader.FindConverterFunc(pctx, pair.Src.ElemType, pair.Dst.ElemType)
		if fn != "" {
			pair.ConverterFunc = fn
			pair.TypeMismatch = false
			continue
		}

		// No converter found -- fatal error with context-specific message.
		expectedFn := loader.ConverterFuncName(pair.Src.ElemType, pair.Dst.ElemType)
		fieldRef := parentType + "." + pair.Dst.Name

		if matcher.IsNumericType(pair.Src.ElemType) && matcher.IsNumericType(pair.Dst.ElemType) {
			log.Fatalf("error: field %s has narrowing conversion: %s -> %s.\n"+
				"  This may lose data. To allow this, add a converter function:\n"+
				"  func %s(v %s) %s { return %s(v) }",
				fieldRef,
				pair.Src.ElemType, pair.Dst.ElemType,
				expectedFn, pair.Src.ElemType, pair.Dst.ElemType,
				pair.Dst.ElemType,
			)
		}

		log.Fatalf("error: field %s has type mismatch: %s -> %s.\n"+
			"  No converter function %s found in package.\n"+
			"  Add a function: func %s(v %s) %s { ... }",
			fieldRef,
			pair.Src.ElemType, pair.Dst.ElemType,
			expectedFn, expectedFn,
			pair.Src.ElemType, pair.Dst.ElemType,
		)
	}
}

// ResolveDir resolves the working directory from an explicit path, the $GOFILE
// environment variable, or the current working directory (in that order).
func ResolveDir(explicit string) string {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			log.Fatalf("resolving directory %s: %v", explicit, err)
		}
		return abs
	}

	if gofile := os.Getenv("GOFILE"); gofile != "" {
		return filepath.Dir(gofile)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("cannot determine working directory")
	}
	return dir
}
