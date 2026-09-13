package cmd

import (
	"path/filepath"
	"strings"

	"llama-manager/internal/runtime"
)

// ParsedCommand 是命令字符串解析后的结构化结果。
type ParsedCommand struct {
	Executable string            `json:"executable"`
	ModelPath  string            `json:"modelPath"`
	Params     map[string]string `json:"params"`
	ExtraArgs  []string          `json:"extraArgs"`
}

// Tokenize 按空格切分命令行，支持双引号包裹含空格的参数。
func Tokenize(s string) []string {
	tokens := []string{}
	var cur strings.Builder
	inQuotes := false
	hasToken := false

	flush := func() {
		if hasToken {
			tokens = append(tokens, cur.String())
			cur.Reset()
			hasToken = false
		}
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuotes = !inQuotes
			hasToken = true
		case (c == ' ' || c == '\t' || c == '\r' || c == '\n') && !inQuotes:
			flush()
		default:
			cur.WriteByte(c)
			hasToken = true
		}
	}
	flush()
	return tokens
}

// Parse 将完整命令行解析为模型路径、参数表与附加参数。
// 已知参数会被归一化为其主标志（例如 --n-gpu-layers -> -ngl），
// 未知参数原样保留到 ExtraArgs，以保证往返一致。
func Parse(command string, defs []runtime.ParamDef) ParsedCommand {
	tokens := Tokenize(command)
	result := ParsedCommand{Params: map[string]string{}, ExtraArgs: []string{}}

	lookup := map[string]runtime.ParamDef{}
	for _, d := range defs {
		lookup[d.Flag] = d
		if d.Alias != "" {
			lookup[d.Alias] = d
		}
	}

	start := 0
	if len(tokens) > 0 && looksLikeExecutable(tokens[0]) {
		result.Executable = tokens[0]
		start = 1
	}

	for i := start; i < len(tokens); i++ {
		token := tokens[i]
		if !isFlag(token) {
			result.ExtraArgs = append(result.ExtraArgs, token)
			continue
		}

		flag := token
		inlineVal := ""
		if idx := strings.Index(token, "="); idx > 0 {
			flag = token[:idx]
			inlineVal = token[idx+1:]
		}

		if flag == "-m" || flag == "--model" {
			v, ni := valueAt(tokens, i, inlineVal)
			i = ni
			if v != "" {
				result.ModelPath = v
			}
			continue
		}

		if def, ok := lookup[flag]; ok {
			if def.HasValue {
				v, ni := valueAt(tokens, i, inlineVal)
				i = ni
				result.Params[def.Flag] = v
			} else {
				result.Params[def.Flag] = ""
			}
			continue
		}

		// 未知标志：连同其可能的值一起放入附加参数。
		result.ExtraArgs = append(result.ExtraArgs, token)
		if inlineVal == "" {
			if v, ni := valueAt(tokens, i, ""); v != "" {
				i = ni
				result.ExtraArgs = append(result.ExtraArgs, v)
			}
		}
	}
	return result
}

// valueAt 返回标志对应的值以及消费后的下标。
func valueAt(tokens []string, i int, inline string) (string, int) {
	if inline != "" {
		return inline, i
	}
	if i+1 < len(tokens) && !isFlag(tokens[i+1]) {
		return tokens[i+1], i + 1
	}
	return "", i
}

func looksLikeExecutable(token string) bool {
	base := strings.ToLower(filepath.Base(token))
	return strings.HasSuffix(base, ".exe") || strings.HasPrefix(base, "llama-server")
}
