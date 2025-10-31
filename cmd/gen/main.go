package main

import (
	"flag"
	"fmt"
	"os"
)

// Options holds command-line settings for the generator.
// The actual behavior is implemented in other files of this package.
type Options struct {
	// Root directory to scan (module/package root)
	Root string
	// When true, write bevi_gen.go files. When false, print to stdout.
	Write bool
	// Verbose logging to stderr
	Verbose bool
	// Only process packages whose name contains this substring (optional)
	PkgPattern string
	// Include _test.go files in the scan
	IncludeTests bool
}

func parseFlags() Options {
	var opt Options
	flag.StringVar(&opt.Root, "root", ".", "root directory to scan (module/package root)")
	flag.BoolVar(&opt.Write, "write", true, "write generated files (bevi_gen.go); if false, print to stdout")
	flag.BoolVar(&opt.Verbose, "v", false, "verbose logging")
	flag.StringVar(&opt.PkgPattern, "pkg", "", "only process packages whose name contains this substring (optional)")
	flag.BoolVar(&opt.IncludeTests, "include-tests", false, "include _test.go files during scanning")
	flag.Parse()
	return opt
}

func main() {
	opt := parseFlags()

	// Run is implemented in other files (split across the package).
	// It performs scanning, analysis and emission.
	if err := Run(opt); err != nil {
		fmt.Fprintf(os.Stderr, "bevi gen: %v\n", err)
		os.Exit(2)
	}
}
