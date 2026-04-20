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

// SamePackageOptions holds all parameters required for same-package mapping generation.
type SamePackageOptions struct {
	WorkDir    string
	SrcType    string
	DstType    string
	FuncName   string
	Output     string
	StructFunc string
	Bidi       bool
}

// RunSamePackage handles same-package mapping generation.
func RunSamePackage(opts SamePackageOptions) {
	if opts.FuncName == "" {
		opts.FuncName = "Map" + opts.SrcType + "To" + opts.DstType
	}

	pctx, err := loader.LoadPackage(opts.WorkDir)
	if err != nil {
		log.Fatalf("loading package: %v", err)
	}

	srcInfo, err := loader.LoadStructFromPkg(pctx, opts.SrcType)
	if err != nil {
		log.Fatalf("loading source type %s: %v", opts.SrcType, err)
	}

	dstInfo, err := loader.LoadStructFromPkg(pctx, opts.DstType)
	if err != nil {
		log.Fatalf("loading destination type %s: %v", opts.DstType, err)
	}

	structMapper := opts.StructFunc
	if structMapper == "" {
		structMapper = dstInfo.MapperFunc
	}

	result := matcher.Match(srcInfo, dstInfo, matcher.MatchOptions{})

	if len(result.Unmatched) > 0 && structMapper == "" {
		for _, f := range result.Unmatched {
			if f.Optional {
				continue
			}
			fmt.Fprintf(os.Stderr, "goxmap: warning: destination field %s.%s has no matching source field (skipped, will use zero value)\n",
				opts.DstType, f.Name)
		}
	}

	// Resolve type mismatches: try auto-discovery of converter functions.
	ResolveTypeMismatches(pctx, result.Pairs, opts.SrcType)

	existingMappers := loader.DiscoverMapperFuncs(pctx)

	rootCfg := generator.Config{
		PackageName:    dstInfo.PackageName,
		FuncName:       opts.FuncName,
		SrcType:        opts.SrcType,
		DstType:        opts.DstType,
		Pairs:          result.Pairs,
		StructMapperFn: structMapper,
	}

	mcfg := generator.MultiConfig{
		PackageName:     dstInfo.PackageName,
		RootFunc:        rootCfg,
		ExistingMappers: existingMappers,
		PkgContext:      pctx,
	}

	// Bidirectional same-package mode.
	if opts.Bidi {
		revFuncName := "Map" + opts.DstType + "To" + opts.SrcType
		revResult := matcher.Match(dstInfo, srcInfo, matcher.MatchOptions{})

		if len(revResult.Unmatched) > 0 {
			for _, f := range revResult.Unmatched {
				if f.Optional {
					continue
				}
				fmt.Fprintf(os.Stderr, "goxmap: warning: reverse mapper: destination field %s.%s has no matching source field (skipped, will use zero value)\n",
					opts.SrcType, f.Name)
			}
		}

		ResolveTypeMismatches(pctx, revResult.Pairs, opts.DstType)

		revCfg := generator.Config{
			PackageName: srcInfo.PackageName,
			FuncName:    revFuncName,
			SrcType:     opts.DstType,
			DstType:     opts.SrcType,
			Pairs:       revResult.Pairs,
		}
		mcfg.Bidirectional = true
		mcfg.ReverseFunc = &revCfg
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		log.Fatalf("generating code: %v", err)
	}

	outPath := filepath.Join(opts.WorkDir, opts.Output)
	if err := os.WriteFile(outPath, code, 0644); err != nil {
		log.Fatalf("writing output file %s: %v", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "goxmap: wrote %s\n", outPath)
}
