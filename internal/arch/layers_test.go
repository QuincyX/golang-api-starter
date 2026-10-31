package arch

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestLayering enforces the dependency direction documented in AGENTS.md:
// api → service → store. It parses imports instead of relying on review.
func TestLayering(t *testing.T) {
	rules := map[string][]string{
		"internal/api":     {"github.com/example/homebrew-api-starter/internal/store"},
		"internal/service": {"net/http", "github.com/example/homebrew-api-starter/internal/api", "github.com/example/homebrew-api-starter/internal/router"},
		"internal/store":   {"github.com/example/homebrew-api-starter/internal/api", "github.com/example/homebrew-api-starter/internal/service", "github.com/example/homebrew-api-starter/internal/router"},
	}

	root := projectRoot(t)
	for dir, forbidden := range rules {
		base := filepath.Join(root, filepath.FromSlash(dir))
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imp := range file.Imports {
				importPath, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					continue
				}
				for _, bad := range forbidden {
					if importPath == bad || strings.HasPrefix(importPath, bad+"/") {
						rel, _ := filepath.Rel(root, path)
						t.Errorf("%s imports %s (forbidden for %s)", rel, importPath, dir)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
}
