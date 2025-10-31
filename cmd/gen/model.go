package main

// Core model types and utility helpers for bevi gen.
// These definitions are shared across analyzers and emitters.

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// System represents a single annotated system function discovered in source.
//
// Example mapping from annotation and signature:
//
//	//bevi:system Update After={"A"} Every=500ms Writes={Position} ResReads={Config}
//	func Tick(ctx context.Context, q ecs.Query1[Position], cfg ecs.Resource[Config]) { ... }
//
// Fields:
//   - Stage/Every/Set/After/Before: derived from annotation
//   - CompReads/CompWrites/ResReads/ResWrites: explicit access overrides from annotation
//   - Params: inferred from function parameters
//   - SystemName: registration name (defaults to function name)
type System struct {
	PkgDir   string
	PkgName  string
	FilePath string
	FuncName string

	// Annotation
	Stage      string         // Startup, Update, etc.
	Every      *time.Duration // optional
	Set        string         // optional
	After      []string       // optional
	Before     []string       // optional
	CompReads  []string       // optional component reads override
	CompWrites []string       // optional component writes override
	ResReads   []string       // optional resource reads override
	ResWrites  []string       // optional resource writes override

	// Parameters inferred
	Params []Param

	// Registration name; defaults to function name if empty.
	SystemName string
}

// ParamKind describes the high-level category for an injected parameter.
type ParamKind int

const (
	ParamUnknown ParamKind = iota
	ParamContext
	ParamWorld
	ParamECSMap
	ParamECSQuery
	ParamECSResource
	ParamEventWriter
	ParamEventReader
)

// String returns a short label for the parameter kind (debugging).
func (k ParamKind) String() string {
	switch k {
	case ParamContext:
		return "Context"
	case ParamWorld:
		return "World"
	case ParamECSMap:
		return "ECSMap"
	case ParamECSQuery:
		return "ECSQuery"
	case ParamECSResource:
		return "ECSResource"
	case ParamEventWriter:
		return "EventWriter"
	case ParamEventReader:
		return "EventReader"

	default:
		return "Unknown"
	}
}

// Param represents an input parameter for a system function.
//
// Fields:
//   - TypeExpr: pretty-printed original type (for debugging)
//   - ElemTypes: type arguments for generic forms (e.g., Query2[T1,T2] => [T1,T2])
//   - HelperKey: deduplication key for prebuilt helpers (e.g., "query:T1,T2")
//   - Pointer: true if parameter type is a pointer to the kind (e.g., *ecs.QueryN[T])
//     This can be used to drive conventions like pointer-marked queries imply write.
type Param struct {
	Kind      ParamKind
	TypeExpr  string
	ElemTypes []string
	HelperKey string
	Pointer   bool
}

// genHelper is an internal declaration used by the emitter to define
// package-level helpers (mappers, filters, resources, readers/writers) exactly once.
type genHelper struct {
	key  string
	kind ParamKind
	typs []string
}

// -----------------------------
// Generic string parsing helpers
// -----------------------------

// trimQuotes removes surrounding single or double quotes, if present.
func trimQuotes(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

// splitTopLevel splits on whitespace while respecting simple quote/bracket nesting.
func splitTopLevel(s string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	inQuote := rune(0)
	for _, r := range s {
		switch r {
		case '"', '\'':
			if inQuote == 0 {
				inQuote = r
			} else if inQuote == r {
				inQuote = 0
			}
			cur.WriteRune(r)
		case '{', '[', '(':
			if inQuote == 0 {
				depth++
			}
			cur.WriteRune(r)
		case '}', ']', ')':
			if inQuote == 0 && depth > 0 {
				depth--
			}
			cur.WriteRune(r)
		case ' ', '\t', '\n', '\r':
			if inQuote == 0 && depth == 0 {
				if cur.Len() > 0 {
					out = append(out, cur.String())
					cur.Reset()
				}
			} else {
				cur.WriteRune(r)
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// splitTopLevelByComma splits a list by commas, respecting simple quote/bracket nesting.
func splitTopLevelByComma(s string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	inQuote := rune(0)
	for _, r := range s {
		switch r {
		case '"', '\'':
			if inQuote == 0 {
				inQuote = r
			} else if inQuote == r {
				inQuote = 0
			}
			cur.WriteRune(r)
		case '{', '[', '(':
			if inQuote == 0 {
				depth++
			}
			cur.WriteRune(r)
		case '}', ']', ')':
			if inQuote == 0 && depth > 0 {
				depth--
			}
			cur.WriteRune(r)
		case ',':
			if inQuote == 0 && depth == 0 {
				out = append(out, cur.String())
				cur.Reset()
			} else {
				cur.WriteRune(r)
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// parseStringArray accepts either { "A", "B" }-style or comma-separated without braces.
func parseStringArray(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	if s == "" {
		return nil, nil
	}
	parts := splitTopLevelByComma(s)
	var out []string
	for _, p := range parts {
		p = trimQuotes(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

// splitGenericArgs splits a generic type list string at top-level commas.
// Example: "T1, ptr[T2], map[string]T3" => ["T1", "ptr[T2]", "map[string]T3"]
func splitGenericArgs(s string) []string {
	parts := splitTopLevelByComma(s)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// -----------------------------
// Emission helpers
// -----------------------------

// genericTypeList joins generic type arguments with ", " and errors if empty.
func genericTypeList(tys []string) (string, error) {
	if len(tys) == 0 {
		return "", errors.New("empty generic type list")
	}
	return strings.Join(tys, ", "), nil
}

// sliceLiteral renders a []string literal from a slice, or "nil" for empty.
func sliceLiteral(ss []string) string {
	if len(ss) == 0 {
		return "nil"
	}
	var b strings.Builder
	b.WriteString("[]string{")
	for i, s := range ss {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strconv.Quote(s))
	}
	b.WriteString("}")
	return b.String()
}

// strOrNil renders a quoted string literal or "" if empty.
func strOrNil(s string) string {
	if s == "" {
		return strconv.Quote("")
	}
	return strconv.Quote(s)
}

// durationLiteral emits a time.Duration constant as <ns>*time.Nanosecond.
func durationLiteral(d time.Duration) string {
	ns := d.Nanoseconds()
	return fmt.Sprintf("%d*time.Nanosecond", ns)
}

// relPath returns the relative path from baseDir to fullPath, or fullPath if it fails.
func relPath(baseDir, fullPath string) string {
	r, err := filepath.Rel(baseDir, fullPath)
	if err != nil {
		return fullPath
	}
	return r
}

// sortUnique returns a sorted copy with unique elements.
func sortUnique(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(in))
	for _, s := range in {
		m[s] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// -----------------------------
// Type normalization helpers
// -----------------------------

var (
	reStripSpaces = regexp.MustCompile(`\s+`)
)

// normalizeTypeExpr removes redundant spaces; call before regex matching.
func normalizeTypeExpr(typeExpr string) string {
	return reStripSpaces.ReplaceAllString(typeExpr, "")
}
