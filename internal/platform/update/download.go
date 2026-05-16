package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DownloadLatest downloads the release asset to a temporary update directory.
// If a sibling .sha256 asset exists, it is used to verify the downloaded exe.
func DownloadLatest(ctx context.Context, info *Info) (string, error) {
	if info == nil {
		return "", fmt.Errorf("更新信息为空")
	}
	sources := downloadSources(info)
	if len(sources) == 0 {
		return "", fmt.Errorf("下载地址为空")
	}
	dir, err := os.MkdirTemp("", "sts2-mod-manager-update-*")
	if err != nil {
		return "", err
	}
	dst := filepath.Join(dir, DefaultAsset)
	var lastErr error
	for _, source := range sources {
		if err := downloadFile(ctx, source.DownloadURL, dst); err != nil {
			lastErr = fmt.Errorf("%s 下载失败: %w", source.Name, err)
			continue
		}
		if expected, err := downloadSHA256(ctx, source.DownloadURL+".sha256"); err == nil && expected != "" {
			if err := verifySHA256(dst, expected); err != nil {
				lastErr = fmt.Errorf("%s sha256 校验失败: %w", source.Name, err)
				_ = os.Remove(dst)
				continue
			}
		}
		return dst, nil
	}
	_ = os.RemoveAll(dir)
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("下载更新失败")
}

func downloadSources(info *Info) []DownloadSource {
	var out []DownloadSource
	seen := map[string]struct{}{}
	add := func(source DownloadSource) {
		source.DownloadURL = strings.TrimSpace(source.DownloadURL)
		if source.DownloadURL == "" {
			return
		}
		if source.Name == "" {
			source.Name = "下载源"
		}
		key := strings.ToLower(source.DownloadURL)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, source)
	}
	for _, source := range info.Sources {
		add(source)
	}
	add(DownloadSource{Name: info.Source, DownloadURL: info.DownloadURL})
	return out
}

func downloadFile(ctx context.Context, url, dst string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: %s", resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func downloadSHA256(ctx context.Context, url string) (string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sha256 不存在: %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return "", nil
	}
	return strings.ToLower(strings.TrimSpace(fields[0])), nil
}

func verifySHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("sha256 校验失败: got %s, want %s", got, expected)
	}
	return nil
}
