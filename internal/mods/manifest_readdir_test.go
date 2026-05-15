package mods

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestListManifestJSONBaseNames_ManyNonJSONFiles(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5000; i++ {
		p := filepath.Join(dir, "a"+strconv.Itoa(i)+".dat")
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mj := `{"id":"x","name":"n","has_pck":false,"has_dll":false,"affects_gameplay":false}`
	if err := os.WriteFile(filepath.Join(dir, "mod.json"), []byte(mj), 0o644); err != nil {
		t.Fatal(err)
	}
	en, bak, err := listManifestJSONBaseNames(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(bak) != 0 || len(en) != 1 || en[0] != "mod.json" {
		t.Fatalf("got en=%v bak=%v", en, bak)
	}
	disc, err := findAllManifestsInFolder(dir)
	if err != nil || len(disc) != 1 {
		t.Fatalf("disc=%v err=%v", disc, err)
	}
}
