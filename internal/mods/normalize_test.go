package mods

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func minJSONManifest(id string) string {
	return `{"id":"` + id + `","name":"n","has_pck":false,"has_dll":false,"affects_gameplay":false}`
}

func minJSONManifestWithVersion(id, version string) string {
	return `{"id":"` + id + `","name":"n","version":"` + version + `","has_pck":false,"has_dll":false,"affects_gameplay":false}`
}

func TestNormalizeLayout_SingleNestedFolder(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "deep", "here")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "m.json"), []byte(minJSONManifest("onlyone")), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := NormalizeLayout(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) > 0 {
		t.Fatalf("errors: %v", rep.Errors)
	}
	ov, err := ListInstalled(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Mods) != 1 {
		t.Fatalf("mods count %d", len(ov.Mods))
	}
	if !ov.Mods[0].LayoutNormalized {
		t.Fatalf("expected normalized layout, got %q", ov.Mods[0].FolderName)
	}
}

func TestNormalizeLayout_MultiModCopiesUnclaimed(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "bundle")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "a.json"), []byte(minJSONManifest("ida")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "b.json"), []byte(minJSONManifest("idb")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "shared.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := NormalizeLayout(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) > 0 {
		t.Fatalf("errors: %v", rep.Errors)
	}
	ov, err := ListInstalled(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Mods) != 2 {
		t.Fatalf("want 2 mods got %d", len(ov.Mods))
	}
	found := 0
	for _, m := range ov.Mods {
		p := filepath.Join(root, filepath.FromSlash(m.FolderName), "shared.txt")
		if _, err := os.Stat(p); err == nil {
			found++
		}
	}
	if found != 2 {
		t.Fatalf("shared.txt copies: %d", found)
	}
}

func TestNormalizeLayout_SameIDDifferentVersionFolders(t *testing.T) {
	root := t.TempDir()
	id := "sts2_lan_connect"
	a := filepath.Join(root, "import_a")
	b := filepath.Join(root, "import_b")
	for _, d := range []string{a, b} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(a, "m.json"), []byte(minJSONManifestWithVersion(id, "0.1.0")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b, "m.json"), []byte(minJSONManifestWithVersion(id, "0.2.0")), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := NormalizeLayout(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) > 0 {
		t.Fatalf("errors: %v", rep.Errors)
	}
	ov, err := ListInstalled(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Mods) != 2 {
		t.Fatalf("want 2 mods got %d", len(ov.Mods))
	}
	for _, m := range ov.Mods {
		if m.Manifest.ID != id {
			t.Fatalf("id %q", m.Manifest.ID)
		}
		if !m.LayoutNormalized {
			t.Fatalf("expected normalized layout %q", m.FolderName)
		}
	}
}

// 同一父目录下两个子文件夹各含一个 mod（常见于压缩包解压后的结构）。
func TestNormalizeLayout_ParentWithTwoModSubdirs(t *testing.T) {
	root := t.TempDir()
	pack := filepath.Join(root, "pack")
	a := filepath.Join(pack, "ModA")
	b := filepath.Join(pack, "ModB")
	for _, d := range []string{a, b} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(a, "m.json"), []byte(minJSONManifest("ida")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b, "m.json"), []byte(minJSONManifest("idb")), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := NormalizeLayout(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) > 0 {
		t.Fatalf("errors: %v", rep.Errors)
	}
	ov, err := ListInstalled(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Mods) != 2 {
		t.Fatalf("want 2 mods got %d", len(ov.Mods))
	}
	for _, m := range ov.Mods {
		if strings.HasPrefix(m.FolderName, "pack/") {
			t.Fatalf("mod should not remain under bundle folder: %q", m.FolderName)
		}
		if !m.LayoutNormalized {
			t.Fatalf("expected normalized layout %q", m.FolderName)
		}
	}
}

func TestActivateModVersion_DisablesOtherSameID(t *testing.T) {
	root := t.TempDir()
	d1 := filepath.Join(root, "s1", "v1")
	d2 := filepath.Join(root, "s2", "v1")
	for _, d := range []string{d1, d2} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	id := "sameid"
	if err := os.WriteFile(filepath.Join(d1, "m.json"), []byte(minJSONManifest(id)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d2, "m.json"), []byte(minJSONManifest(id)), 0o644); err != nil {
		t.Fatal(err)
	}
	ov, err := ListInstalled(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.DuplicateIDs) == 0 {
		t.Fatal("expected duplicate id before activate")
	}
	// Activate first path — should disable second (duplicate id layout is invalid until one is off)
	if err := ActivateModVersion(root, "s1/v1", "m.json"); err != nil {
		t.Fatal(err)
	}
	ov2, err := ListInstalled(root)
	if err != nil {
		t.Fatal(err)
	}
	var d1on, d2off bool
	for _, m := range ov2.Mods {
		if m.FolderName == "s1/v1" && !m.Disabled {
			d1on = true
		}
		if m.FolderName == "s2/v1" && m.Disabled {
			d2off = true
		}
	}
	if !d1on || !d2off {
		t.Fatalf("d1on=%v d2off=%v overview=%+v", d1on, d2off, ov2.Mods)
	}
}
