package mods

import (
	"path/filepath"
	"testing"
)

func TestEnumerateRelSkipBadTower(t *testing.T) {
	if !enumerateRelSkipBadTower("Merchant2CuteII/v1.3.1/v1.3.1/v1.3.1") {
		t.Fatal("expected skip triple v1.3.1")
	}
	if enumerateRelSkipBadTower("a/b/c") {
		t.Fatal("normal path should not skip")
	}
	if enumerateRelSkipBadTower("a/a") {
		t.Fatal("only double repeat should not skip")
	}
}

func TestPathIsStrictDescendant(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Slug")
	ver := filepath.Join(root, "ver")
	inner := filepath.Join(ver, "nested")
	if !pathIsStrictDescendant(root, inner) {
		t.Fatal("expected descendant")
	}
	if pathIsStrictDescendant(root, root) {
		t.Fatal("same path")
	}
	if !pathIsStrictDescendant(ver, filepath.Join(ver, "v1")) {
		t.Fatal("expected ver/v1 under ver")
	}
}
