package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hacks1ash/goxmap/internal/generator"
	"github.com/hacks1ash/goxmap/internal/loader"
	"github.com/hacks1ash/goxmap/internal/matcher"
	"github.com/hacks1ash/goxmap/internal/reference"
)

// RunOptions are the parameters for the unified Run entry point.
type RunOptions struct {
	WorkDir    string
	Src        reference.Ref
	Dst        reference.Ref
	FuncName   string
	Output     string
	StructFunc string
	Bidi       bool
	GetterMode matcher.GetterMode
}

// Run resolves the source and destination references, runs matching, and
// generates the mapper file. It selects same-package or cross-package
// generation based on whether the two resolved packages share a path.
func Run(opts RunOptions) error {
	rr, err := ResolveRefs(opts.WorkDir, opts.Src, opts.Dst)
	if err != nil {
		return err
	}

	srcGetters := loader.DiscoverGetters(rr.Src.Pctx, rr.Src.Info.Name)
	dstGetters := loader.DiscoverGetters(rr.Dst.Pctx, rr.Dst.Info.Name)

	if rr.SamePackage {
		return runSamePkg(opts, rr, srcGetters, dstGetters)
	}
	return runCrossPkg(opts, rr, srcGetters, dstGetters)
}

func runSamePkg(opts RunOptions, rr *ResolvedRefs, srcGetters, dstGetters map[string]loader.GetterInfo) error {
	mopts := matcher.MatchOptions{
		SrcGetters: srcGetters,
		DstGetters: dstGetters,
		GetterMode: opts.GetterMode,
	}
	result := matcher.Match(rr.Src.Info, rr.Dst.Info, mopts)

	if err := checkMissingForceGetters(result.Pairs, rr.Src.Info.Name); err != nil {
		return err
	}

	structMapper := opts.StructFunc
	if structMapper == "" {
		structMapper = rr.Dst.Info.MapperFunc
	}

	if structMapper == "" {
		warnUnmatched(result.Unmatched, rr.Dst.Info.Name, false)
	}

	ResolveTypeMismatches(rr.Dst.Pctx, result.Pairs, rr.Src.Info.Name)

	funcName := opts.FuncName
	if funcName == "" {
		funcName = "Map" + rr.Src.Info.Name + "To" + rr.Dst.Info.Name
	}

	rootCfg := generator.Config{
		PackageName:    rr.OutputPackageName,
		FuncName:       funcName,
		SrcType:        rr.Src.Info.Name,
		DstType:        rr.Dst.Info.Name,
		Pairs:          result.Pairs,
		StructMapperFn: structMapper,
	}

	// DiscoverMapperFuncs finds functions already in the package so that nested
	// helper mappers are not re-generated. The root function (and reverse, if
	// bidi) must be removed from this set: they are the ones we are about to
	// regenerate, and treating them as "existing" would cause enqueue to skip
	// them entirely, producing an empty output file.
	existingMappers := loader.DiscoverMapperFuncs(rr.Dst.Pctx)
	delete(existingMappers, funcName)

	mcfg := generator.MultiConfig{
		PackageName:     rr.OutputPackageName,
		RootFunc:        rootCfg,
		ExistingMappers: existingMappers,
		PkgContext:      rr.Dst.Pctx,
	}

	if opts.Bidi {
		revMopts := matcher.MatchOptions{
			SrcGetters: dstGetters,
			DstGetters: srcGetters,
			GetterMode: opts.GetterMode,
		}
		revResult := matcher.Match(rr.Dst.Info, rr.Src.Info, revMopts)
		if err := checkMissingForceGetters(revResult.Pairs, rr.Dst.Info.Name); err != nil {
			return err
		}
		warnUnmatched(revResult.Unmatched, rr.Src.Info.Name, true)
		ResolveTypeMismatches(rr.Src.Pctx, revResult.Pairs, rr.Dst.Info.Name)
		revFuncName := "Map" + rr.Dst.Info.Name + "To" + rr.Src.Info.Name
		delete(existingMappers, revFuncName)
		revCfg := generator.Config{
			PackageName: rr.OutputPackageName,
			FuncName:    revFuncName,
			SrcType:     rr.Dst.Info.Name,
			DstType:     rr.Src.Info.Name,
			Pairs:       revResult.Pairs,
		}
		mcfg.Bidirectional = true
		mcfg.ReverseFunc = &revCfg
	}

	code, err := generator.GenerateMulti(mcfg)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}
	return writeOutput(rr.OutputDir, opts.Output, code)
}

