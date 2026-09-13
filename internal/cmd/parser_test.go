package cmd

import (
	"reflect"
	"testing"

	"llama-manager/internal/runtime"
)

func testDefs() []runtime.ParamDef {
	return []runtime.ParamDef{
		{Flag: "-m", Alias: "--model", HasValue: true, Group: "模型"},
		{Flag: "-ngl", Alias: "--n-gpu-layers", HasValue: true, Group: "GPU"},
		{Flag: "-c", Alias: "--ctx-size", HasValue: true, Group: "上下文"},
		{Flag: "--no-mmap", HasValue: false, Group: "模型"},
	}
}

func TestTokenize(t *testing.T) {
	got := Tokenize(`llama-server.exe -m "C:\my models\q.gguf" -ngl 99`)
	want := []string{"llama-server.exe", "-m", `C:\my models\q.gguf`, "-ngl", "99"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Tokenize() = %#v, want %#v", got, want)
	}
}

func TestParse(t *testing.T) {
	cmdline := `C:\bin\llama-server.exe -m "D:\models\qwen 8b.gguf" --n-gpu-layers 99 -c=8192 --no-mmap --unknown-flag value --verbose`
	got := Parse(cmdline, testDefs())

	if got.Executable != `C:\bin\llama-server.exe` {
		t.Errorf("Executable = %q", got.Executable)
	}
	if got.ModelPath != `D:\models\qwen 8b.gguf` {
		t.Errorf("ModelPath = %q", got.ModelPath)
	}
	if got.Params["-ngl"] != "99" {
		t.Errorf("-ngl = %q (alias should normalize to primary)", got.Params["-ngl"])
	}
	if got.Params["-c"] != "8192" {
		t.Errorf("-c = %q (inline value)", got.Params["-c"])
	}
	if _, ok := got.Params["--no-mmap"]; !ok {
		t.Errorf("--no-mmap should be present")
	}
	wantExtra := []string{"--unknown-flag", "value", "--verbose"}
	if !reflect.DeepEqual(got.ExtraArgs, wantExtra) {
		t.Errorf("ExtraArgs = %#v, want %#v", got.ExtraArgs, wantExtra)
	}
}
