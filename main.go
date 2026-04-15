// Command goxmap generates struct-to-struct mapping functions.
//
// Usage:
//
//	//go:generate go run github.com/hacks1ash/goxmap -src SourceType -dst DestType
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
	"log"

	"github.com/hacks1ash/goxmap/internal/cli"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("goxmap: ")

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

	workDir := cli.ResolveDir(*dir)

	if *output == "" {
		*output = cli.ToSnakeCase(*dstType) + "_mapper_gen.go"
	}

	if *externalPkg != "" {
		cli.RunCrossPackage(cli.CrossPackageOptions{
			WorkDir:      workDir,
			InternalType: *srcType,
			ExternalType: *dstType,
			FuncName:     *funcName,
			Output:       *output,
			ExtPkgPath:   *externalPkg,
			Bidi:         *bidi,
		})
		return
	}

	cli.RunSamePackage(cli.SamePackageOptions{
		WorkDir:    workDir,
		SrcType:    *srcType,
		DstType:    *dstType,
		FuncName:   *funcName,
		Output:     *output,
		StructFunc: *structFunc,
		Bidi:       *bidi,
	})
}
