package mods

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

// 非 mod manifest 的 JSON 文件名（小写），扫描时跳过。
var ignoredNonModJSONBaseNames = map[string]struct{}{
	"package": {}, "tsconfig": {}, "jsconfig": {}, "manifest": {},
	"settings": {}, "launch": {}, "tasks": {}, "extensions": {},
	"package-lock": {}, "composer": {}, "cargo": {}, "pyproject": {},
	"poetry": {}, "pipfile": {}, "renovate": {}, "dependabot": {},
	"azure-pipelines": {}, "bitbucket-pipelines": {}, "travis": {},
	"appveyor": {}, "circleci": {}, "gitlab-ci": {}, "jenkins": {},
	"firebase": {}, "angular": {}, "nx": {}, "lerna": {},
	"bower": {}, "yarn": {}, "pnpm-workspace": {}, "rush": {},
	"webpack": {}, "rollup": {}, "vite": {}, "esbuild": {},
	"babel": {}, "eslint": {}, "prettier": {}, "stylelint": {},
	"jest": {}, "vitest": {}, "cypress": {}, "playwright": {},
	"sonar-project": {}, "codecov": {}, "coveralls": {},
	"swagger": {}, "openapi": {}, "redocly": {}, "spectral": {},
	"graphql-codegen": {}, "graphql.config": {}, "apollo": {},
	"terraform": {}, "pulumi": {}, "serverless": {}, "sam": {},
	"cloudformation": {}, "cdk": {}, "helm": {}, "kustomization": {},
	"skaffold": {}, "tilt": {}, "docker-compose": {}, "compose": {},
	"lefthook": {}, "husky": {},
	"lint-staged": {}, "commitlint": {}, "semantic-release": {},
	"release-please": {}, "changesets": {}, "beachball": {},
	"turbo": {}, "moon": {},
}

func isIgnoredNonModJSONBaseName(base string) bool {
	base = strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(base), ".json.bak"), ".json")
	_, ok := ignoredNonModJSONBaseNames[base]
	return ok
}

// quickProbablyModManifest 对 manifest 做轻量启发式，避免对明显非 mod 的 JSON 做完整解析。
func quickProbablyModManifest(data []byte) bool {
	data = normalizeManifestEncoding(data)
	if len(data) == 0 {
		return false
	}
	if data[0] != '{' {
		return false
	}
	if !bytes.Contains(data, []byte(`"id"`)) && !bytes.Contains(data, []byte(`"Id"`)) && !bytes.Contains(data, []byte(`"ID"`)) {
		return false
	}
	if !bytes.Contains(data, []byte(`"has_pck"`)) && !bytes.Contains(data, []byte(`"hasPck"`)) {
		return false
	}
	if !bytes.Contains(data, []byte(`"has_dll"`)) && !bytes.Contains(data, []byte(`"hasDll"`)) {
		return false
	}
	if !bytes.Contains(data, []byte(`"affects_gameplay"`)) && !bytes.Contains(data, []byte(`"affectsGameplay"`)) {
		return false
	}
	return true
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
	var m ModManifest
	if err := json.Unmarshal(keys["id"], &m.ID); err != nil || strings.TrimSpace(m.ID) == "" {
		return ModManifest{}, false
	}
	if err := json.Unmarshal(keys["has_pck"], &m.HasPck); err != nil {
		return ModManifest{}, false
	}
	if err := json.Unmarshal(keys["has_dll"], &m.HasDll); err != nil {
		return ModManifest{}, false
	}
	if err := json.Unmarshal(keys["affects_gameplay"], &m.AffectsGameplay); err != nil {
		return ModManifest{}, false
	}
	_ = json.Unmarshal(keys["name"], &m.Name)
	_ = json.Unmarshal(keys["author"], &m.Author)
	_ = json.Unmarshal(keys["description"], &m.Description)
	_ = json.Unmarshal(keys["version"], &m.Version)
	if raw, ok := keys["dependencies"]; ok {
		_ = json.Unmarshal(raw, &m.Dependencies)
	}
	return m, true
}

type manifestDiscovered struct {
	FileName string
	Man      ModManifest
	Disabled bool
}

// 防止误把巨型 JSON 当 manifest 一次性读入内存；正常 manifest 很小。
const maxModManifestReadBytes = 256 * 1024

func readModManifestBytes(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	lim := io.LimitReader(f, maxModManifestReadBytes+1)
	b, err := io.ReadAll(lim)
	if err != nil {
		return nil, err
	}
	if len(b) > maxModManifestReadBytes {
		return nil, errors.New("manifest 文件过大")
	}
	return b, nil
}

// listManifestJSONBaseNames 仅枚举目录下 manifest 候选文件名，使用分批 Readdirnames，避免 os.ReadDir
// 在「单目录海量文件」时一次性分配巨大内存。
func listManifestJSONBaseNames(dir string) (enabledNames, bakNames []string, err error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	const chunk = 4096
	for {
		names, rdErr := f.Readdirnames(chunk)
		for _, name := range names {
			if strings.HasPrefix(name, ".") {
				continue
			}
			if isIgnoredNonModJSONBaseName(name) {
				continue
			}
			lower := strings.ToLower(name)
			if strings.HasSuffix(lower, ".json.bak") {
				bakNames = append(bakNames, name)
			} else if strings.HasSuffix(lower, ".json") {
				enabledNames = append(enabledNames, name)
			}
		}
		if rdErr != nil {
			if rdErr != io.EOF {
				return nil, nil, rdErr
			}
			break
		}
		if len(names) < chunk {
			break
		}
	}
	sort.Strings(enabledNames)
	sort.Strings(bakNames)
	const maxJSONNamesPerSide = 96
	if len(enabledNames) > maxJSONNamesPerSide {
		enabledNames = enabledNames[:maxJSONNamesPerSide]
	}
	if len(bakNames) > maxJSONNamesPerSide {
		bakNames = bakNames[:maxJSONNamesPerSide]
	}
	return enabledNames, bakNames, nil
}

func findAllManifestsInFolder(dir string) ([]manifestDiscovered, error) {
	enabledNames, bakNames, err := listManifestJSONBaseNames(dir)
	if err != nil {
		return nil, err
	}
	var out []manifestDiscovered
	for _, name := range enabledNames {
		p := filepath.Join(dir, name)
		b, err := readModManifestBytes(p)
		if err != nil {
			continue
		}
		if !quickProbablyModManifest(b) {
			continue
		}
		if m, ok := tryParseModManifest(b); ok {
			out = append(out, manifestDiscovered{name, m, false})
		}
	}
	for _, name := range bakNames {
		p := filepath.Join(dir, name)
		b, err := readModManifestBytes(p)
		if err != nil {
			continue
		}
		if !quickProbablyModManifest(b) {
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
	b, err := readModManifestBytes(fullPath)
	if err != nil {
		return ModManifest{}, err
	}
	if !quickProbablyModManifest(b) {
		return ModManifest{}, errors.New("JSON 不是有效的 mod 描述")
	}
	m, ok := tryParseModManifest(b)
	if !ok {
		return ModManifest{}, errors.New("JSON 不是有效的 mod 描述")
	}
	return m, nil
}
