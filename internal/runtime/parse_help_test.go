package runtime

import "testing"

const sampleHelp = `----- common params -----

-h,    --help, --usage                  print this help and exit
--version                               show version and build info
-m,    --model FNAME                    model path to load
-mu,   --model-url MODEL_URL            model download url
-ngl,  --gpu-layers, --n-gpu-layers N   max. number of layers to store in VRAM
-c,    --ctx-size N                     size of the prompt context
-t,    --threads N                      number of threads to use
--temp, --temperature N                 temperature
--top-k N                               top-k sampling
--host HOST                             ip address to listen
--port PORT                             port to listen
-np,   --parallel N                     number of parallel sequences
--no-mmap                               do not memory-map model
--cpu-mask M                            CPU affinity mask
`

func TestParseHelp(t *testing.T) {
	defs := ParseHelp(sampleHelp)
	if len(defs) < 10 {
		t.Fatalf("expected at least 10 params, got %d", len(defs))
	}

	byFlag := map[string]ParamDef{}
	for _, d := range defs {
		byFlag[d.Flag] = d
	}

	if d, ok := byFlag["-m"]; !ok || d.Alias != "--model" || !d.HasValue || d.Group != "模型" {
		t.Errorf("unexpected -m: %+v", d)
	}
	if d, ok := byFlag["-ngl"]; !ok || d.Group != "GPU" {
		t.Errorf("unexpected -ngl: %+v", d)
	}
	if d, ok := byFlag["-c"]; !ok || d.Group != "上下文" {
		t.Errorf("unexpected -c: %+v", d)
	}
	if d, ok := byFlag["--temp"]; !ok || d.Group != "采样" {
		t.Errorf("unexpected --temp: %+v", d)
	}
	if d, ok := byFlag["--port"]; !ok || d.Group != "服务器" {
		t.Errorf("unexpected --port: %+v", d)
	}
	if d, ok := byFlag["-t"]; !ok || d.Group != "性能" {
		t.Errorf("unexpected -t: %+v", d)
	}
	if d, ok := byFlag["--cpu-mask"]; !ok || d.Group != "性能" {
		t.Errorf("--cpu-mask should be 性能, got %+v", d)
	}
	if d, ok := byFlag["--no-mmap"]; !ok || d.HasValue {
		t.Errorf("unexpected --no-mmap: %+v", d)
	}
}
