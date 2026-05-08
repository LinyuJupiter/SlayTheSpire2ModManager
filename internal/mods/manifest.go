package mods

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func tryParseModManifest(data []byte) (ModManifest, bool) {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil {
		return ModManifest{}, false
	}
	required := []string{"id", "has_pck", "has_dll", "affects_gameplay"}
	for _, k := range required {
		if _, ok := keys[k]; !ok {
			return ModManifest{}, false
		}
	}
	var m ModManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return ModManifest{}, false
	}
	if strings.TrimSpace(m.ID) == "" {
		return ModManifest{}, false
	}
	return m, true
}

type manifestDiscovered struct {
	FileName string
	Man      ModManifest
	Disabled bool
}

func findAllManifestsInFolder(dir string) ([]manifestDiscovered, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var enabledNames, bakNames []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".json") && !strings.HasSuffix(lower, ".json.bak") {
			enabledNames = append(enabledNames, name)
		}
		if strings.HasSuffix(lower, ".json.bak") {
			bakNames = append(bakNames, name)
		}
	}
	sort.Strings(enabledNames)
	sort.Strings(bakNames)
	var out []manifestDiscovered
	for _, name := range enabledNames {
		p := filepath.Join(dir, name)
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if m, ok := tryParseModManifest(b); ok {
			out = append(out, manifestDiscovered{name, m, false})
		}
	}
	for _, name := range bakNames {
		p := filepath.Join(dir, name)
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if m, ok := tryParseModManifest(b); ok {
			out = append(out, manifestDiscovered{name, m, true})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].FileName < out[j].FileName
	})
	return out, nil
}

func loadManifestFromPath(fullPath string) (ModManifest, error) {
	b, err := os.ReadFile(fullPath)
	if err != nil {
		return ModManifest{}, err
	}
	m, ok := tryParseModManifest(b)
	if !ok {
		return ModManifest{}, errors.New("JSON 不是有效的 mod 描述")
	}
	return m, nil
}
