// Package update handles self-update checks and installation.
package update

const (
	GitHubOwner  = "LinyuJupiter"
	GiteeOwner   = "plume-rain-jupiter"
	DefaultRepo  = "SlayTheSpire2ModManager"
	DefaultAsset = "ModManager.exe"
)

// DownloadSource describes one possible mirror for downloading an update asset.
type DownloadSource struct {
	Name        string `json:"name"`
	DownloadURL string `json:"downloadUrl"`
}

// Info describes the latest release state visible to the UI.
type Info struct {
	CurrentVersion string           `json:"currentVersion"`
	LatestVersion  string           `json:"latestVersion"`
	HasUpdate      bool             `json:"hasUpdate"`
	ReleaseURL     string           `json:"releaseUrl"`
	DownloadURL    string           `json:"downloadUrl"`
	AssetName      string           `json:"assetName"`
	PublishedAt    string           `json:"publishedAt"`
	Notes          string           `json:"notes"`
	Source         string           `json:"source"`
	Sources        []DownloadSource `json:"sources"`
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type githubRelease struct {
	TagName     string         `json:"tag_name"`
	HTMLURL     string         `json:"html_url"`
	Body        string         `json:"body"`
	PublishedAt string         `json:"published_at"`
	Assets      []releaseAsset `json:"assets"`
	Draft       bool           `json:"draft"`
	Prerelease  bool           `json:"prerelease"`
}
