package main

// Analyzer implementations:
// - SystemTagAnalyzer: finds //bevi:system ... annotations and creates System model entries.
// - ParamInferAnalyzer: infers parameter kinds/types for each annotated system,
//   including pointer-marked *ecs.QueryN[T] => write signal via Param.Pointer=true.
//   The emitter should treat ParamECSQuery with Pointer=true as WRITE access and
//   non-pointer queries as READ access by default.

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"regexp"
	"strings"
	"time"
)

// -----------------------------
// Public analyzers to plug into the registry
// -----------------------------

// BuiltinAnalyzers exposes the default analyzers used by the generator.
// The runner may wire this via its DefaultAnalyzers function.
var BuiltinAnalyzers = []Analyzer{
	SystemTagAnalyzer{},
	ParamInferAnalyzer{},
}

// -----------------------------
// SystemTagAnalyzer
// -----------------------------

type SystemTagAnalyzer struct{}

func (SystemTagAnalyzer) Name() string { return "SystemTagAnalyzer" }

var beviTagRe = regexp.MustCompile(`^\s*bevi:system\s+([A-Za-z_][A-Za-z0-9_]*)\s*(.*)$`)

func (SystemTagAnalyzer) Run(ctx *Context) error {
	for _, pkg := range ctx.Packages {
		for _, gf := range pkg.Files {
			if gf.Ast == nil {
				continue
			}
			for _, decl := range gf.Ast.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Name == nil || fd.Type == nil {
					continue
				}
				if fd.Doc == nil {
					continue
				}
				var tagLine string
				for _, c := range fd.Doc.List {
					txt := strings.TrimPrefix(c.Text, "//")
					txt = strings.TrimPrefix(txt, "/*")
					txt = strings.TrimSuffix(txt, "*/")
					txt = strings.TrimSpace(txt)
					if strings.HasPrefix(txt, "bevi:system") {
						tagLine = txt
						break
					}
				}
				if tagLine == "" {
					continue
				}

				m := beviTagRe.FindStringSubmatch(tagLine)
				if len(m) == 0 {
					return fmt.Errorf("invalid bevi:system tag near %s: %q", gf.Path, tagLine)
				}
				stage := m[1]
				optsStr := m[2]

				sys := &System{
					PkgDir:     pkg.Dir,
					PkgName:    pkg.Name,
					FilePath:   gf.Path,
					FuncName:   fd.Name.Name,
					Stage:      stage,
					SystemName: fd.Name.Name,
				}
				if err := parseOptionsInto(optsStr, sys); err != nil {
					return fmt.Errorf("parse options for %s: %w", sys.FuncName, err)
				}

				// Attach to package (Package exposes SysSpecs and addSystem).
				pkg.addSystem(sys)
			}
		}
	}
	return nil
}

func parseOptionsInto(opts string, out *System) error {
	opts = strings.TrimSpace(opts)
	if opts == "" {
		return nil
	}
	// Options format: Key=Value whitespace separated.
	// Keys: Every, After, Before, Set, Reads, Writes, ResReads, ResWrites
	toks := splitTopLevel(opts)
	for _, tok := range toks {
		kv := strings.SplitN(tok, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("option without '=': %q", tok)
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])
		switch key {
		case "every":
			d, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("Every=%q: %w", val, err)
			}
			out.Every = &d
		case "after":
			items, err := parseStringArray(val)
			if err != nil {
				return fmt.Errorf("After=%q: %w", val, err)
			}
			out.After = items
		case "before":
			items, err := parseStringArray(val)
			if err != nil {
				return fmt.Errorf("Before=%q: %w", val, err)
			}
			out.Before = items
		case "set":
			out.Set = trimQuotes(val)
		case "reads":
			items, err := parseStringArray(val)
			if err != nil {
				return fmt.Errorf("Reads=%q: %w", val, err)
			}
			out.CompReads = items
		case "writes":
			items, err := parseStringArray(val)
			if err != nil {
				return fmt.Errorf("Writes=%q: %w", val, err)
			}
			out.CompWrites = items
		case "resreads":
			items, err := parseStringArray(val)
			if err != nil {
				return fmt.Errorf("ResReads=%q: %w", val, err)
			}
			out.ResReads = items
		case "reswrites":
			items, err := parseStringArray(val)
			if err != nil {
				return fmt.Errorf("ResWrites=%q: %w", val, err)
			}
			out.ResWrites = items
		default:
			return fmt.Errorf("unknown option %q", key)
		}
	}
	return nil
}

// -----------------------------
// ParamInferAnalyzer
// -----------------------------

type ParamInferAnalyzer struct{}

func (ParamInferAnalyzer) Name() string { return "ParamInferAnalyzer" }

