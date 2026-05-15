package mods

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallFromExtractedTree_FlatRootFiles(t *testing.T) {
	modsRoot := t.TempDir()
	ext := t.TempDir()
	manifest := `{"id":"flat_pack","name":"n","version":"1","has_pck":false,"has_dll":true,"affects_gameplay":false}`
	if err := os.WriteFile(filepath.Join(ext, "flat_pack.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ext, "flat_pack.dll"), []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstallFromExtractedTree(ext, modsRoot); err != nil {
		t.Fatal(err)
	}
	ov, err := ListInstalled(modsRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Mods) != 1 {
		t.Fatalf("mods=%d", len(ov.Mods))
	}
	if ov.Mods[0].Manifest.ID != "flat_pack" {
		t.Fatalf("id %q", ov.Mods[0].Manifest.ID)
	}
}

func TestInstallFromExtractedTree_NestedTwoFolders(t *testing.T) {
	modsRoot := t.TempDir()
	ext := t.TempDir()
	inner := filepath.Join(ext, "Outer", "Inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"nested_two","name":"n","version":"1","has_pck":false,"has_dll":true,"affects_gameplay":false}`
	if err := os.WriteFile(filepath.Join(inner, "nested_two.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inner, "nested_two.dll"), []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstallFromExtractedTree(ext, modsRoot); err != nil {
		t.Fatal(err)
	}
	ov, err := ListInstalled(modsRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Mods) != 1 {
		t.Fatalf("mods=%d", len(ov.Mods))
	}
	if ov.Mods[0].Manifest.ID != "nested_two" {
		t.Fatalf("id %q", ov.Mods[0].Manifest.ID)
	}
}
