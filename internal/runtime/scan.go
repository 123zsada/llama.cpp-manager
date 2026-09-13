package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	versionRe = regexp.MustCompile(`(?i)version:\s*(\d+)`)
	tagRe     = regexp.MustCompile(`(?i)\bb(\d{4,})\b`)
)

// ScanLocalDir 扫描本地目录，识别其中的 llama.cpp 构建。
func ScanLocalDir(dir string) (*LlamaRuntime, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("目录不存在: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("不是目录: %s", abs)
	}

	exe, err := FindServerExecutable(abs)
	if err != nil {
		return nil, err
	}

	buildTag := detectBuildTag(exe)
	backend := DetectBackend(exe, abs)

	return &LlamaRuntime{
		ID:          fmt.Sprintf("%s-%s", buildTag, backend),
		BuildTag:    buildTag,
		Backend:     backend,
		Variant:     parseCudaVersion(abs),
		Arch:        "x64",
		Executable:  exe,
		WorkDir:     filepath.Dir(exe),
		Source:      "local",
		InstalledAt: time.Now(),
	}, nil
}

// FindServerExecutable 在目录中递归查找 llama-server.exe。
func FindServerExecutable(dir string) (string, error) {
	var found string
	errStop := fmt.Errorf("found")
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		name := strings.ToLower(info.Name())
		if name == "llama-server.exe" || name == "llama-server" {
			found = path
			return errStop
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("在 %s 中未找到 llama-server.exe", dir)
	}
	return found, nil
}

// detectBuildTag 通过 --version 读取构建号，失败时回退到目录名。
func detectBuildTag(exe string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, exe, "--version").CombinedOutput()
	if err == nil {
		if tag := parseBuildTag(string(out)); tag != "" {
			return tag
		}
	}
	// 回退：从路径中寻找 bNNNN
	if m := tagRe.FindStringSubmatch(exe); len(m) == 2 {
		return "b" + m[1]
	}
	return "bunknown"
}

func parseBuildTag(output string) string {
	if m := versionRe.FindStringSubmatch(output); len(m) == 2 {
		return "b" + m[1]
	}
	if m := tagRe.FindStringSubmatch(output); len(m) == 2 {
		return "b" + m[1]
	}
	return ""
}

// DetectBackend 依据路径与同目录 DLL 判断后端类型。
func DetectBackend(exe, dir string) string {
	lower := strings.ToLower(exe)
	names := collectNames(dir)

	switch {
	case strings.Contains(lower, "cuda") || names["ggml-cuda.dll"] || names["cudart64"]:
		return "cuda"
	case strings.Contains(lower, "vulkan") || names["ggml-vulkan.dll"]:
		return "vulkan"
	case strings.Contains(lower, "avx2"):
		return "avx2"
	}
	return "cpu"
}

func collectNames(dir string) map[string]bool {
	set := map[string]bool{}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		set[strings.ToLower(info.Name())] = true
		return nil
	})
	// 只关心前缀匹配 cudart
	for n := range set {
		if strings.HasPrefix(n, "cudart64") {
			set["cudart64"] = true
		}
	}
	return set
}
