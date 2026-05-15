package mods

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoveDirContents_avoidsNestingVersionIntoItself(t *testing.T) {
	root := t.TempDir()
	slug := filepath.Join(root, "Merchant2CuteII")
	ver := filepath.Join(slug, "v1.3.1")
	if err := os.MkdirAll(ver, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(slug, "m.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ver, "inner.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := ver
	if err := moveDirContents(slug, dest); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(ver, "inner.txt"))
	if err != nil || string(b) != "x" {
		t.Fatalf("inner.txt: %v %q", err, b)
	}
	if _, err := os.Stat(filepath.Join(ver, "m.json")); err != nil {
		t.Fatal("m.json should be under version dir:", err)
	}
	if _, err := os.Stat(filepath.Join(ver, "v1.3.1")); err == nil {
		t.Fatal("should not create ver/ver")
	}
}