var (
	reContext     = regexp.MustCompile(`^(?:\*|\s*)?context\.Context$`)
	reWorld       = regexp.MustCompile(`^\*?ecs\.World$`)
	reECSMap      = regexp.MustCompile(`^ecs\.Map\d+\[(.+)\]$`)
	reECSQuery    = regexp.MustCompile(`^ecs\.Query\d+\[(.+)\]$`)
	reECSFilter   = regexp.MustCompile(`^ecs\.Filter\d+\[(.+)\]$`)
	reECSBatch    = regexp.MustCompile(`^ecs\.Batch\d+\[(.+)\]$`)
	reECSResource = regexp.MustCompile(`^ecs\.Resource\[(.+)\]$`)
	reEventWriter = regexp.MustCompile(`^(?:bevi\.)?EventWriter\[(.+)\]$`)
	reEventReader = regexp.MustCompile(`^(?:bevi\.)?EventReader\[(.+)\]$`)
)

func (ParamInferAnalyzer) Run(ctx *Context) error {
	for _, pkg := range ctx.Packages {
		// Build index of systems by file+func for quick lookup.
		sysByFileFunc, err := indexSystemsByFileFunc(pkg)
		if err != nil {
			return err
		}
		for _, gf := range pkg.Files {
			if gf.Ast == nil {
				continue
			}
			ast.Inspect(gf.Ast, func(n ast.Node) bool {
				fd, ok := n.(*ast.FuncDecl)
				if !ok || fd.Name == nil || fd.Type == nil {
					return true
				}
				key := gf.Path + "::" + fd.Name.Name
				sys := sysByFileFunc[key]
				if sys == nil {
					return true
				}
				// Collect parameters in original order
				if fd.Type.Params != nil {
					for _, f := range fd.Type.Params.List {
						var buf bytes.Buffer
						_ = format.Node(&buf, token.NewFileSet(), f.Type)
						typeExpr := buf.String()

						// Detect pointer marker at the param root, e.g., *ecs.Query1[T]
						norm := normalizeTypeExpr(typeExpr)
						pointer := strings.HasPrefix(norm, "*")
						target := strings.TrimPrefix(norm, "*")

						params := 1
						if len(f.Names) > 0 {
							params = len(f.Names)
						}
						p := inferParamFromType(target, typeExpr, pointer)

						for range params {
							sys.Params = append(sys.Params, p)
						}
					}
				}
				return true
			})
		}
	}
	return nil
}

func indexSystemsByFileFunc(pkg *Package) (map[string]*System, error) {
	m := make(map[string]*System, len(pkg.SysSpecs))
	for _, s := range pkg.SysSpecs {
		m[s.FilePath+"::"+s.FuncName] = s
	}
	return m, nil
}

func inferParamFromType(normalizedNoStar string, originalExpr string, pointer bool) Param {
	// Decide parameter kind based on normalized type string (without leading '*')
	out := Param{TypeExpr: originalExpr, Pointer: pointer}
	switch {
	case reContext.MatchString(normalizedNoStar):
		out.Kind = ParamContext
	case reWorld.MatchString(normalizedNoStar):
		out.Kind = ParamWorld
	case reECSMap.MatchString(normalizedNoStar):
		out.Kind = ParamECSMap
		out.ElemTypes = splitGenericArgs(reECSMap.FindStringSubmatch(normalizedNoStar)[1])
		out.HelperKey = "map:" + strings.Join(out.ElemTypes, ",")
	case reECSQuery.MatchString(normalizedNoStar):
		out.Kind = ParamECSQuery
		out.ElemTypes = splitGenericArgs(reECSQuery.FindStringSubmatch(normalizedNoStar)[1])
		out.HelperKey = "query:" + strings.Join(out.ElemTypes, ",")
		// Note: out.Pointer already indicates *ecs.QueryN[...] (pointer-marked) for write intent.
	case reECSFilter.MatchString(normalizedNoStar):
		out.Kind = ParamECSFilter
		out.ElemTypes = splitGenericArgs(reECSFilter.FindStringSubmatch(normalizedNoStar)[1])
		out.HelperKey = "query:" + strings.Join(out.ElemTypes, ",")
	case reECSBatch.MatchString(normalizedNoStar):
		out.Kind = ParamECSBatch
		out.ElemTypes = splitGenericArgs(reECSBatch.FindStringSubmatch(normalizedNoStar)[1])
		out.HelperKey = "query:" + strings.Join(out.ElemTypes, ",")
	case reECSResource.MatchString(normalizedNoStar):
		out.Kind = ParamECSResource
		out.ElemTypes = splitGenericArgs(reECSResource.FindStringSubmatch(normalizedNoStar)[1])
		out.HelperKey = "res:" + strings.Join(out.ElemTypes, ",")
	case reEventWriter.MatchString(normalizedNoStar):
		out.Kind = ParamEventWriter
		out.ElemTypes = splitGenericArgs(reEventWriter.FindStringSubmatch(normalizedNoStar)[1])
		out.HelperKey = "ew:" + strings.Join(out.ElemTypes, ",")
	case reEventReader.MatchString(normalizedNoStar):
		out.Kind = ParamEventReader
		out.ElemTypes = splitGenericArgs(reEventReader.FindStringSubmatch(normalizedNoStar)[1])
		out.HelperKey = "er:" + strings.Join(out.ElemTypes, ",")
	default:
		out.Kind = ParamUnknown
	}
	return out
}
