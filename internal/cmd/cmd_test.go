package cmd

import (
	"reflect"
	"testing"

	"llama-manager/internal/runtime"
)

func TestBuild(t *testing.T) {
	args := Build("C:\\models\\qwen.gguf", map[string]string{
		"-ngl":      "99",
		"-c":        "8192",
		"--no-mmap": "",
	}, []string{"--verbose"})

	want := []string{"-m", "C:\\models\\qwen.gguf", "--no-mmap", "-c", "8192", "-ngl", "99", "--verbose"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("Build() = %v, want %v", args, want)
	}
}

func TestExplain(t *testing.T) {
	defs := []runtime.ParamDef{
		{Flag: "-m", Alias: "--model", HasValue: true, Desc: "model path", Group: "模型"},
		{Flag: "-ngl", Alias: "--n-gpu-layers", HasValue: true, Desc: "gpu layers", Group: "GPU"},
		{Flag: "--no-mmap", HasValue: false, Desc: "disable mmap", Group: "模型"},
	}
	args := []string{"-m", "a.gguf", "-ngl", "99", "--no-mmap"}
	got := Explain(args, defs, "b10290")

	if len(got) != 3 {
		t.Fatalf("expected 3 explanations, got %d: %+v", len(got), got)
	}
	if got[0].Value != "a.gguf" || got[0].Desc != "model path" {
		t.Errorf("unexpected first: %+v", got[0])
	}
	if got[2].Flag != "--no-mmap" || got[2].Value != "" {
		t.Errorf("unexpected third: %+v", got[2])
	}
	if got[0].Note == "" {
		t.Errorf("expected version note on -m")
	}
}
