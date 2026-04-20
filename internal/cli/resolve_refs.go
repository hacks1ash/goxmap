package cli

import (
	"fmt"

	"github.com/hacks1ash/goxmap/internal/loader"
	"github.com/hacks1ash/goxmap/internal/reference"
)

// ResolvedSide holds the loaded package context and struct metadata for one
// side (src or dst) of a mapping pair.
type ResolvedSide struct {
	Pctx *loader.PackageContext
	Info *loader.StructInfo
	Ref  reference.Ref
}

// ResolvedRefs is the result of resolving both sides of a mapping together
// with the output package selection.
type ResolvedRefs struct {
	Src               ResolvedSide
	Dst               ResolvedSide
	OutputDir         string
	OutputPackageName string
	SamePackage       bool
}

// ResolveRefs loads both struct types referred to by src and dst, validates
// that at least one of them lives in the current working-dir package, and
// returns the resolved pair together with output package metadata.
func ResolveRefs(workDir string, src, dst reference.Ref) (*ResolvedRefs, error) {
	modPath, _, err := reference.ModuleRoot(workDir)
	if err != nil {
		modPath = ""
	}

	srcPctx, err := loadForRef(workDir, src, modPath)
	if err != nil {
		return nil, fmt.Errorf("loading -src: %w", err)
	}
	dstPctx, err := loadForRef(workDir, dst, modPath)
	if err != nil {
		return nil, fmt.Errorf("loading -dst: %w", err)
	}

	srcInfo, err := loader.LoadStructFromPkg(srcPctx, src.TypeName)
	if err != nil {
		return nil, fmt.Errorf("loading -src type: %w", err)
	}
	dstInfo, err := loader.LoadStructFromPkg(dstPctx, dst.TypeName)
	if err != nil {
		return nil, fmt.Errorf("loading -dst type: %w", err)
	}

	samePackage := srcPctx.Pkg.PkgPath == dstPctx.Pkg.PkgPath

	currentPctx, err := loader.LoadPackage(workDir)
	if err != nil {
		return nil, fmt.Errorf("loading working-dir package: %w", err)
	}
	currentPath := currentPctx.Pkg.PkgPath

	srcIsCurrent := srcPctx.Pkg.PkgPath == currentPath
	dstIsCurrent := dstPctx.Pkg.PkgPath == currentPath

	if !srcIsCurrent && !dstIsCurrent && src.Kind != reference.KindBare && dst.Kind != reference.KindBare {
		return nil, fmt.Errorf(
			"neither -src (%s) nor -dst (%s) lives in the current package (%s); "+
				"run goxmap from the package the mapper should belong to, or pass -dir",
			srcPctx.Pkg.PkgPath, dstPctx.Pkg.PkgPath, currentPath)
	}

	return &ResolvedRefs{
		Src:               ResolvedSide{Pctx: srcPctx, Info: srcInfo, Ref: src},
		Dst:               ResolvedSide{Pctx: dstPctx, Info: dstInfo, Ref: dst},
		OutputDir:         workDir,
		OutputPackageName: currentPctx.Pkg.Name,
		SamePackage:       samePackage,
	}, nil
}

// loadForRef returns a PackageContext for the given reference. Bare references
// load the working-dir package; all other kinds load by import path.
func loadForRef(workDir string, r reference.Ref, modPath string) (*loader.PackageContext, error) {
	importPath := r.ImportPath(modPath)
	if importPath == "" {
		return loader.LoadPackage(workDir)
	}
	return loader.LoadExternalPackage(workDir, importPath)
}
