package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hacks1ash/goxmap/internal/generator"
	"github.com/hacks1ash/goxmap/internal/loader"
	"github.com/hacks1ash/goxmap/internal/matcher"
)

// CrossPackageOptions holds all parameters required for cross-package mapping generation.
type CrossPackageOptions struct {
	WorkDir      string
	InternalType string
	ExternalType string
	FuncName     string
	Output       string
	ExtPkgPath   string
	Bidi         bool
}

// RunCrossPackage handles cross-package mapping generation.
// The InternalType refers to the local struct; ExternalType refers to the external struct.
func RunCrossPackage(opts CrossPackageOptions) {
	// Load the local package.
	localPctx, err := loader.LoadPackage(opts.WorkDir)
	if err != nil {
		log.Fatalf("loading local package: %v", err)
	}

	// Load the external package.
	extPctx, err := loader.LoadExternalPackage(opts.WorkDir, opts.ExtPkgPath)
	if err != nil {
		log.Fatalf("loading external package %s: %v", opts.ExtPkgPath, err)
	}

	// Load internal struct.
	internalInfo, err := loader.LoadStructFromPkg(localPctx, opts.InternalType)
	if err != nil {
		log.Fatalf("loading internal type %s: %v", opts.InternalType, err)
	}

	// Load external struct.
	externalInfo, err := loader.LoadStructFromPkg(extPctx, opts.ExternalType)
	if err != nil {
		log.Fatalf("loading external type %s: %v", opts.ExternalType, err)
	}

	// Discover getters on the external type (for Protobuf support).
	getters := loader.DiscoverGetters(extPctx, opts.ExternalType)

	// Perform cross-package matching.
	crossResult := matcher.MatchCross(internalInfo, externalInfo, getters)

	// Warn about unmatched fields.
	for _, f := range crossResult.ToExternal.Unmatched {
		if f.Optional {
			continue
		}
		fmt.Fprintf(os.Stderr, "goxmap: warning: field %s.%s has no matching external field\n",
			opts.InternalType, f.Name)
	}

	// Resolve type mismatches (including narrowing numeric casts).
	ResolveTypeMismatches(localPctx, crossResult.ToExternal.Pairs, opts.InternalType)
	ResolveTypeMismatches(localPctx, crossResult.FromExternal.Pairs, opts.ExternalType)

	// Determine function names.
	toExtFn := opts.FuncName
	if toExtFn == "" {
		toExtFn = "Map" + opts.InternalType + "To" + opts.ExternalType
	}
	fromExtFn := "Map" + opts.ExternalType + "To" + opts.InternalType

	ccfg := generator.CrossConfig{
		PackageName:          internalInfo.PackageName,
		InternalType:         opts.InternalType,
		ExternalType:         opts.ExternalType,
		ExternalPkgName:      externalInfo.PackageName,
		ExternalPkgPath:      opts.ExtPkgPath,
		ToExternalFuncName:   toExtFn,
		FromExternalFuncName: fromExtFn,
		ToExternalPairs:      crossResult.ToExternal.Pairs,
		FromExternalPairs:    crossResult.FromExternal.Pairs,
		Bidirectional:        opts.Bidi,
	}

	code, err := generator.GenerateCross(ccfg)
	if err != nil {
		log.Fatalf("generating cross-package code: %v", err)
	}

	outPath := filepath.Join(opts.WorkDir, opts.Output)
	if err := os.WriteFile(outPath, code, 0644); err != nil {
		log.Fatalf("writing output file %s: %v", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "goxmap: wrote %s\n", outPath)
}
