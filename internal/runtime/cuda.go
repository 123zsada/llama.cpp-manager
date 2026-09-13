package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	cudaVersionRe = regexp.MustCompile(`cuda-(\d+)\.(\d+)`)
	driverCudaRe  = regexp.MustCompile(`CUDA Version:\s*([0-9]+\.[0-9]+)`)
)

// DetectDriverCUDA 通过 nvidia-smi 获取驱动支持的最高 CUDA 版本。
func DetectDriverCUDA() (string, bool) {
	candidates := []string{"nvidia-smi"}
	if sysRoot := os.Getenv("SystemRoot"); sysRoot != "" {
		candidates = append(candidates, filepath.Join(sysRoot, "System32", "nvidia-smi.exe"))
	}
	candidates = append(candidates, `C:\Program Files\NVIDIA Corporation\NVSMI\nvidia-smi.exe`)

	for _, exe := range candidates {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		out, err := exec.CommandContext(ctx, exe).CombinedOutput()
		cancel()
		if err != nil && len(out) == 0 {
			continue
		}
		if m := driverCudaRe.FindStringSubmatch(string(out)); len(m) == 2 {
			return m[1], true
		}
	}
	return "", false
}

// parseCudaVersion 从资产名中提取 CUDA 版本，例如 ...cuda-12.4-x64.zip -> 12.4。
func parseCudaVersion(name string) string {
	m := cudaVersionRe.FindStringSubmatch(strings.ToLower(name))
	if len(m) == 3 {
		return m[1] + "." + m[2]
	}
	return ""
}

func splitVersion(v string) (int, int) {
	parts := strings.SplitN(v, ".", 2)
	major, _ := strconv.Atoi(parts[0])
	minor := 0
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor
}

// versionLessEqual 判断 a <= b（x.y 形式）。
func versionLessEqual(a, b string) bool {
	am, an := splitVersion(a)
	bm, bn := splitVersion(b)
	if am != bm {
		return am < bm
	}
	return an <= bn
}

func versionLess(a, b string) bool {
	if a == b {
		return false
	}
	if a == "" {
		return true
	}
	if b == "" {
		return false
	}
	return !versionLessEqual(b, a)
}

// SelectCudaAssets 根据驱动版本选择匹配的 CUDA 主程序与 cudart 运行库。
// 规则：优先选择不高于驱动 CUDA 版本的最高版本；无驱动信息时选择最低版本。
func SelectCudaAssets(rel Release, driverCUDA string) (main Asset, cudart Asset, version string, err error) {
	type candidate struct {
		version string
		asset   Asset
	}
	var candidates []candidate
	for _, a := range rel.Assets {
		name := strings.ToLower(a.Name)
		if !strings.HasSuffix(name, ".zip") {
			continue
		}
		if !strings.HasPrefix(name, "llama-") || !strings.Contains(name, "cuda") {
			continue
		}
		if !strings.Contains(name, "bin-win") || !strings.Contains(name, "x64") {
			continue
		}
		candidates = append(candidates, candidate{version: parseCudaVersion(name), asset: a})
	}
	if len(candidates) == 0 {
		return Asset{}, Asset{}, "", fmt.Errorf("版本 %s 没有 CUDA 的 Windows x64 构建", rel.TagName)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return versionLess(candidates[i].version, candidates[j].version)
	})

	pick := candidates[0]
	if driverCUDA != "" {
		found := false
		for _, c := range candidates {
			if c.version != "" && versionLessEqual(c.version, driverCUDA) {
				pick = c
				found = true
			}
		}
		if !found {
			pick = candidates[0]
		}
	} else {
		for _, c := range candidates {
			if c.version == "" {
				pick = c
				break
			}
		}
	}

	cudart = findCudartAsset(rel, pick.version, driverCUDA)
	if cudart.Name == "" {
		return Asset{}, Asset{}, "", fmt.Errorf("版本 %s 缺少匹配的 CUDA 运行库（cudart %s）", rel.TagName, pick.version)
	}
	return pick.asset, cudart, pick.version, nil
}

// findCudartAsset 查找与指定版本匹配的 cudart 资产。
// version 为空（旧命名）时按驱动版本回退选择。
func findCudartAsset(rel Release, version, driverCUDA string) Asset {
	type candidate struct {
		version string
		asset   Asset
	}
	var candidates []candidate
	for _, a := range rel.Assets {
		name := strings.ToLower(a.Name)
		if !strings.HasPrefix(name, "cudart-") || !strings.HasSuffix(name, ".zip") {
			continue
		}
		if !strings.Contains(name, "bin-win") || !strings.Contains(name, "x64") {
			continue
		}
		candidates = append(candidates, candidate{version: parseCudaVersion(name), asset: a})
	}
	if len(candidates) == 0 {
		return Asset{}
	}
	if version != "" {
		for _, c := range candidates {
			if c.version == version {
				return c.asset
			}
		}
		return Asset{}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return versionLess(candidates[i].version, candidates[j].version)
	})
	if driverCUDA != "" {
		var best Asset
		for _, c := range candidates {
			if c.version == "" || versionLessEqual(c.version, driverCUDA) {
				best = c.asset
			}
		}
		if best.Name != "" {
			return best
		}
	}
	return candidates[0].asset
}

// MissingRuntimeLibs 返回版本运行所需但缺失的本地库文件。
func MissingRuntimeLibs(rt LlamaRuntime) []string {
	dir := rt.WorkDir
	if dir == "" {
		dir = filepath.Dir(rt.Executable)
	}
	var missing []string
	switch rt.Backend {
	case "cuda":
		if !fileExistsGlob(dir, "ggml-cuda.dll") {
			missing = append(missing, "ggml-cuda.dll")
		}
		if !fileExistsGlob(dir, "cublas64_*.dll") {
			missing = append(missing, "cublas64_*.dll")
		}
		if !fileExistsGlob(dir, "cublasLt64_*.dll") {
			missing = append(missing, "cublasLt64_*.dll")
		}
		if !fileExistsGlob(dir, "cudart64_*.dll") {
			missing = append(missing, "cudart64_*.dll")
		}
	case "vulkan":
		if !fileExistsGlob(dir, "ggml-vulkan.dll") {
			missing = append(missing, "ggml-vulkan.dll")
		}
	}
	return missing
}

// ValidateRuntime 校验版本运行所需的本地库是否齐全。
func ValidateRuntime(rt LlamaRuntime) error {
	missing := MissingRuntimeLibs(rt)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%s 运行库缺失（%s），请点击「补全运行库」或重新下载该版本",
		strings.ToUpper(rt.Backend), strings.Join(missing, ", "))
}

func fileExistsGlob(dir, pattern string) bool {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	return err == nil && len(matches) > 0
}
