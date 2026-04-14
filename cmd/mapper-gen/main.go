// Command mapper-gen generates struct-to-struct mapping functions.
//
// Usage:
//
//	//go:generate go run github.com/hacks1ash/goxmap/cmd/mapper-gen -src SourceType -dst DestType
//
// Flags:
//
//	-src           Source struct type name (required)
//	-dst           Destination struct type name (required)
//	-func          Generated function name (default: Map<Src>To<Dst>)
//	-dir           Package directory to load types from (default: current directory from $GOFILE or ".")
//	-output        Output file name (default: <dst_lowercase>_mapper_gen.go)
//	-struct-func   Custom function to delegate entire struct mapping
//	-external-pkg  Import path of external package for cross-package mapping
//	-bidi          Generate bidirectional mapper functions
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/hacks1ash/goxmap/internal/generator"
	"github.com/hacks1ash/goxmap/internal/loader"
	"github.com/hacks1ash/goxmap/internal/matcher"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("mapper-gen: ")

	var (
		srcType     = flag.String("src", "", "source struct type name (required)")
		dstType     = flag.String("dst", "", "destination struct type name (required)")
		funcName    = flag.String("func", "", "generated function name (default: Map<Src>To<Dst>)")
		dir         = flag.String("dir", "", "package directory (default: $GOFILE dir or \".\")")
		output      = flag.String("output", "", "output file name (default: <dst_lowercase>_mapper_gen.go)")
		structFunc  = flag.String("struct-func", "", "delegate entire mapping to this function")
		externalPkg = flag.String("external-pkg", "", "import path of external package for cross-package mapping")
		bidi        = flag.Bool("bidi", false, "generate bidirectional mapper functions")
	)

	flag.Parse()

	if *srcType == "" || *dstType == "" {
		log.Fatal("both -src and -dst flags are required")
	}

	workDir := resolveDir(*dir)

	if *output == "" {
		*output = toSnakeCase(*dstType) + "_mapper_gen.go"
	}

	// Cross-package mode.
	if *externalPkg != "" {
		runCrossPackage(workDir, *srcType, *dstType, *funcName, *output, *externalPkg, *bidi)
		return
	}

	// Same-package mode (original behavior).
	if *funcName == "" {
		*funcName = "Map" + *srcType + "To" + *dstType
	}

	pctx, err := loader.LoadPackage(workDir)
	if err != nil {
		log.Fatalf("loading package: %v", err)
	}

	srcInfo, err := loader.LoadStructFromPkg(pctx, *srcType)
	if err != nil {
		log.Fatalf("loading source type %s: %v", *srcType, err)
	}

	dstInfo, err := loader.LoadStructFromPkg(pctx, *dstType)
	if err != nil {
		log.Fatalf("loading destination type %s: %v", *dstType, err)
	}

	structMapper := *structFunc
	if structMapper == "" {
		structMapper = dstInfo.MapperFunc
	}

	result := matcher.Match(srcInfo, dstInfo)

	if len(result.Unmatched) > 0 && structMapper == "" {
		for _, f := range result.Unmatched {
			fmt.Fprintf(os.Stderr, "mapper-gen: warning: destination field %s.%s has no matching source field (skipped, will use zero value)\n",
				*dstType, f.Name)
		}
	}

	// Resolve type mismatches: try auto-discovery of converter functions.
	resolveTypeMismatches(pctx, result.Pairs, *srcType)

	existingMappers := loader.DiscoverMapperFuncs(pctx)

	rootCfg := generator.Config{
		PackageName:    dstInfo.PackageName,
		FuncName:       *funcName,
		SrcType:        *srcType,
		DstType:        *dstType,
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
	if *bidi {
		revFuncName := "Map" + *dstType + "To" + *srcType
		revResult := matcher.Match(dstInfo, srcInfo)

		if len(revResult.Unmatched) > 0 {
			for _, f := range revResult.Unmatched {
				fmt.Fprintf(os.Stderr, "mapper-gen: warning: reverse mapper: destination field %s.%s has no matching source field (skipped, will use zero value)\n",
					*srcType, f.Name)
			}
		}

		resolveTypeMismatches(pctx, revResult.Pairs, *dstType)

		revCfg := generator.Config{
			PackageName: srcInfo.PackageName,
			FuncName:    revFuncName,
			SrcType:     *dstType,
			DstType:     *srcType,
			Pairs:       revResult.Pairs,
		}
		mcfg.Bidirectional = true
		mcfg.ReverseFunc = &revCfg
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		log.Fatalf("generating code: %v", err)
	}

	outPath := filepath.Join(workDir, *output)
	if err := os.WriteFile(outPath, code, 0644); err != nil {
		log.Fatalf("writing output file %s: %v", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "mapper-gen: wrote %s\n", outPath)
}

// resolveTypeMismatches iterates over pairs and resolves type mismatches
// by looking for converter functions in the package scope.
func resolveTypeMismatches(pctx *loader.PackageContext, pairs []matcher.FieldPair, parentType string) {
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

		// No converter found -- fatal error.
		expectedFn := loader.ConverterFuncName(pair.Src.ElemType, pair.Dst.ElemType)
		log.Fatalf("error: field %s has type mismatch: %s -> %s.\n"+
			"  No converter function %s found in package.\n"+
			"  Add a function: func %s(v %s) %s { ... }",
			parentType+"."+pair.Dst.Name,
			pair.Src.ElemType, pair.Dst.ElemType,
			expectedFn, expectedFn,
			pair.Src.ElemType, pair.Dst.ElemType,
		)
	}
}

