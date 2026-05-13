package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"sshm/internal/config"
)

var addFormLabels = []string{
	"显示名称", "主机地址", "端口", "用户名",
	"认证方式 (password/key)", "密码（可留空）", "私钥路径（key 认证时填）",
	"跳板机 user@host:port", "分组路径（如 隐私计算平台/stg1）",
}

var editFormLabels = addFormLabels

func (m Model) viewForm(title string, labels []string) string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(" "+title+" ") + "\n\n")

	for i, inp := range m.inputs {
		label := ""
		if i < len(labels) {
			label = labels[i]
		}
		focused := i == m.inputIdx
		lStyle := styleMuted
		if focused {
			lStyle = lipgloss.NewStyle().Foreground(clrAccent).Bold(true)
		}
		sb.WriteString(lStyle.Render(fmt.Sprintf("  %-30s", label)))
		sb.WriteString(inp.View())
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styleHint.Render("  [Tab] 下一个字段  [Enter] 保存  [Esc] 取消"))
	sb.WriteString("\n")
	return sb.String()
}

func (m Model) viewImportForm() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(" 导入 FinalShell 连接 ") + "\n\n")

	sb.WriteString("  " + styleMuted.Render("conn 目录默认位置：") + "\n")
	sb.WriteString("  " + styleMuted.Render("  Windows : %LOCALAPPDATA%\\FinalShell\\conn") + "\n")
	sb.WriteString("  " + styleMuted.Render("  macOS   : ~/Library/FinalShell/conn") + "\n")
	sb.WriteString("  " + styleMuted.Render("  Linux   : ~/.finalshell/conn") + "\n\n")

	focused := m.inputIdx == 0
	lStyle := styleMuted
	if focused {
		lStyle = lipgloss.NewStyle().Foreground(clrAccent).Bold(true)
	}
	sb.WriteString("  " + lStyle.Render("conn 目录路径: "))
	if len(m.inputs) > 0 {
		sb.WriteString(m.inputs[0].View())
	}
	sb.WriteString("\n\n")

	if m.importStatus != "" {
		sb.WriteString("  " + m.importStatus + "\n\n")
	}

	sb.WriteString("  " + styleHint.Render("[Enter] 开始导入  [Esc] 取消"))
	sb.WriteString("\n")
	return sb.String()
}

func (m *Model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.page = pageList
		return m, nil
	case "tab", "shift+tab":
		dir := 1
		if msg.String() == "shift+tab" {
			dir = -1
		}
		m.inputs[m.inputIdx].Blur()
		m.inputIdx = (m.inputIdx + dir + len(m.inputs)) % len(m.inputs)
		m.inputs[m.inputIdx].Focus()
		return m, nil
	case "enter":
		if m.page == pageAdd {
			return m.submitAdd()
		}
		if m.page == pageEdit {
			return m.submitEdit()
		}
	}

	var cmd tea.Cmd
	m.inputs[m.inputIdx], cmd = m.inputs[m.inputIdx].Update(msg)
	return m, cmd
}

func (m *Model) handleImportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.page = pageList
		return m, nil
	case "enter":
		return m.submitImport()
	}

	var cmd tea.Cmd
	if len(m.inputs) > 0 {
		m.inputs[0], cmd = m.inputs[0].Update(msg)
	}
	return m, cmd
}

func (m Model) submitAdd() (tea.Model, tea.Cmd) {
	vals := inputVals(m.inputs)
	port := 22
	if p, err := strconv.Atoi(vals[2]); err == nil && p > 0 {
		port = p
	}
	if vals[0] == "" || vals[1] == "" || vals[3] == "" {
		m.setStatus("名称/主机/用户名不能为空", false)
		return &m, nil
	}
	auth := vals[4]
	if auth != "key" {
		auth = "password"
	}
	m.cfg.Connections = append(m.cfg.Connections, config.Connection{
		Name:      vals[0],
		Host:      vals[1],
		Port:      port,
		User:      vals[3],
		Auth:      auth,
		Password:  vals[5],
		KeyPath:   vals[6],
		ProxyJump: vals[7],
		Group:     vals[8],
	})
	_ = config.Save(m.cfg)
	m.order = buildOrder(m.cfg)
	m.page = pageList
	m.setStatus("已添加: "+vals[0], true)
	return &m, nil
}

