package arch

import (
	"os/exec"
	"strings"
	"testing"
)

// TestGofmtClean makes formatting part of `go test ./...`, so an agent that
// edits Go code cannot finish without running (or matching) gofmt.
func TestGofmtClean(t *testing.T) {
	root := projectRoot(t)

	cmd := exec.Command("gofmt", "-l", ".")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gofmt -l .: %v\n%s", err, out)
	}
	if files := strings.TrimSpace(string(out)); files != "" {
		t.Fatalf("gofmt found unformatted files (run `gofmt -w .`):\n%s", files)
	}
}
