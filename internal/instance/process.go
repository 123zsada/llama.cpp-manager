package instance

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"llama-manager/internal/cmd"
	llamaruntime "llama-manager/internal/runtime"
)

// Start 启动指定实例。
func (m *Manager) Start(id string) error {
	m.mu.Lock()
	inst, ok := m.instances[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("实例不存在: %s", id)
	}
	if st, ok := m.states[id]; ok && (st.Status == StatusRunning || st.Status == StatusStarting) {
		m.mu.Unlock()
		return fmt.Errorf("实例 %s 已在运行", inst.Name)
	}
	rt, ok := m.reg.Get(inst.RuntimeID)
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("绑定的版本不存在: %s，请在实例编辑中重新选择版本", inst.RuntimeID)
	}
	instCopy := *inst
	m.stopping[id] = false
	m.mu.Unlock()

	args := cmd.Build(instCopy.ModelPath, instCopy.Params, instCopy.ExtraArgs)

	if err := llamaruntime.ValidateRuntime(rt); err != nil {
		m.appendLog(id, LogLine{Stream: "system", Line: err.Error()})
		m.updateState(id, StatusError, 0, err.Error())
		return err
	}

	if unknown := m.UnknownParams(rt, instCopy.Params); len(unknown) > 0 {
		err := fmt.Errorf("版本 %s 中不存在以下参数: %s；请在实例编辑中移除，或改为附加参数",
			rt.BuildTag, strings.Join(unknown, ", "))
		m.appendLog(id, LogLine{Stream: "system", Line: err.Error()})
		m.updateState(id, StatusError, 0, err.Error())
		return err
	}

	full := cmd.Quote(append([]string{rt.Executable}, args...))
	m.appendLog(id, LogLine{Stream: "system", Line: "命令: " + full})
	m.updateState(id, StatusStarting, 0, "")

	c := exec.Command(rt.Executable, args...)
	if instCopy.WorkDir != "" {
		c.Dir = instCopy.WorkDir
	} else {
		c.Dir = rt.WorkDir
	}
	c.SysProcAttr = sysProcAttr()

	stdout, err := c.StdoutPipe()
	if err != nil {
		m.updateState(id, StatusError, 0, err.Error())
		return err
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		m.updateState(id, StatusError, 0, err.Error())
		return err
	}

	if err := c.Start(); err != nil {
		m.updateState(id, StatusError, 0, err.Error())
		return err
	}

	m.mu.Lock()
	if st, ok := m.states[id]; ok {
		st.cmd = c
	}
	m.mu.Unlock()

	go m.streamLogs(id, stdout, "stdout")
	go m.streamLogs(id, stderr, "stderr")

	m.updateState(id, StatusRunning, c.Process.Pid, "")
	m.appendLog(id, LogLine{Stream: "system", Line: fmt.Sprintf("已启动: %s", rt.Executable)})

	go m.wait(id, c)
	return nil
}

// Stop 停止指定实例。
func (m *Manager) Stop(id string) error {
	m.mu.Lock()
	st, ok := m.states[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("实例不存在: %s", id)
	}
	c := st.cmd
	if c == nil || c.Process == nil || (st.Status != StatusRunning && st.Status != StatusStarting) {
		m.mu.Unlock()
		return nil
	}
	m.stopping[id] = true
	pid := c.Process.Pid
	m.mu.Unlock()

	killProcessTree(pid)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if m.GetState(id).Status == StatusStopped {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	m.updateState(id, StatusStopped, 0, "")
	return nil
}

// Restart 重启指定实例。
func (m *Manager) Restart(id string) error {
	if err := m.Stop(id); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	return m.Start(id)
}

func (m *Manager) wait(id string, c *exec.Cmd) {
	err := c.Wait()

	m.mu.Lock()
	stopping := m.stopping[id]
	m.stopping[id] = false
	if st, ok := m.states[id]; ok {
		st.cmd = nil
	}
	m.mu.Unlock()

	if stopping {
		m.appendLog(id, LogLine{Stream: "system", Line: "已停止"})
		m.updateState(id, StatusStopped, 0, "")
		return
	}
	if err != nil {
		m.appendLog(id, LogLine{Stream: "system", Line: "进程异常退出: " + err.Error()})
		m.updateState(id, StatusError, 0, err.Error())
		return
	}
	m.appendLog(id, LogLine{Stream: "system", Line: "进程已退出"})
	m.updateState(id, StatusStopped, 0, "")
}

func (m *Manager) streamLogs(id string, r io.Reader, stream string) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		m.appendLog(id, LogLine{Stream: stream, Line: scanner.Text()})
	}
}
