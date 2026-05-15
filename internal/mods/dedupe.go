package mods

import (
	"path/filepath"
	"sort"
	"strings"
)

// DedupeEnabledRetainHighestPerSlugID 对「已规范」且同 slug、同 id 的多份启用，只保留 version 最高的一份启用。
func DedupeEnabledRetainHighestPerSlugID(modsRoot string) error {
	ov, err := ListInstalled(modsRoot)
	if err != nil {
		return err
	}
	type gk struct {
		slug string
		id   string
	}
	groups := map[gk][]InstalledMod{}
	for _, m := range ov.Mods {
		if !m.LayoutNormalized {
			continue
		}
		slug := slugFirstSegment(m.FolderName)
		if slug == "" {
			continue
		}
		k := gk{strings.ToLower(slug), m.Manifest.ID}
		groups[k] = append(groups[k], m)
	}
	for _, ms := range groups {
		var enabled []InstalledMod
		for _, m := range ms {
			if !m.Disabled {
				enabled = append(enabled, m)
			}
		}
		if len(enabled) <= 1 {
			continue
		}
		sort.Slice(enabled, func(i, j int) bool {
			return CompareModVersionStrings(enabled[i].Manifest.Version, enabled[j].Manifest.Version) > 0
		})
		keep := enabled[0]
		for _, m := range enabled[1:] {
			if m.FolderName == keep.FolderName && m.ManifestFile == keep.ManifestFile {
				continue
			}
			if err := Disable(modsRoot, m.FolderName, m.ManifestFile); err != nil {
				return err
			}
		}
	}
	return nil
}

func slugFirstSegment(folderName string) string {
	s := strings.Trim(strings.TrimSpace(folderName), `/\`)
	segs := strings.Split(filepath.ToSlash(s), "/")
	if len(segs) == 0 {
		return ""
	}
	return segs[0]
}
