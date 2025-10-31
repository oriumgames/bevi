package main

// Runner for bevi gen: orchestrates scanning, analyzing, and emitting.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Run is the entrypoint for the split generator.
// It scans packages, parses files, runs analyzers, and finally emitters.
func Run(opt Options) error {
	log := func(format string, args ...any) {
		if opt.Verbose {
			fmt.Fprintf(os.Stderr, "[gen] "+format+"\n", args...)
		}
	}

	// 1) Scan for packages/files
	pkgs, err := scanPackages(opt)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	if len(pkgs) == 0 {
		log("no packages found under %s", opt.Root)
		return nil
	}

	// 2) Parse files and set package names
	for _, pkg := range pkgs {
		if err := pkg.Parse(); err != nil {
			return fmt.Errorf("parse package %s: %w", pkg.Dir, err)
		}
	}

	// 3) Filter by package name if requested
	var filtered []*Package
	for _, p := range pkgs {
		if opt.PkgPattern == "" || strings.Contains(p.Name, opt.PkgPattern) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		log("no packages match -pkg %q", opt.PkgPattern)
		return nil
	}

	// 4) Build registry and context
	reg := NewRegistry(DefaultAnalyzers(), DefaultEmitters())
	ctx := &Context{
		Options:  opt,
		Packages: filtered,
		Logger:   log,
	}

	// 5) Run analyzers (populate model)
	for _, a := range reg.Analyzers {
		log("run analyzer: %s", a.Name())
		if err := a.Run(ctx); err != nil {
			return fmt.Errorf("analyzer %s: %w", a.Name(), err)
		}
	}

	// 6) Run emitters (write generated files or print)
	for _, e := range reg.Emitters {
		log("run emitter: %s", e.Name())
		if err := e.Run(ctx); err != nil {
			return fmt.Errorf("emitter %s: %w", e.Name(), err)
		}
	}

	return nil
}

// Package holds files that share a directory/package name.
type Package struct {
	Dir      string
	Name     string
	FileSet  *token.FileSet
	Files    []*GoFile
	SysSpecs []*System
}

// addSystem allows analyzers to attach discovered systems to this package.
func (p *Package) addSystem(s *System) {
	p.SysSpecs = append(p.SysSpecs, s)
}

// GoFile represents a file path and its parsed AST.
type GoFile struct {
	Path string
	Ast  *ast.File
}

// Parse parses all Go files in the package and sets the package name.
func (p *Package) Parse() error {
	p.FileSet = token.NewFileSet()
	for _, f := range p.Files {
		astFile, err := parser.ParseFile(p.FileSet, f.Path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", f.Path, err)
		}
		f.Ast = astFile
		// Set package name consistently across files
		if p.Name == "" {
			p.Name = astFile.Name.Name
		}
	}
	return nil
}

// scanPackages walks the root and groups Go files by directory into packages.
// It excludes vendor/.git/node_modules, generated target bevi_gen.go and (optionally) _test.go files.
func scanPackages(opt Options) ([]*Package, error) {
	var pkgs []*Package
	byDir := map[string]*Package{}

	ignoreDir := func(name string) bool {
		if name == "vendor" || name == ".git" || name == "node_modules" || strings.HasPrefix(name, ".") {
			return true
		}
		return false
	}
	ignoreFile := func(name string) bool {
		if !strings.HasSuffix(name, ".go") {
			return true
		}
		// Skip our generated file
		if name == "bevi_gen.go" {
			return true
		}
		// Optionally skip tests
		if strings.HasSuffix(name, "_test.go") && !opt.IncludeTests {
			return true
		}
		return false
	}

	err := filepath.WalkDir(opt.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip ignored directories (except root itself)
		if d.IsDir() {
			if path != opt.Root && ignoreDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if ignoreFile(d.Name()) {
			return nil
		}

		dir := filepath.Dir(path)
		p := byDir[dir]
		if p == nil {
			p = &Package{Dir: dir}
			byDir[dir] = p
			pkgs = append(pkgs, p)
		}
		p.Files = append(p.Files, &GoFile{Path: path})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort for deterministic processing
	for _, p := range pkgs {
		sort.Slice(p.Files, func(i, j int) bool { return p.Files[i].Path < p.Files[j].Path })
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Dir < pkgs[j].Dir })
	return pkgs, nil
}

// Context holds state shared across analyzers and emitters.
type Context struct {
	Options  Options
	Packages []*Package
	Logger   func(format string, args ...any)
}

// Analyzer inspects AST and populates codegen model.
type Analyzer interface {
	Name() string
	Run(ctx *Context) error
}

// Emitter generates source files/code from the model.
type Emitter interface {
	Name() string
	Run(ctx *Context) error
}

// Registry bundles analyzers and emitters.
type Registry struct {
	Analyzers []Analyzer
	Emitters  []Emitter
}

// NewRegistry creates a registry with the provided analyzers and emitters.
func NewRegistry(analyzers []Analyzer, emitters []Emitter) *Registry {
	return &Registry{
		Analyzers: append([]Analyzer(nil), analyzers...),
		Emitters:  append([]Emitter(nil), emitters...),
	}
}

// DefaultAnalyzers returns the default analyzer pipeline.
//
// Note: Implementations should be provided in separate files in this package.
// This default returns an empty slice to keep the runner independent.
func DefaultAnalyzers() []Analyzer { return BuiltinAnalyzers }

// DefaultEmitters returns the default emitter pipeline.
//
// Note: Implementations should be provided in separate files in this package.
// This default returns an empty slice to keep the runner independent.
func DefaultEmitters() []Emitter { return []Emitter{GenEmitter{}} }
