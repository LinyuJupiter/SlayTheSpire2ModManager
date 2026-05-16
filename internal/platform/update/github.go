package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const userAgent = "SlayTheSpire2ModManager"

type releaseSource struct {
	name           string
	latestAPI      string
	releaseURLBase string
}

var releaseSources = []releaseSource{
	{
		name:           "GitHub",
		latestAPI:      fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubOwner, DefaultRepo),
		releaseURLBase: fmt.Sprintf("https://github.com/%s/%s/releases", GitHubOwner, DefaultRepo),
	},
	{
		name:           "Gitee",
		latestAPI:      fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/releases/latest", GiteeOwner, DefaultRepo),
		releaseURLBase: fmt.Sprintf("https://gitee.com/%s/%s/releases", GiteeOwner, DefaultRepo),
	},
}

// CheckLatest checks release mirrors for the latest stable release.
func CheckLatest(ctx context.Context, currentVersion string) (*Info, error) {
	var lastErr error
	for _, source := range releaseSources {
		info, err := checkLatestFromSource(ctx, currentVersion, source)
		if err == nil {
			info.Sources = buildDownloadSources(info.LatestVersion, source.name)
			return info, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("检查更新失败")
}

func checkLatestFromSource(ctx context.Context, currentVersion string, source releaseSource) (*Info, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.latestAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("检查更新失败: %s 返回 %s", source.name, resp.Status)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.Draft {
		return nil, fmt.Errorf("最新 release 仍是草稿")
	}
	latest := strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
	if latest == "" {
		return nil, fmt.Errorf("%s 最新 release 缺少 tag_name", source.name)
	}
	current := strings.TrimPrefix(strings.TrimSpace(currentVersion), "v")
	downloadURL := downloadURLFor(source.name, latest)
	if asset := findAsset(rel.Assets, DefaultAsset); asset != nil && strings.TrimSpace(asset.BrowserDownloadURL) != "" {
		downloadURL = asset.BrowserDownloadURL
	}
	releaseURL := rel.HTMLURL
	if strings.TrimSpace(releaseURL) == "" {
		releaseURL = source.releaseURLBase + "/tag/v" + latest
	}
	return &Info{
		CurrentVersion: current,
		LatestVersion:  latest,
		HasUpdate:      CompareVersions(latest, current) > 0,
		ReleaseURL:     releaseURL,
		DownloadURL:    downloadURL,
		AssetName:      DefaultAsset,
		PublishedAt:    rel.PublishedAt,
		Notes:          rel.Body,
		Source:         source.name,
	}, nil
}

func buildDownloadSources(version, primary string) []DownloadSource {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	all := []DownloadSource{
		{Name: "GitHub", DownloadURL: downloadURLFor("GitHub", version)},
		{Name: "Gitee", DownloadURL: downloadURLFor("Gitee", version)},
	}
	if primary == "" || strings.EqualFold(primary, all[0].Name) {
		return all
	}
	for i := range all {
		if strings.EqualFold(primary, all[i].Name) {
			return append([]DownloadSource{all[i]}, append(all[:i], all[i+1:]...)...)
		}
	}
	return all
}

func downloadURLFor(sourceName, version string) string {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	switch strings.ToLower(sourceName) {
	case "gitee":
		return fmt.Sprintf("https://gitee.com/%s/%s/releases/download/v%s/%s", GiteeOwner, DefaultRepo, version, DefaultAsset)
	default:
		return fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s/%s", GitHubOwner, DefaultRepo, version, DefaultAsset)
	}
}

func findAsset(assets []releaseAsset, name string) *releaseAsset {
	for i := range assets {
		if strings.EqualFold(assets[i].Name, name) {
			return &assets[i]
		}
	}
	return nil
}
