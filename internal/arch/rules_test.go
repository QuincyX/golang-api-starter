package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestMechanicalBoundaries enforces the rules that must not depend on an agent
// remembering prose: routes go through router.Register, and only the config
// package reads environment variables directly.
func TestMechanicalBoundaries(t *testing.T) {
	root := projectRoot(t)
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "testdata", "vendor", ".tmp", "node_modules":
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if strings.HasSuffix(rel, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", rel, err)
		}

		if rel != "internal/router/register.go" {
			if pos, ok := firstSelectorCall(file, "huma", "Register"); ok {
				t.Errorf("%s: register routes with router.Register, not huma.Register (line %d)", rel, fset.Position(pos).Line)
			}
		}

		if strings.HasPrefix(rel, "internal/") && !strings.HasPrefix(rel, "internal/config/") {
			for _, name := range []string{"Getenv", "LookupEnv"} {
				if pos, ok := firstSelectorCall(file, "os", name); ok {
					t.Errorf("%s: read configuration through internal/config, not os.%s (line %d)", rel, name, fset.Position(pos).Line)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk Go files: %v", err)
	}
}

func firstSelectorCall(file *ast.File, pkg, name string) (token.Pos, bool) {
	var pos token.Pos
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		if found {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != pkg || sel.Sel.Name != name {
			return true
		}
		pos, found = sel.Pos(), true
		return false
	})
	return pos, found
}
