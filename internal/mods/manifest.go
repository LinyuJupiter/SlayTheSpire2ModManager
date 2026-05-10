package mods

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"
)

// UTF-8 BOM：记事本等保存的 JSON 若带 BOM，标准 json 解析会失败，此前会导致整份 manifest 被跳过。
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// normalizeManifestEncoding 将 UTF-16 LE/BE（常见于 Windows「Unicode」记事本）转为 UTF-8，并去掉 UTF-8 BOM。
func normalizeManifestEncoding(b []byte) []byte {
	if len(b) >= 2 {
		if b[0] == 0xFF && b[1] == 0xFE {
			return utf16PayloadToUTF8(b[2:], binaryLittleEndian)
		}
		if b[0] == 0xFE && b[1] == 0xFF {
			return utf16PayloadToUTF8(b[2:], binaryBigEndian)
		}
	}
	return bytes.TrimPrefix(b, utf8BOM)
}

type utf16Endian byte

const (
	binaryLittleEndian utf16Endian = iota
	binaryBigEndian
)

func utf16PayloadToUTF8(b []byte, endian utf16Endian) []byte {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		if endian == binaryLittleEndian {
			u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
		} else {
			u[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
		}
	}
	return []byte(string(utf16.Decode(u)))
}

// jsonKeyAliases：部分工具导出为驼峰字段名，合并为与本程序 / 游戏一致的 snake_case 后再校验。
var jsonKeyAliases = map[string]string{
	"Id":              "id",
	"ID":              "id",
	"hasPck":          "has_pck",
	"hasDll":          "has_dll",
	"affectsGameplay": "affects_gameplay",
}

func mergeCanonicalJSONKeys(keys map[string]json.RawMessage) {
	for alias, canon := range jsonKeyAliases {
		if _, has := keys[canon]; has {
			continue
		}
		if v, ok := keys[alias]; ok {
			keys[canon] = v
			delete(keys, alias)
		}
	}
}

func tryParseModManifest(data []byte) (ModManifest, bool) {
	data = normalizeManifestEncoding(data)
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil {
		return ModManifest{}, false
	}
	mergeCanonicalJSONKeys(keys)
	required := []string{"id", "has_pck", "has_dll", "affects_gameplay"}
	for _, k := range required {
		if _, ok := keys[k]; !ok {
			return ModManifest{}, false
		}
	}
	normalized, err := json.Marshal(keys)
	if err != nil {
		return ModManifest{}, false
	}
	var m ModManifest
	if err := json.Unmarshal(normalized, &m); err != nil {
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