func (m Model) submitEdit() (tea.Model, tea.Cmd) {
	vals := inputVals(m.inputs)
	c := &m.cfg.Connections[m.editIdx]
	port := c.Port
	if p, err := strconv.Atoi(vals[2]); err == nil && p > 0 {
		port = p
	}
	auth := vals[4]
	if auth != "key" {
		auth = "password"
	}
	c.Name = orKeep(vals[0], c.Name)
	c.Host = orKeep(vals[1], c.Host)
	c.Port = port
	c.User = orKeep(vals[3], c.User)
	c.Auth = auth
	c.Password = vals[5]
	c.KeyPath = vals[6]
	c.ProxyJump = vals[7]
	c.Group = orKeep(vals[8], c.Group)
	_ = config.Save(m.cfg)
	m.order = buildOrder(m.cfg)
	m.page = pageList
	m.setStatus("已更新: "+c.Name, true)
	return &m, nil
}

func (m Model) submitImport() (tea.Model, tea.Cmd) {
	dir := ""
	if len(m.inputs) > 0 {
		dir = strings.TrimSpace(m.inputs[0].Value())
	}

	if dir == "" {
		m.importStatus = styleDanger.Render("✗ 请输入目录路径")
		return &m, nil
	}

	before := len(m.cfg.Connections)
	imported, skipped, err := config.ImportFinalShell(dir, m.cfg)
	if err != nil {
		m.importStatus = styleDanger.Render("✗ 导入失败: " + err.Error())
		return &m, nil
	}

	if imported > 0 {
		_ = config.Save(m.cfg)
		m.order = buildOrder(m.cfg)
		m.page = pageList
		msg := fmt.Sprintf("导入 %d 个连接（原 %d，现 %d）", imported, before, len(m.cfg.Connections))
		if skipped > 0 {
			msg += fmt.Sprintf("，%d 个已存在跳过", skipped)
		}
		m.setStatus(msg, true)
		return &m, nil
	}

	if skipped > 0 {
		m.importStatus = styleWarn.Render(fmt.Sprintf("⚠ %d 个连接已存在，无需重复导入", skipped))
	} else {
		m.importStatus = styleWarn.Render("⚠ 未找到有效连接，请检查目录路径是否正确")
	}
	return &m, nil
}

// Input factory functions

func newInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.PromptStyle = lipgloss.NewStyle().Foreground(clrAccent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(clrText)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(clrMuted)
	ti.Width = 40
	return ti
}

func newPasswordInput(placeholder string) textinput.Model {
	ti := newInput(placeholder)
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	return ti
}

func makeAddInputs() []textinput.Model {
	return []textinput.Model{
		newInput("服务名称"),
		newInput("192.168.1.10"),
		newInput("22"),
		newInput("root"),
		newInput("password"),
		newPasswordInput("留空则连接时提示输入"),
		newInput("/home/user/.ssh/id_rsa"),
		newInput("user@jumphost:22"),
		newInput("隐私计算平台/stg1"),
	}
}

func makeEditInputs(c *config.Connection) []textinput.Model {
	inputs := makeAddInputs()
	vals := []string{c.Name, c.Host, fmt.Sprintf("%d", c.Port), c.User,
		c.Auth, c.Password, c.KeyPath, c.ProxyJump, c.Group}
	for i := range inputs {
		if i < len(vals) {
			inputs[i].SetValue(vals[i])
		}
	}
	return inputs
}

func makeImportInputs() []textinput.Model {
	ti := newInput("例如: C:\\Users\\xxx\\AppData\\Local\\FinalShell\\conn")
	ti.Width = 60
	return []textinput.Model{ti}
}

func inputVals(inputs []textinput.Model) []string {
	out := make([]string, len(inputs))
	for i, inp := range inputs {
		out[i] = strings.TrimSpace(inp.Value())
	}
	return out
}

func orKeep(new, old string) string {
	if new == "" {
		return old
	}
	return new
}
