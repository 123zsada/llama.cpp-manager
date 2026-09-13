package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func fakeCudaRelease() Release {
	mk := func(name string) Asset { return Asset{Name: name, URL: "http://example/" + name} }
	return Release{
		TagName: "b10930",
		Assets: []Asset{
			mk("llama-b10930-bin-win-cuda-12.4-x64.zip"),
			mk("llama-b10930-bin-win-cuda-13.3-x64.zip"),
			mk("cudart-llama-bin-win-cuda-12.4-x64.zip"),
			mk("cudart-llama-bin-win-cuda-13.3-x64.zip"),
			mk("cudart-llama-bin-win-cuda-13.4-arm64.zip"),
			mk("llama-b10930-bin-win-cpu-x64.zip"),
		},
	}
}

func TestParseCudaVersion(t *testing.T) {
	cases := map[string]string{
		"llama-b10930-bin-win-cuda-12.4-x64.zip": "12.4",
		"cudart-llama-bin-win-cuda-13.3-x64.zip": "13.3",
		"llama-b10930-bin-win-cpu-x64.zip":       "",
		"llama-b10930-bin-win-vulkan-x64.zip":    "",
	}
	for name, want := range cases {
		if got := parseCudaVersion(name); got != want {
			t.Errorf("parseCudaVersion(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestSelectCudaAssets(t *testing.T) {
	rel := fakeCudaRelease()
	cases := []struct {
		driver     string
		wantMain   string
		wantCudart string
		wantVer    string
	}{
		{"13.2", "llama-b10930-bin-win-cuda-12.4-x64.zip", "cudart-llama-bin-win-cuda-12.4-x64.zip", "12.4"},
		{"13.3", "llama-b10930-bin-win-cuda-13.3-x64.zip", "cudart-llama-bin-win-cuda-13.3-x64.zip", "13.3"},
		{"13.4", "llama-b10930-bin-win-cuda-13.3-x64.zip", "cudart-llama-bin-win-cuda-13.3-x64.zip", "13.3"},
		{"12.0", "llama-b10930-bin-win-cuda-12.4-x64.zip", "cudart-llama-bin-win-cuda-12.4-x64.zip", "12.4"},
		{"", "llama-b10930-bin-win-cuda-12.4-x64.zip", "cudart-llama-bin-win-cuda-12.4-x64.zip", "12.4"},
	}
	for _, c := range cases {
		main, cudart, ver, err := SelectCudaAssets(rel, c.driver)
		if err != nil {
			t.Fatalf("driver %q: %v", c.driver, err)
		}
		if main.Name != c.wantMain || cudart.Name != c.wantCudart || ver != c.wantVer {
			t.Errorf("driver %q: got main=%q cudart=%q ver=%q, want %q %q %q",
				c.driver, main.Name, cudart.Name, ver, c.wantMain, c.wantCudart, c.wantVer)
		}
	}
}

func TestSelectCudaAssetsNoCudart(t *testing.T) {
	rel := Release{
		TagName: "b1",
		Assets:  []Asset{{Name: "llama-b1-bin-win-cuda-12.4-x64.zip"}},
	}
	if _, _, _, err := SelectCudaAssets(rel, "13.2"); err == nil {
		t.Fatal("expected error when cudart missing")
	}
}

func TestMissingRuntimeLibs(t *testing.T) {
	dir := t.TempDir()
	rt := LlamaRuntime{Backend: "cuda", WorkDir: dir}

	if got := len(MissingRuntimeLibs(rt)); got != 4 {
		t.Fatalf("expected 4 missing libs, got %d", got)
	}
	if err := ValidateRuntime(rt); err == nil {
		t.Fatal("ValidateRuntime should fail when libs missing")
	}

	for _, name := range []string{"ggml-cuda.dll", "cublas64_12.dll", "cublasLt64_12.dll", "cudart64_12.dll"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := MissingRuntimeLibs(rt); len(got) != 0 {
		t.Fatalf("expected no missing libs, got %v", got)
	}
	if err := ValidateRuntime(rt); err != nil {
		t.Fatalf("ValidateRuntime() = %v", err)
	}
}
