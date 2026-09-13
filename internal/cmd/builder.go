package cmd

import "sort"

// Build 根据模型路径、参数表与附加参数拼装命令行参数。
// 固定顺序：先 -m，其余参数按 flag 字典序，最后附加原始参数。
func Build(modelPath string, params map[string]string, extra []string) []string {
	args := []string{}
	if modelPath != "" {
		args = append(args, "-m", modelPath)
	}

	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "-m" || k == "--model" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := params[k]
		args = append(args, k)
		if v != "" {
			args = append(args, v)
		}
	}
	args = append(args, extra...)
	return args
}

// Quote 在需要时用引号包裹参数，便于展示。
func Quote(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		if needsQuote(a) {
			out += "\"" + a + "\""
		} else {
			out += a
		}
	}
	return out
}

func needsQuote(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '"' {
			return true
		}
	}
	return false
}
