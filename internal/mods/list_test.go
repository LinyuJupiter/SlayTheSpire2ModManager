package mods

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListInstalled_NestedModFolder(t *testing.T) {
	root := t.TempDir()
	modDir := filepath.Join(root, "dir1", "dir2")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"nested_test_mod","name":"n","has_pck":false,"has_dll":false,"affects_gameplay":false}`
	if err := os.WriteFile(filepath.Join(modDir, "mod.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	overview, err := ListInstalled(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Mods) != 1 {
		t.Fatalf("want 1 mod, got %d", len(overview.Mods))
	}
	if overview.Mods[0].FolderName != "dir1/dir2" {
		t.Fatalf("folderName: got %q want dir1/dir2", overview.Mods[0].FolderName)
	}
}
