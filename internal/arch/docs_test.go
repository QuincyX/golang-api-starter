package arch

import (
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var markdownLink = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestRequiredDocsExist(t *testing.T) {
	required := []string{
		"AGENTS.md",
		"README.md",
		"docs/README.md",
		"docs/architecture.md",
		"docs/engineering.md",
		"docs/testing.md",
		"docs/template-guide.md",
		"docs/changelog.md",
		"docs/design.md",
		"specs/README.md",
		"specs/constitution.md",
		"specs/_templates/spec.md",
		"specs/_templates/plan.md",
		"specs/_templates/tasks.md",
		"specs/_templates/verification.md",
		"specs/001-account-auth/spec.md",
		"specs/001-account-auth/plan.md",
		"specs/001-account-auth/tasks.md",
		"specs/001-account-auth/verification.md",
		"specs/002-article-management/spec.md",
		"specs/002-article-management/verification.md",
		"specs/003-audit-observability/spec.md",
		"specs/003-audit-observability/verification.md",
	}
	root := projectRoot(t)
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Errorf("required document missing: %s (%v)", rel, err)
		}
	}
}

func TestRelativeMarkdownLinks(t *testing.T) {
	root := projectRoot(t)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "tmp", "bin":
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(root, path)

		for _, match := range markdownLink.FindAllStringSubmatch(string(data), -1) {
			target := strings.TrimSpace(match[1])
			target = strings.Trim(target, "<>")
			if target == "" || strings.HasPrefix(target, "#") {
				continue
			}
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			target = strings.SplitN(target, "#", 2)[0]
			if target == "" {
				continue
			}
			if decoded, err := url.PathUnescape(target); err == nil {
				target = decoded
			}

			resolved := filepath.Join(filepath.Dir(path), filepath.FromSlash(target))
			if _, err := os.Stat(resolved); err != nil {
				t.Errorf("%s: link %q does not resolve: %v", relPath, match[1], err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk markdown files: %v", err)
	}
}
