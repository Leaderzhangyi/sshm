// Package tui implements the terminal UI using BubbleTea.
package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"sshm/internal/config"
	"sshm/internal/ssh"
	"sshm/internal/terminal"
)

type page int

const (
	pageList page = iota
	pageAction
	pageAdd
	pageEdit
	pageDelete
	pageImport
	pageBrowser
	pageTerminal
)

type Model struct {
	cfg     *config.Config
	page    page
	cursor  int
	order   []int
	selConn int

	status   string
	statusOK bool

	inputs   []textinput.Model
	inputIdx int
	editIdx  int

	width  int
	height int

	// browser state
	browserFocus        string
	browserLocalCwd     string
	browserRemoteCwd    string
	browserLocalFiles   []ssh.FileEntry
	browserRemoteFiles  []ssh.FileEntry
	browserLocalCur     int
	browserRemoteCur    int
	browserStatus       string
	browserTransferring bool
	browserPercent      float64

	termPane *terminal.Pane

	importStatus string
}

func New(cfg *config.Config) Model {
	m := Model{cfg: cfg}
	m.order = buildOrder(cfg)
	return m
}

func (m Model) Init() tea.Cmd { return nil }

type msgStatus struct {
	text string
	ok   bool
}

type msgBrowserDir struct {
	side    string
	entries []ssh.FileEntry
	err     error
}

type msgBrowserTransferDone struct {
	err error
	msg string
}

type msgBrowserTransferProgress struct {
	percent float64
}

type msgSSHDone struct{ err error }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.termPane != nil {
			m.termPane.Resize(msg.Width, msg.Height-1)
		}

	case msgStatus:
		m.status, m.statusOK = msg.text, msg.ok

	case msgBrowserDir:
		if msg.err != nil {
			m.browserStatus = styleDanger.Render("✗ " + msg.err.Error())
			return m, nil
		}
		if msg.side == "local" {
			m.browserLocalFiles = msg.entries
			m.browserLocalCur = 0
		} else {
			m.browserRemoteFiles = msg.entries
			m.browserRemoteCur = 0
		}
		m.browserStatus = ""

	case msgBrowserTransferProgress:
		m.browserPercent = msg.percent

	case msgBrowserTransferDone:
		m.browserTransferring = false
		if msg.err != nil {
			m.browserStatus = styleDanger.Render("✗ 失败: " + msg.err.Error())
		} else {
			m.browserStatus = styleSuccess.Render("✓ " + msg.msg)
		}

	case terminal.MsgOutput:
		if m.termPane != nil && m.termPane.Running {
			m.termPane.Screen.Process(msg.Data)
			return m, m.termPane.ReadCmd()
		}

	case terminal.MsgDone:
		if m.termPane != nil {
			m.termPane.Close()
			m.termPane = nil
			m.page = pageAction
			m.setStatus("SSH 会话结束", true)
		}

	case msgSSHDone:
		m.page = pageList
		if msg.err != nil {
			m.setStatus("SSH 会话结束: "+msg.err.Error(), false)
		} else {
			m.setStatus("SSH 会话正常结束", true)
		}

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) setStatus(text string, ok bool) {
	m.status, m.statusOK = text, ok
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.page == pageTerminal {
		return m.handleTerminalKey(msg)
	}
	if m.page == pageAdd || m.page == pageEdit {
		return m.handleFormKey(msg)
	}
	if m.page == pageImport {
		return m.handleImportKey(msg)
	}
	if m.page == pageBrowser {
		return m.handleBrowserKey(msg)
	}
	if m.page == pageList {
		return m.handleListKey(msg)
	}
	if m.page == pageAction {
		return m.handleActionKey(msg)
	}
	if m.page == pageDelete {
		return m.handleDeleteKey(msg)
	}
	return m, nil
}

func (m Model) handleTerminalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "f10" {
		if m.termPane != nil {
			m.termPane.Close()
			m.termPane = nil
		}
		m.page = pageAction
		return m, nil
	}
	if m.termPane != nil && m.termPane.Running {
		m.termPane.Write(terminal.KeyToBytes(msg))
	}
	return m, nil
}

func (m Model) viewTerminal() string {
	if m.termPane == nil {
		return ""
	}
	c := m.cfg.Connections[m.selConn]
	port := c.Port
	if port == 0 {
		port = 22
	}
	info := fmt.Sprintf(" SSH: %s (%s@%s:%d) ", c.Name, c.User, c.Host, port)
	hint := " F10 返回 "
	pad := m.width - len(info) - len(hint)
	if pad < 0 {
		pad = 0
	}
	bar := styleStatusBar.Render(info + stringsRepeat(" ", pad) + styleMuted.Render(hint))
	content := m.termPane.Screen.String()
	return content + "\n" + bar
}

func stringsRepeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

func (m Model) View() string {
	switch m.page {
	case pageList:
		return m.viewList()
	case pageAction:
		return m.viewAction()
	case pageAdd:
		return m.viewForm("新建连接", addFormLabels)
	case pageEdit:
		return m.viewForm("编辑连接", editFormLabels)
	case pageImport:
		return m.viewImportForm()
	case pageDelete:
		return m.viewDelete()
	case pageBrowser:
		return m.viewBrowser()
	case pageTerminal:
		return m.viewTerminal()
	}
	return ""
}
