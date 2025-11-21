package main

// Analyzer implementations:
// - SystemTagAnalyzer: finds //bevi:system ... annotations and creates System model entries.
// - ParamInferAnalyzer: infers parameter kinds/types for each annotated system,
//   including pointer-marked *bevi.QueryN[T] => write signal via Param.Pointer=true.
//   The emitter should treat ParamECSQuery with Pointer=true as WRITE access and
//   non-pointer queries as READ access by default.

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"path"
	"regexp"
	"strconv"
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
					PkgDir:             pkg.Dir,
					PkgName:            pkg.Name,
					FilePath:           gf.Path,
					FuncName:           fd.Name.Name,
					Stage:              stage,
					SystemName:         fd.Name.Name,
					ExtraImports:       make(map[string]string),
					DerivedAliasCounts: make(map[string]int),
					FilterByParam:      make(map[string]FilterOptions),
				}
				if err := parseOptionsInto(optsStr, sys); err != nil {
					return fmt.Errorf("parse options for %s: %w", sys.FuncName, err)
				}

				// Collect imports (normal and blank) from this file for later synthesis.
				// Map alias -> import path. For blank imports, derive alias from last path segment.
				// Resolve import aliases (normal and blank), derive unique aliases for blanks/conflicts.
				aliasMap := make(map[string]string) // original/base alias -> resolved unique alias
				for _, imp := range gf.Ast.Imports {
					if imp == nil || imp.Path == nil {
						continue
					}
					ip := strings.Trim(imp.Path.Value, "\"")
					if ip == "" {
						continue
					}
					// Prefer explicit alias if provided and not blank ("_")
					if imp.Name != nil && imp.Name.Name != "" && imp.Name.Name != "_" {
						al := imp.Name.Name
						aliasMap[al] = al
						// Keep explicit alias as-is
						if _, exists := sys.ExtraImports[al]; !exists {
							sys.ExtraImports[al] = ip
						}
						continue
					}
					// Derive base alias from last path segment for blank or unaliased imports
					base := path.Base(ip)
					if base == "" {
						continue
					}
					resolved := base
					// Ensure uniqueness across this system's imports
					if prevPath, ok := sys.ExtraImports[resolved]; ok && prevPath != ip {
						// Collision: derive suffixed alias using counters
						start := max(sys.DerivedAliasCounts[base], 2)
						for {
							cand := base + strconv.Itoa(start)
							if _, taken := sys.ExtraImports[cand]; !taken {
								resolved = cand
								sys.DerivedAliasCounts[base] = start + 1
								break
							}
							start++
						}
					}
					aliasMap[base] = resolved
					sys.ExtraImports[resolved] = ip
				}

				// Parse //bevi:filter DSL lines: "<target> [+Type|-Type|!exclusive|!register]..."
				for _, c := range fd.Doc.List {
					txt := strings.TrimPrefix(c.Text, "//")
					txt = strings.TrimPrefix(txt, "/*")
					txt = strings.TrimSuffix(txt, "*/")
					txt = strings.TrimSpace(txt)
					if !strings.HasPrefix(txt, "bevi:filter") {
						continue
					}
					rest := strings.TrimSpace(strings.TrimPrefix(txt, "bevi:filter"))
					if rest == "" {
						continue
					}
					toks := splitTopLevel(rest)
					if len(toks) == 0 {
						continue
					}
					target := toks[0] // parameter name or positional like Q0/F0
					opts := sys.FilterByParam[target]
					for _, tk := range toks[1:] {
						if tk == "" {
							continue
						}
						switch {
						case strings.HasPrefix(tk, "+"):
							ty := strings.TrimSpace(strings.TrimPrefix(tk, "+"))
							if ty != "" {
								// Rewrite alias if qualified and known; warn if unknown alias used
								if dot := strings.IndexByte(ty, '.'); dot > 0 {
									al := ty[:dot]
									name := ty[dot+1:]
									if res, ok := aliasMap[al]; ok && res != "" {
										ty = res + "." + name
									} else if _, ok := sys.ExtraImports[al]; !ok {
										ctx.Logger("filter type %q references unknown import alias %q in %s (%s)", ty, al, sys.FuncName, gf.Path)
									}
								}
								opts.With = append(opts.With, ty)
							}
						case strings.HasPrefix(tk, "-"):
							ty := strings.TrimSpace(strings.TrimPrefix(tk, "-"))
							if ty != "" {
								if dot := strings.IndexByte(ty, '.'); dot > 0 {
									al := ty[:dot]
									name := ty[dot+1:]
									if res, ok := aliasMap[al]; ok && res != "" {
										ty = res + "." + name
									} else if _, ok := sys.ExtraImports[al]; !ok {
										ctx.Logger("filter type %q references unknown import alias %q in %s (%s)", ty, al, sys.FuncName, gf.Path)
									}
								}
								opts.Without = append(opts.Without, ty)
							}
						case tk == "!exclusive":
							opts.Exclusive = true
						case tk == "!register":
							opts.Register = true
						}
					}
					sys.FilterByParam[target] = opts
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
						// Use AST inspection instead of regex on string representation
						p := inferParam(f.Type)

						// Capture the full type expression string for debugging/logging
						var buf bytes.Buffer
						_ = format.Node(&buf, token.NewFileSet(), f.Type)
						p.TypeExpr = buf.String()

						// Preserve parameter names when present; otherwise, append anonymous param once.
						if len(f.Names) == 0 {
							sys.Params = append(sys.Params, p)
						} else {
							for _, nm := range f.Names {
								p2 := p
								if nm != nil {
									p2.Name = nm.Name
								}
								sys.Params = append(sys.Params, p2)
							}
						}
					}
					// Bind per-parameter filter options from //bevi:filter lines by name or positional index (Qk/Fk)
					qi, fi := 0, 0
					matched := make(map[string]bool)
					for i := range sys.Params {
						switch sys.Params[i].Kind {
						case ParamECSQuery, ParamECSFilter:
							// Prefer binding by parameter name
							if sys.Params[i].Name != "" {
								if fo, ok := sys.FilterByParam[sys.Params[i].Name]; ok {
									sys.Params[i].FilterOpts = fo
									matched[sys.Params[i].Name] = true
									if sys.Params[i].Kind == ParamECSQuery {
										qi++
									} else {
										fi++
									}
									continue
								}
							}
							// Fallback to positional aliases Qk/Fk
							if sys.Params[i].Kind == ParamECSQuery {
								key := fmt.Sprintf("Q%d", qi)
								if fo, ok := sys.FilterByParam[key]; ok {
									sys.Params[i].FilterOpts = fo
									matched[key] = true
								}
								qi++
							} else {
								key := fmt.Sprintf("F%d", fi)
								if fo, ok := sys.FilterByParam[key]; ok {
									sys.Params[i].FilterOpts = fo
									matched[key] = true
								}
								fi++
							}
						}
					}
					// Diagnostics for unknown filter targets (typos or no matching param)
					for k := range sys.FilterByParam {
						if !matched[k] {
							ctx.Logger("unknown filter target %q for system %s (%s)", k, sys.FuncName, sys.FilePath)
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

func inferParam(expr ast.Expr) Param {
	var p Param

	// Check for pointer
	if star, ok := expr.(*ast.StarExpr); ok {
		p.Pointer = true
		expr = star.X
	}

	// Handle generics (IndexExpr / IndexListExpr)
	var typeArgs []ast.Expr
	var baseExpr ast.Expr = expr

	if idx, ok := expr.(*ast.IndexExpr); ok {
		baseExpr = idx.X
		typeArgs = []ast.Expr{idx.Index}
	} else if idxList, ok := expr.(*ast.IndexListExpr); ok {
		baseExpr = idxList.X
		typeArgs = idxList.Indices
	}

	// Resolve type name (e.g. "context.Context", "bevi.Query1")
	typeName := ""
	if sel, ok := baseExpr.(*ast.SelectorExpr); ok {
		if xIdent, ok := sel.X.(*ast.Ident); ok {
			typeName = xIdent.Name + "." + sel.Sel.Name
		}
	} else if ident, ok := baseExpr.(*ast.Ident); ok {
		typeName = ident.Name
	}

	// Determine Kind
	switch {
	case typeName == "context.Context" && !p.Pointer:
		p.Kind = ParamContext
	case typeName == "bevi.World": // *bevi.World or bevi.World
		p.Kind = ParamWorld
	case strings.HasPrefix(typeName, "bevi.Map"):
		p.Kind = ParamECSMap
	case strings.HasPrefix(typeName, "bevi.Query"):
		p.Kind = ParamECSQuery
	case strings.HasPrefix(typeName, "bevi.Filter"):
		p.Kind = ParamECSFilter
	case typeName == "bevi.Resource":
		p.Kind = ParamECSResource
	case typeName == "bevi.EventWriter":
		p.Kind = ParamEventWriter
	case typeName == "bevi.EventReader":
		p.Kind = ParamEventReader
	default:
		p.Kind = ParamUnknown
	}

	// Process type arguments if present
	if len(typeArgs) > 0 {
		for _, arg := range typeArgs {
			var b bytes.Buffer
			format.Node(&b, token.NewFileSet(), arg)
			p.ElemTypes = append(p.ElemTypes, b.String())
		}

		prefix := ""
		switch p.Kind {
		case ParamECSMap:
			prefix = "map:"
		case ParamECSQuery:
			prefix = "query:"
		case ParamECSFilter:
			prefix = "flt:"
		case ParamECSResource:
			prefix = "res:"
		case ParamEventWriter:
			prefix = "ew:"
		case ParamEventReader:
			prefix = "er:"
		}
		if prefix != "" {
			p.HelperKey = prefix + strings.Join(p.ElemTypes, ",")
		}
	}

	return p
}
