package cmd

import (
	"strings"

	"llama-manager/internal/runtime"
)

// Explanation 描述单个参数的含义。
type Explanation struct {
	Flag  string `json:"flag"`
	Value string `json:"value"`
	Desc  string `json:"desc"`
	Note  string `json:"note"`
}

// knownIssues 记录与特定版本相关的已知问题。
var knownIssues = map[string]map[string]string{
	"b10290": {
		"-m":     "b10290 已知存在 token 重复 bug，建议升级或添加重复惩罚参数。",
		"--temp": "b10290 采样可能触发 token 重复，可适当提高重复惩罚。",
	},
}

// Explain 将参数列表解释为逐项说明。
func Explain(args []string, defs []runtime.ParamDef, buildTag string) []Explanation {
	lookup := map[string]runtime.ParamDef{}
	for _, d := range defs {
		lookup[d.Flag] = d
		if d.Alias != "" {
			lookup[d.Alias] = d
		}
	}

	issues := knownIssues[buildTag]
	explanations := []Explanation{}

	for i := 0; i < len(args); i++ {
		token := args[i]
		if !isFlag(token) {
			// 游离参数，作为上一条的值补充说明。
			if len(explanations) > 0 && explanations[len(explanations)-1].Value == "" {
				explanations[len(explanations)-1].Value = token
			}
			continue
		}

		flag := token
		value := ""
		if idx := strings.Index(token, "="); idx > 0 {
			flag = token[:idx]
			value = token[idx+1:]
		}

		def, ok := lookup[flag]
		desc := ""
		hasValue := false
		if ok {
			desc = def.Desc
			hasValue = def.HasValue
		}

		if value == "" && (hasValue || (!ok && i+1 < len(args) && !isFlag(args[i+1]))) {
			if i+1 < len(args) && !isFlag(args[i+1]) {
				value = args[i+1]
				i++
			}
		}

		note := ""
		if issues != nil {
			if n, ok := issues[flag]; ok {
				note = n
			}
		}
		if !ok && desc == "" {
			desc = "未在参数表中找到该参数"
		}

		explanations = append(explanations, Explanation{
			Flag:  flag,
			Value: value,
			Desc:  desc,
			Note:  note,
		})
	}
	return explanations
}

func isFlag(s string) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}
	// 负数不算 flag
	if len(s) > 1 && s[1] >= '0' && s[1] <= '9' {
		return false
	}
	return true
}
