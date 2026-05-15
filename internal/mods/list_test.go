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

func TestListInstalled_SameIDSameSlugDifferentVersionsNoConflict(t *testing.T) {
	root := t.TempDir()
	id := "sts2_lan_connect"
	man := func(ver string) string {
		return `{"id":"` + id + `","name":"STS2 LAN Connect","version":"` + ver + `","has_pck":false,"has_dll":false,"affects_gameplay":false}`
	}
	slug := "STS2 LAN Connect"
	for _, ver := range []string{"0.3.0", "0.3.1"} {
		d := filepath.Join(root, slug, ver)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "m.json"), []byte(man(ver)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ov, err := ListInstalled(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.DuplicateIDs) != 0 {
		t.Fatalf("duplicateIDs: %v", ov.DuplicateIDs)
	}
	for _, m := range ov.Mods {
		if len(m.ConflictWith) != 0 {
			t.Fatalf("conflicts %q: %v", m.FolderName, m.ConflictWith)
		}
		if !m.IDUnique || !m.Available {
			t.Fatalf("mod %q: IDUnique=%v Available=%v", m.FolderName, m.IDUnique, m.Available)
		}
	}
}