// runCrossPackage handles cross-package mapping generation.
// The -src flag refers to the local (internal) struct type.
// The -dst flag refers to the external struct type.
func runCrossPackage(workDir, internalType, externalType, funcName, output, extPkgPath string, bidi bool) {
	// Load the local package.
	localPctx, err := loader.LoadPackage(workDir)
	if err != nil {
		log.Fatalf("loading local package: %v", err)
	}

	// Load the external package.
	extPctx, err := loader.LoadExternalPackage(workDir, extPkgPath)
	if err != nil {
		log.Fatalf("loading external package %s: %v", extPkgPath, err)
	}

	// Load internal struct.
	internalInfo, err := loader.LoadStructFromPkg(localPctx, internalType)
	if err != nil {
		log.Fatalf("loading internal type %s: %v", internalType, err)
	}

	// Load external struct.
	externalInfo, err := loader.LoadStructFromPkg(extPctx, externalType)
	if err != nil {
		log.Fatalf("loading external type %s: %v", externalType, err)
	}

	// Discover getters on the external type (for Protobuf support).
	getters := loader.DiscoverGetters(extPctx, externalType)

	// Perform cross-package matching.
	crossResult := matcher.MatchCross(internalInfo, externalInfo, getters)

	// Warn about unmatched fields.
	for _, f := range crossResult.ToExternal.Unmatched {
		fmt.Fprintf(os.Stderr, "mapper-gen: warning: field %s.%s has no matching external field\n",
			internalType, f.Name)
	}

	// Determine function names.
	toExtFn := funcName
	if toExtFn == "" {
		toExtFn = "Map" + internalType + "To" + externalType
	}
	fromExtFn := "Map" + externalType + "To" + internalType

	ccfg := generator.CrossConfig{
		PackageName:          internalInfo.PackageName,
		InternalType:         internalType,
		ExternalType:         externalType,
		ExternalPkgName:      externalInfo.PackageName,
		ExternalPkgPath:      extPkgPath,
		ToExternalFuncName:   toExtFn,
		FromExternalFuncName: fromExtFn,
		ToExternalPairs:      crossResult.ToExternal.Pairs,
		FromExternalPairs:    crossResult.FromExternal.Pairs,
		Bidirectional:        bidi,
	}

	code, err := generator.GenerateCross(ccfg)
	if err != nil {
		log.Fatalf("generating cross-package code: %v", err)
	}

	outPath := filepath.Join(workDir, output)
	if err := os.WriteFile(outPath, code, 0644); err != nil {
		log.Fatalf("writing output file %s: %v", outPath, err)
	}

	fmt.Fprintf(os.Stderr, "mapper-gen: wrote %s\n", outPath)
}

func resolveDir(explicit string) string {
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

func toSnakeCase(s string) string {
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
