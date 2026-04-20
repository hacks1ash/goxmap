// Command goxmap generates struct-to-struct mapping functions.
//
// Usage:
//
//	//go:generate go run github.com/hacks1ash/goxmap -src SourceType -dst DestType
//
// Flags:
//
//	-src           Source struct reference (required). Accepts a bare type name,
//	               a module-relative reference (pkg.Type), or a full import-path
//	               reference (github.com/org/repo/pkg.Type).
//	-dst           Destination struct reference (required). Same formats as -src.
//	-func          Generated function name (default: Map<Src>To<Dst>)
//	-dir           Package directory to load types from (default: current directory from $GOFILE or ".")
//	-output        Output file name (default: <dst_lowercase>_mapper_gen.go)
//	-struct-func   Custom function to delegate entire struct mapping
//	-bidi          Generate bidirectional mapper functions
//	-getters       Force getter-based field reads
//	-no-getters    Force direct field reads
package main

import (
	"flag"
	"log"

	"github.com/hacks1ash/goxmap/internal/cli"
	"github.com/hacks1ash/goxmap/internal/matcher"
	"github.com/hacks1ash/goxmap/internal/reference"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("goxmap: ")

	var (
		srcType    = flag.String("src", "", "source struct reference (required)")
		dstType    = flag.String("dst", "", "destination struct reference (required)")
		funcName   = flag.String("func", "", "generated function name (default: Map<Src>To<Dst>)")
		dir        = flag.String("dir", "", "package directory (default: $GOFILE dir or \".\")")
		output     = flag.String("output", "", "output file name (default: <dst_lowercase>_mapper_gen.go)")
		structFunc = flag.String("struct-func", "", "delegate entire mapping to this function")
		bidi       = flag.Bool("bidi", false, "generate bidirectional mapper functions")
		useGetters = flag.Bool("getters", false, "force getter-based field reads")
		noGetters  = flag.Bool("no-getters", false, "force direct field reads")
	)
	flag.Parse()

	if *srcType == "" || *dstType == "" {
		log.Fatal("both -src and -dst flags are required")
	}
	if *useGetters && *noGetters {
		log.Fatal("-getters and -no-getters are mutually exclusive")
	}

	srcRef, err := reference.Parse(*srcType)
	if err != nil {
		log.Fatalf("parsing -src: %v", err)
	}
	dstRef, err := reference.Parse(*dstType)
	if err != nil {
		log.Fatalf("parsing -dst: %v", err)
	}

	workDir := cli.ResolveDir(*dir)
	if *output == "" {
		*output = cli.ToSnakeCase(dstRef.TypeName) + "_mapper_gen.go"
	}

	mode := matcher.GetterModeAuto
	if *useGetters {
		mode = matcher.GetterModeForce
	} else if *noGetters {
		mode = matcher.GetterModeDisabled
	}

	if err := cli.Run(cli.RunOptions{
		WorkDir:    workDir,
		Src:        srcRef,
		Dst:        dstRef,
		FuncName:   *funcName,
		Output:     *output,
		StructFunc: *structFunc,
		Bidi:       *bidi,
		GetterMode: mode,
	}); err != nil {
		log.Fatal(err)
	}
}
