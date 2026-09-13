package runtime

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ParamDef 描述一个命令行参数。
type ParamDef struct {
	Flag     string `json:"flag"`
	Alias    string `json:"alias"`
	Desc     string `json:"desc"`
	HasValue bool   `json:"hasValue"`
	Group    string `json:"group"`
}

var (
	// 将一行按两个以上空格拆分为若干列。
	splitColumns = regexp.MustCompile(`\s{2,}`)
	// 匹配标志列中的单个标志，可选带一个值占位符。
	flagFindRe = regexp.MustCompile(`(-{1,2}[A-Za-z0-9][A-Za-z0-9_-]*)(?:\s+([A-Za-z0-9_\[\]{}|.,:]+))?`)
)

// ParseHelp 解析 llama.cpp 的 --help 文本。
func ParseHelp(help string) []ParamDef {
	defs := []ParamDef{}
	seen := map[string]bool{}

	for _, raw := range strings.Split(help, "\n") {
		line := strings.TrimRight(raw, " \t\r")
		if !strings.HasPrefix(strings.TrimLeft(line, " \t"), "-") {
			continue
		}
		flagCol, desc := splitFlagAndDesc(strings.TrimSpace(line))
		def, ok := parseFlagColumn(flagCol, desc)
		if !ok {
			continue
		}
		if seen[def.Flag] {
			continue
		}
		seen[def.Flag] = true
		defs = append(defs, def)
	}
	return defs
}

// splitFlagAndDesc 将一行拆成「标志列」与「说明」。标志列可能被多个空格
// 分隔成多段（例如 "-m,    --model FNAME"），需要把以 '-' 开头的段落
// 全部并入标志列。
func splitFlagAndDesc(line string) (string, string) {
	parts := splitColumns.Split(line, -1)
	if len(parts) == 0 {
		return line, ""
	}
	idx := 1
	for idx < len(parts) && strings.HasPrefix(strings.TrimSpace(parts[idx]), "-") {
		idx++
	}
	flagCol := strings.Join(parts[:idx], " ")
	desc := ""
	if idx < len(parts) {
		desc = strings.TrimSpace(strings.Join(parts[idx:], " "))
	}
	return flagCol, desc
}

func parseFlagColumn(col, desc string) (ParamDef, bool) {
	var short, long string
	hasValue := false
	for _, m := range flagFindRe.FindAllStringSubmatch(col, -1) {
		flag := m[1]
		if m[2] != "" {
			hasValue = true
		}
		if strings.HasPrefix(flag, "--") {
			if long == "" {
				long = flag
			}
		} else if short == "" {
			short = flag
		}
	}
	primary := short
	if primary == "" {
		primary = long
	}
	if primary == "" {
		return ParamDef{}, false
	}
	alias := long
	if alias == primary {
		alias = short
	}
	return ParamDef{
		Flag:     primary,
		Alias:    alias,
		Desc:     desc,
		HasValue: hasValue,
		Group:    assignGroup(primary, alias, desc),
	}, true
}

var groupKeywords = []struct {
	group string
	flags []string
	words []string
}{
	{"模型", []string{"--model", "-m", "--lora", "--mmproj", "--vocab", "--alias", "--override-kv", "--grammar-file", "--chat-template", "--jinja", "--mmap", "--no-mmap", "--mlock"},
		[]string{"model", "lora", "adapter", "vocab", "tokenizer", "mmproj", "multimodal", "chat template"}},
	{"GPU", []string{"-ngl", "--gpu-layers", "--n-gpu-layers", "--tensor-split", "--main-gpu", "--split-mode", "--flash-attn", "--device", "--no-kv-offload", "--offload", "--gpu"},
		[]string{"gpu", "vram", "cuda", "vulkan", "offload", "tensor", "flash attention"}},
	{"上下文", []string{"-c", "--ctx-size", "--n-ctx", "-n", "--predict", "-b", "--batch-size", "--ubatch-size", "--keep", "--rope", "--rope-scaling", "--rope-freq", "--context-shift", "--no-context-shift"},
		[]string{"context", "ctx", "rope", "batch", "token", "predict", "sequence"}},
	{"采样", []string{"--temp", "--top-k", "--top-p", "--min-p", "--typical", "--repeat-penalty", "--repeat-last-n", "--presence-penalty", "--frequency-penalty", "--seed", "--samplers", "--mirostat", "--grammar", "--dynatemp", "--xtc"},
		[]string{"sampl", "temperature", "temp", "top-k", "top-p", "penalt", "seed", "mirostat", "grammar", "greedy"}},
	{"服务器", []string{"--host", "--port", "--threads-http", "--timeout", "-np", "--parallel", "--api-key", "--webui", "--no-webui", "--ssl", "--path", "--no-slots", "--slots", "--metrics", "--props", "--log-format", "--log-verbose", "--log-disable"},
		[]string{"server", "http", "host", "port", "listen", "webui", "api", "slot", "parallel", "ssl", "endpoint"}},
	{"性能", []string{"-t", "--threads", "--threads-batch", "--cpu-mask", "--cpu-range", "--cpu-strict", "--prio", "--numa", "--no-mmap", "--mlock", "--poll", "--numa"},
		[]string{"thread", "cpu", "numa", "core", "memory", "map", "lock", "priority", "performance"}},
}

func assignGroup(flag, alias, desc string) string {
	candidates := map[string]bool{strings.ToLower(flag): true}
	if alias != "" {
		candidates[strings.ToLower(alias)] = true
	}
	for _, g := range groupKeywords {
		for _, f := range g.flags {
			if candidates[strings.ToLower(f)] {
				return g.group
			}
		}
	}
	descLower := strings.ToLower(desc)
	for _, g := range groupKeywords {
		for _, w := range g.words {
			if strings.Contains(descLower, w) {
				return g.group
			}
		}
	}
	return "其他"
}

// DefaultParams 在 --help 解析失败时提供内置的常用参数表。
func DefaultParams() []ParamDef {
	return []ParamDef{
		{Flag: "-m", Alias: "--model", Desc: "模型文件路径", HasValue: true, Group: "模型"},
		{Flag: "-ngl", Alias: "--n-gpu-layers", Desc: "放入 VRAM 的层数", HasValue: true, Group: "GPU"},
		{Flag: "-c", Alias: "--ctx-size", Desc: "上下文大小", HasValue: true, Group: "上下文"},
		{Flag: "-t", Alias: "--threads", Desc: "线程数", HasValue: true, Group: "性能"},
		{Flag: "--host", Desc: "监听地址", HasValue: true, Group: "服务器"},
		{Flag: "--port", Desc: "监听端口", HasValue: true, Group: "服务器"},
		{Flag: "-np", Alias: "--parallel", Desc: "并行请求数", HasValue: true, Group: "服务器"},
		{Flag: "--no-mmap", Desc: "禁用内存映射", HasValue: false, Group: "模型"},
		{Flag: "--temp", Alias: "--temperature", Desc: "采样温度", HasValue: true, Group: "采样"},
		{Flag: "--top-k", Desc: "top-k 采样", HasValue: true, Group: "采样"},
		{Flag: "--top-p", Desc: "top-p 采样", HasValue: true, Group: "采样"},
		{Flag: "--seed", Desc: "随机种子", HasValue: true, Group: "采样"},
	}
}

// GetParams 通过执行 --help 获取参数定义，失败时回退到内置表。
func GetParams(exe string) []ParamDef {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, exe, "--help").CombinedOutput()
	if err != nil && len(out) == 0 {
		return DefaultParams()
	}
	defs := ParseHelp(string(out))
	if len(defs) < 5 {
		return DefaultParams()
	}
	return defs
}
