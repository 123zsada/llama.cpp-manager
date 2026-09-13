package instance

import (
	"os/exec"
	"time"
)

// 实例状态常量。
const (
	StatusStopped  = "stopped"
	StatusStarting = "starting"
	StatusRunning  = "running"
	StatusError    = "error"
)

// LlamaInstance 表示一个可启动的 llama.cpp 实例配置。
type LlamaInstance struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	RuntimeID string            `json:"runtimeId"`
	ModelPath string            `json:"modelPath"`
	Params    map[string]string `json:"params"`
	ExtraArgs []string          `json:"extraArgs"`
	WorkDir   string            `json:"workDir"`
	AutoStart bool              `json:"autoStart"`
	CreatedAt time.Time         `json:"createdAt"`
}

// InstanceState 表示实例的运行时状态（不持久化）。
type InstanceState struct {
	InstanceID string    `json:"instanceId"`
	Status     string    `json:"status"`
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"startedAt"`
	LastError  string    `json:"lastError"`

	cmd *exec.Cmd
}

// LogLine 表示一行日志。
type LogLine struct {
	Time   time.Time `json:"time"`
	Line   string    `json:"line"`
	Stream string    `json:"stream"`
}

// StatusEvent 是推送给前端的实例状态事件。
type StatusEvent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	PID    int    `json:"pid"`
	Error  string `json:"error"`
}

// LogEvent 是推送给前端的日志事件。
type LogEvent struct {
	ID     string `json:"id"`
	Line   string `json:"line"`
	Stream string `json:"stream"`
}
