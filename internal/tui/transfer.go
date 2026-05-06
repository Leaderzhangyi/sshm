package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"sshm/internal/ssh"
)

func (m Model) viewTransfer() string {
	c := m.cfg.Connections[m.selConn]
	modeStr := "上传 (本地 → 远端)"
	if m.xferMode == "download" {
		modeStr = "下载 (远端 → 本地)"
	}

	var sb strings.Builder
	sb.WriteString(styleTitle.Render(" 文件传输 · "+modeStr+" ") + "\n\n")
	sb.WriteString(styleMuted.Render(fmt.Sprintf("  目标主机: %s@%s:%d\n\n", c.User, c.Host, c.Port)))

	labels := []string{"本地路径", "远端路径"}
	for i, inp := range m.inputs {
		focused := m.inputIdx == i
		lStyle := styleMuted
		if focused {
			lStyle = lipgloss.NewStyle().Foreground(clrAccent).Bold(true)
		}
		sb.WriteString(lStyle.Render(fmt.Sprintf("  %-12s", labels[i])))
		sb.WriteString(inp.View())
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	if m.xferRunning {
		sb.WriteString(styleWarn.Render("  ⟳ 传输中，请稍候...") + "\n")
		sb.WriteString("  " + renderProgressBar(40, m.xferPercent) + "\n")
	} else if m.xferStatus != "" {
		sb.WriteString("  " + m.xferStatus + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styleHint.Render("  [Tab] 切换字段  [Enter] 开始传输  [Esc] 返回"))
	sb.WriteString("\n")
	return sb.String()
}

func (m *Model) handleXferKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if !m.xferRunning {
			m.page = pageAction
		}
		return m, nil
	case "tab", "shift+tab":
		if m.xferRunning {
			return m, nil
		}
		dir := 1
		if msg.String() == "shift+tab" {
			dir = -1
		}
		m.inputs[m.inputIdx].Blur()
		m.inputIdx = (m.inputIdx + dir + len(m.inputs)) % len(m.inputs)
		m.inputs[m.inputIdx].Focus()
		return m, nil
	case "enter":
		if m.xferRunning {
			return m, nil
		}
		local := m.inputs[0].Value()
		remote := m.inputs[1].Value()
		if local == "" || remote == "" {
			m.xferStatus = styleDanger.Render("请填写本地路径和远端路径")
			return m, nil
		}
		m.xferRunning = true
		m.xferStatus = ""
		conn := m.cfg.Connections[m.selConn]
		mode := m.xferMode

		return m, func() tea.Msg {
			var err error
			var msg string
			if mode == "upload" {
				err = ssh.Upload(&conn, local, remote, func(p float64) {})
				if err == nil {
					msg = fmt.Sprintf("上传完成: %s → %s:%s", local, conn.Host, remote)
				}
			} else {
				err = ssh.Download(&conn, remote, local, func(p float64) {})
				if err == nil {
					msg = fmt.Sprintf("下载完成: %s:%s → %s", conn.Host, remote, local)
				}
			}
			return msgXferDone{err: err, msg: msg}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.inputIdx], cmd = m.inputs[m.inputIdx].Update(msg)
	return m, cmd
}
