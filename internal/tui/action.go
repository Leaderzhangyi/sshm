package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"sshm/internal/config"
	"sshm/internal/ssh"
)

func (m Model) viewAction() string {
	c := m.cfg.Connections[m.selConn]
	port := c.Port
	if port == 0 {
		port = 22
	}

	header := styleTitle.Render(" " + c.Name + " ")
	info := []string{
		fmtKV("分组", c.Group),
		fmtKV("地址", fmt.Sprintf("%s@%s:%d", c.User, c.Host, port)),
		fmtKV("认证", c.Auth),
	}
	if c.ProxyJump != "" {
		info = append(info, fmtKV("跳板", c.ProxyJump))
	}

	actions := []string{
		styleKey.Render(" [1] ") + styleMuted.Render("SSH 终端"),
		styleKey.Render(" [2] ") + styleMuted.Render("上传文件 → 远端"),
		styleKey.Render(" [3] ") + styleMuted.Render("下载文件 ← 远端"),
		"",
		styleKey.Render(" [Esc] ") + styleMuted.Render("返回"),
	}

	body := header + "\n\n" +
		strings.Join(info, "\n") + "\n\n" +
		styleSectionHeader.Render("操作") + "\n" +
		strings.Join(actions, "\n")

	return stylePanel.Render(body) + "\n"
}

func (m Model) viewDelete() string {
	c := m.cfg.Connections[m.selConn]
	body := styleDanger.Render("⚠  确认删除以下连接？") + "\n\n" +
		fmtKV("名称", c.Name) + "\n" +
		fmtKV("地址", fmt.Sprintf("%s@%s:%d", c.User, c.Host, c.Port)) + "\n\n" +
		styleKey.Render("[y]") + styleMuted.Render(" 确认删除  ") +
		styleKey.Render("[n/Esc]") + styleMuted.Render(" 取消")
	return stylePanel.Render(body) + "\n"
}

func (m *Model) handleActionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "0":
		m.page = pageList
	case "1":
		conn := m.cfg.Connections[m.selConn]
		return m, tea.ExecProcess(ssh.BuildCmd(&conn), func(err error) tea.Msg {
			return msgSSHDone{err}
		})
	case "2":
		m.page = pageTransfer
		m.xferMode = "upload"
		m.xferLocal, m.xferRemote, m.xferStatus = "", "", ""
		m.xferPercent = 0
		m.inputs = makeXferInputs()
		m.inputIdx = 0
		m.inputs[0].Focus()
	case "3":
		m.page = pageTransfer
		m.xferMode = "download"
		m.xferLocal, m.xferRemote, m.xferStatus = "", "", ""
		m.xferPercent = 0
		m.inputs = makeXferInputs()
		m.inputIdx = 0
		m.inputs[0].Focus()
	}
	return m, nil
}

func (m *Model) handleDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		name := m.cfg.Connections[m.selConn].Name
		idx := m.selConn
		m.cfg.Connections = append(m.cfg.Connections[:idx], m.cfg.Connections[idx+1:]...)
		_ = config.Save(m.cfg)
		m.order = buildOrder(m.cfg)
		if m.cursor >= len(m.order) && m.cursor > 0 {
			m.cursor--
		}
		m.page = pageList
		m.setStatus("已删除: "+name, true)
	case "n", "N", "esc":
		m.page = pageList
	}
	return m, nil
}