func runCrossPkg(opts RunOptions, rr *ResolvedRefs, srcGetters, dstGetters map[string]loader.GetterInfo) error {
	if rr.Src.Pctx.Pkg.PkgPath == rr.Dst.Pctx.Pkg.PkgPath {
		return fmt.Errorf("internal error: cross-package run with same package path")
	}

	// Determine which side is local (in the working-dir package) and which is external.
	localPctx, err := loader.LoadPackage(opts.WorkDir)
	if err != nil {
		return fmt.Errorf("loading local package: %w", err)
	}

	var localSide, externalSide *ResolvedSide
	var localGetters, externalGetters map[string]loader.GetterInfo

	if rr.Src.Pctx.Pkg.PkgPath == localPctx.Pkg.PkgPath {
		localSide = &rr.Src
		externalSide = &rr.Dst
		localGetters = srcGetters
		externalGetters = dstGetters
	} else {
		localSide = &rr.Dst
		externalSide = &rr.Src
		localGetters = dstGetters
		externalGetters = srcGetters
	}

	// Match local -> external.
	toExtMopts := matcher.MatchOptions{
		SrcGetters: localGetters,
		DstGetters: externalGetters,
		GetterMode: opts.GetterMode,
	}
	toExtResult := matcher.Match(localSide.Info, externalSide.Info, toExtMopts)
	if err := checkMissingForceGetters(toExtResult.Pairs, localSide.Info.Name); err != nil {
		return err
	}
	warnUnmatchedCross(toExtResult.Unmatched, localSide.Info.Name)
	ResolveTypeMismatches(localSide.Pctx, toExtResult.Pairs, localSide.Info.Name)

	// Match external -> local.
	fromExtMopts := matcher.MatchOptions{
		SrcGetters: externalGetters,
		DstGetters: localGetters,
		GetterMode: opts.GetterMode,
	}
	fromExtResult := matcher.Match(externalSide.Info, localSide.Info, fromExtMopts)
	if err := checkMissingForceGetters(fromExtResult.Pairs, externalSide.Info.Name); err != nil {
		return err
	}
	ResolveTypeMismatches(localSide.Pctx, fromExtResult.Pairs, externalSide.Info.Name)

	// Function names — disambiguate when local and external share a struct name.
	intLabel := localSide.Info.Name
	extLabel := externalSide.Info.Name
	if localSide.Info.Name == externalSide.Info.Name {
		extLabel = pascalCase(externalSide.Info.PackageName) + externalSide.Info.Name
	}

	toExtFn := opts.FuncName
	if toExtFn == "" {
		toExtFn = "Map" + intLabel + "To" + extLabel
	}
	fromExtFn := "Map" + extLabel + "To" + intLabel

	ccfg := generator.CrossConfig{
		PackageName:          rr.OutputPackageName,
		InternalType:         localSide.Info.Name,
		ExternalType:         externalSide.Info.Name,
		ExternalPkgName:      externalSide.Info.PackageName,
		ExternalPkgPath:      externalSide.Info.PackagePath,
		ToExternalFuncName:   toExtFn,
		FromExternalFuncName: fromExtFn,
		ToExternalPairs:      toExtResult.Pairs,
		FromExternalPairs:    fromExtResult.Pairs,
		Bidirectional:        opts.Bidi,
	}

	code, err := generator.GenerateCross(ccfg)
	if err != nil {
		return fmt.Errorf("generating cross-package code: %w", err)
	}
	return writeOutput(rr.OutputDir, opts.Output, code)
}

func warnUnmatched(unmatched []loader.StructField, dstName string, reverse bool) {
	prefix := ""
	if reverse {
		prefix = "reverse mapper: "
	}
	for _, f := range unmatched {
		if f.Optional {
			continue
		}
		fmt.Fprintf(os.Stderr,
			"goxmap: warning: %sdestination field %s.%s has no matching source field (skipped, will use zero value)\n",
			prefix, dstName, f.Name)
	}
}

func warnUnmatchedCross(unmatched []loader.StructField, side string) {
	for _, f := range unmatched {
		if f.Optional {
			continue
		}
		fmt.Fprintf(os.Stderr,
			"goxmap: warning: field %s.%s has no matching external field\n",
			side, f.Name)
	}
}

func checkMissingForceGetters(pairs []matcher.FieldPair, srcTypeName string) error {
	for _, p := range pairs {
		if p.MissingGetterForce {
			return fmt.Errorf("-getters required but no getter for field %s.%s", srcTypeName, p.Src.Name)
		}
	}
	return nil
}

func writeOutput(dir, name string, code []byte) error {
	out := filepath.Join(dir, name)
	if err := os.WriteFile(out, code, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}
	fmt.Fprintf(os.Stderr, "goxmap: wrote %s\n", out)
	return nil
}
