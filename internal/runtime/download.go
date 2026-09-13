package runtime

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"llama-manager/internal/config"
)

const (
	githubAPI   = "https://api.github.com/repos/ggml-org/llama.cpp/releases"
	userAgent   = "llama-manager"
	cacheTTL    = time.Hour
	maxReleases = 30
)

// Release 表示一个 GitHub Release。
type Release struct {
	TagName     string    `json:"tagName"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"publishedAt"`
	Prerelease  bool      `json:"prerelease"`
	Assets      []Asset   `json:"assets"`
}

// Asset 表示 Release 中的一个附件。
type Asset struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

type releaseCache struct {
	mu        sync.Mutex
	items     []Release
	fetchedAt time.Time
}

var cache releaseCache

// FetchReleases 获取最近的官方 Release，结果缓存 1 小时。
func FetchReleases() ([]Release, error) {
	cache.mu.Lock()
	if len(cache.items) > 0 && time.Since(cache.fetchedAt) < cacheTTL {
		items := cache.items
		cache.mu.Unlock()
		return items, nil
	}
	cache.mu.Unlock()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?per_page=%d", githubAPI, maxReleases), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("GitHub API 限流（HTTP %d），请稍后重试", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API 返回 HTTP %d", resp.StatusCode)
	}

	var raw []struct {
		TagName     string    `json:"tag_name"`
		Name        string    `json:"name"`
		PublishedAt time.Time `json:"published_at"`
		Prerelease  bool      `json:"prerelease"`
		Assets      []struct {
			Name               string `json:"name"`
			Size               int64  `json:"size"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	items := make([]Release, 0, len(raw))
	for _, r := range raw {
		rel := Release{
			TagName:     r.TagName,
			Name:        r.Name,
			PublishedAt: r.PublishedAt,
			Prerelease:  r.Prerelease,
		}
		for _, a := range r.Assets {
			rel.Assets = append(rel.Assets, Asset{Name: a.Name, Size: a.Size, URL: a.BrowserDownloadURL})
		}
		items = append(items, rel)
	}

	cache.mu.Lock()
	cache.items = items
	cache.fetchedAt = time.Now()
	cache.mu.Unlock()
	return items, nil
}

// ProgressFunc 报告下载进度。phase 为 "main" 或 "cudart"。
type ProgressFunc func(phase, name string, downloaded, total int64)

// findRelease 按 tag 查找 Release。
func findRelease(releases []Release, tag string) *Release {
	for i := range releases {
		if strings.EqualFold(releases[i].TagName, tag) {
			return &releases[i]
		}
	}
	return nil
}

// pickSimpleAsset 挑选非 CUDA 后端的 Windows x64 压缩包。
func pickSimpleAsset(rel Release, backend string) (Asset, error) {
	var candidates []Asset
	for _, a := range rel.Assets {
		name := strings.ToLower(a.Name)
		if !strings.HasSuffix(name, ".zip") {
			continue
		}
		if !strings.HasPrefix(name, "llama-") {
			continue
		}
		if !strings.Contains(name, "bin-win") || !strings.Contains(name, "x64") {
			continue
		}
		if !strings.Contains(name, backend) {
			continue
		}
		candidates = append(candidates, a)
	}
	if len(candidates) == 0 {
		return Asset{}, fmt.Errorf("版本 %s 没有 %s 后端的 Windows x64 构建", rel.TagName, backend)
	}
	// 名称越短越接近通用版本。
	sort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i].Name) < len(candidates[j].Name)
	})
	return candidates[0], nil
}

// DownloadRuntime 下载并解压指定版本，返回注册用的 LlamaRuntime。
// CUDA 后端会同时下载独立的 cudart 运行库。
func DownloadRuntime(store *config.Store, tag, backend string, onProgress ProgressFunc) (*LlamaRuntime, error) {
	releases, err := FetchReleases()
	if err != nil {
		return nil, err
	}
	target := findRelease(releases, tag)
	if target == nil {
		return nil, fmt.Errorf("未找到版本 %s", tag)
	}

	backend = strings.ToLower(backend)
	var main, cudart Asset
	variant := ""
	if backend == "cuda" {
		driver, _ := DetectDriverCUDA()
		main, cudart, variant, err = SelectCudaAssets(*target, driver)
		if err != nil {
			return nil, err
		}
	} else {
		main, err = pickSimpleAsset(*target, backend)
		if err != nil {
			return nil, err
		}
	}

	destID := fmt.Sprintf("%s-%s", tag, backend)
	if variant != "" {
		destID += "-" + variant
	}
	destDir := filepath.Join(store.RuntimesDir(), destID)
	if err := os.RemoveAll(destDir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}
	if err := downloadAndExtract(main, destDir, "main", onProgress); err != nil {
		return nil, err
	}
	if cudart.Name != "" {
		if err := downloadAndExtract(cudart, destDir, "cudart", onProgress); err != nil {
			return nil, err
		}
	}

	exe, err := FindServerExecutable(destDir)
	if err != nil {
		return nil, err
	}

	buildTag := parseBuildTag(tag)
	if buildTag == "" {
		buildTag = tag
	}
	return &LlamaRuntime{
		ID:          destID,
		BuildTag:    buildTag,
		Backend:     backend,
		Variant:     variant,
		Arch:        "x64",
		Executable:  exe,
		WorkDir:     filepath.Dir(exe),
		Source:      "download",
		InstalledAt: time.Now(),
	}, nil
}

// DownloadCudartFor 为已安装的 CUDA 版本补全 cudart 运行库。
func DownloadCudartFor(tag, variant, workDir string, onProgress ProgressFunc) error {
	releases, err := FetchReleases()
	if err != nil {
		return err
	}
	target := findRelease(releases, tag)
	if target == nil {
		return fmt.Errorf("未找到版本 %s", tag)
	}
	driver, _ := DetectDriverCUDA()
	asset := findCudartAsset(*target, variant, driver)
	if asset.Name == "" {
		return fmt.Errorf("未找到匹配的 CUDA 运行库（版本 %s）", variant)
	}
	return downloadAndExtract(asset, workDir, "cudart", onProgress)
}

func downloadAndExtract(asset Asset, destDir, phase string, onProgress ProgressFunc) error {
	tmpDir, err := os.MkdirTemp("", "llama-dl-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, asset.Name)
	if err := downloadFile(asset.URL, zipPath, func(downloaded, total int64) {
		if onProgress != nil {
			onProgress(phase, asset.Name, downloaded, total)
		}
	}); err != nil {
		return err
	}
	return unzip(zipPath, destDir)
}

func downloadFile(url, dest string, onProgress func(downloaded, total int64)) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败 HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 64*1024)
	lastReport := time.Now()
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			downloaded += int64(n)
			if onProgress != nil && time.Since(lastReport) > 100*time.Millisecond {
				onProgress(downloaded, total)
				lastReport = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if onProgress != nil {
		onProgress(downloaded, total)
	}
	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	destClean := filepath.Clean(dest) + string(os.PathSeparator)
	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(filepath.Clean(target), destClean) {
			return fmt.Errorf("非法的压缩包路径: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
