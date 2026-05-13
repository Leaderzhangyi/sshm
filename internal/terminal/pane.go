package terminal

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"

	"sshm/internal/config"
	"sshm/internal/ssh"
)

type MsgOutput struct{ Data []byte }
type MsgDone struct{ Err error }

type Pane struct {
	Screen  *Screen
	ptmx    *os.File
	cmd     *exec.Cmd
	Running bool
}

func NewPane(c *config.Connection, w, h int) (*Pane, error) {
	args := ssh.BuildArgs(c)
	cmd := exec.Command("ssh", args...)
	cmd.Env = append(os.Environ(), "SSH_ASKPASS=")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(h),
		Cols: uint16(w),
	})
	if err != nil {
		return nil, fmt.Errorf("启动SSH终端失败: %w", err)
	}

	return &Pane{
		Screen:  NewScreen(w, h),
		ptmx:    ptmx,
		cmd:     cmd,
		Running: true,
	}, nil
}

func (p *Pane) ReadCmd() tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := p.ptmx.Read(buf)
		if err != nil {
			return MsgDone{Err: err}
		}
		return MsgOutput{Data: buf[:n]}
	}
}

func (p *Pane) Write(data []byte) error {
	_, err := p.ptmx.Write(data)
	return err
}

func (p *Pane) Resize(w, h int) {
	if p.ptmx != nil {
		pty.Setsize(p.ptmx, &pty.Winsize{
			Rows: uint16(h),
			Cols: uint16(w),
		})
	}
	p.Screen.Resize(w, h)
}

func (p *Pane) Close() {
	if !p.Running {
		return
	}
	p.Running = false
	if p.ptmx != nil {
		p.ptmx.Close()
	}
	if p.cmd.Process != nil {
		p.cmd.Process.Kill()
		p.cmd.Wait()
	}
}
