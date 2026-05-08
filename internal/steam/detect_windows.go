//go:build windows

package steam

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func installPathFromRegistry() (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	s, _, err := k.GetStringValue("SteamPath")
	return strings.TrimSpace(s), err
}

func parseLibraryFolders(vdfPath string) []string {
	b, err := os.ReadFile(vdfPath)
	if err != nil {
		return nil
	}
	content := string(b)
	re := regexp.MustCompile(`"path"\s+"([^"]+)"`)
	matches := re.FindAllStringSubmatch(content, -1)
	var out []string
	seen := map[string]struct{}{}
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		p := strings.ReplaceAll(m[1], `\\`, `\`)
		p = filepath.Clean(p)
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// FindSlayTheSpire2Exe 在 Steam 库路径下查找游戏主程序。
func FindSlayTheSpire2Exe() string {
	steamRoot, err := installPathFromRegistry()
	if err != nil {
		steamRoot = ""
	}
	steamRoot = strings.TrimRight(steamRoot, `/\`)
	roots := []string{}
	if steamRoot != "" {
		roots = append(roots, steamRoot)
		roots = append(roots, parseLibraryFolders(filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf"))...)
	}
	seen := map[string]bool{}
	rel := filepath.Join("steamapps", "common", "Slay the Spire 2", "SlayTheSpire2.exe")
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" || seen[root] {
			continue
		}
		seen[root] = true
		p := filepath.Join(root, rel)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}
