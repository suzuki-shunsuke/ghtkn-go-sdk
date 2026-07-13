package env

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// TestAll_matchesConstants parses env.go and asserts that All contains exactly the
// string constants declared in this package (every one is an environment variable name).
// Adding a constant without adding it to All (or leaving a stale entry in All) fails this
// test, so `ghtkn info`, which iterates All, can never silently omit a variable.
func TestAll_matchesConstants(t *testing.T) {
	t.Parallel()

	consts := declaredConstants(t)

	all := map[string]struct{}{}
	for _, name := range All {
		if _, dup := all[name]; dup {
			t.Errorf("All contains a duplicate: %s", name)
		}
		all[name] = struct{}{}
	}

	for name := range consts {
		if _, ok := all[name]; !ok {
			t.Errorf("constant %q is declared but missing from All", name)
		}
	}
	for name := range all {
		if _, ok := consts[name]; !ok {
			t.Errorf("All contains %q which is not a declared GHTKN_ constant", name)
		}
	}
}

// declaredConstants parses env.go and returns the set of every string constant value
// declared in it (each is an environment variable name).
func declaredConstants(t *testing.T) map[string]struct{} {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "env.go", nil, 0)
	if err != nil {
		t.Fatalf("parse env.go: %v", err)
	}
	consts := map[string]struct{}{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, v := range vs.Values {
				lit, ok := v.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", lit.Value, err)
				}
				consts[s] = struct{}{}
			}
		}
	}
	return consts
}
