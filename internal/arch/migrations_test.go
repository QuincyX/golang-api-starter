package arch

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var updateMigrations = flag.Bool("update-migrations", false, "rewrite the migration checksum manifest")

// TestMigrationChecksums turns "applied migrations are immutable" from prose
// into a mechanical check. Adding a migration is fine; editing one that is
// already recorded in the manifest fails until the manifest is refreshed and
// the change is reviewed. Run `task migrations:checksum` to refresh it.
func TestMigrationChecksums(t *testing.T) {
	root := projectRoot(t)
	dir := filepath.Join(root, "migrations")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	type migration struct {
		name string
		sum  [sha256.Size]byte
	}
	var items []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		items = append(items, migration{name: entry.Name(), sum: sha256.Sum256(data)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })

	var lines []string
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%x  %s", item.sum, item.name))
	}
	got := strings.Join(lines, "\n") + "\n"

	manifest := filepath.Join(root, "internal", "arch", "testdata", "migrations.sha256")
	if *updateMigrations {
		if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
			t.Fatalf("create testdata dir: %v", err)
		}
		if err := os.WriteFile(manifest, []byte(got), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		return
	}

	want, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("read manifest: %v (run `task migrations:checksum`)", err)
	}
	if string(want) != got {
		t.Fatalf("migrations changed since the manifest was recorded; run `task migrations:checksum` and review the diff")
	}
}
