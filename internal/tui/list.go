package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewList() string {
	var sb strings.Builder

	sb.WriteString(renderLogo())
	sb.WriteString(styleMuted.Render(fmt.Sprintf("  (%d 连接)", len(m.cfg.Connections))))
	sb.WriteString("\n\n")

	if len(m.cfg.Connections) == 0 {
		sb.WriteString(styleHint.Render("  暂无连接  按 [a] 添加，或按 [i] 从 FinalShell 导入"))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString(m.renderConnectionList())
	}

	hints := []string{
		styleKey.Render("[↑↓/jk]") + styleMuted.Render(" 移动"),
		styleKey.Render("[Enter]") + styleMuted.Render(" 选择"),
		styleKey.Render("[a]") + styleMuted.Render(" 添加"),
		styleKey.Render("[e]") + styleMuted.Render(" 编辑"),
		styleKey.Render("[d]") + styleMuted.Render(" 删除"),
		styleKey.Render("[i]") + styleMuted.Render(" 导入"),
		styleKey.Render("[q]") + styleMuted.Render(" 退出"),
	}
	sb.WriteString(styleMuted.Render(strings.Repeat("─", 50)) + "\n")
	sb.WriteString(strings.Join(hints, "  ") + "\n")

	if m.status != "" {
		s := m.status
		if m.statusOK {
			s = styleSuccess.Render("✓ " + s)
		} else {
			s = styleDanger.Render("✗ " + s)
		}
		sb.WriteString(styleStatusBar.Render(s) + "\n")
	}
	return sb.String()
}

type listRow struct {
	isGroup bool
	depth   int
	label   string
	idx     int
}

func (m Model) buildListRows() []listRow {
	var rows []listRow
	seenGroups := map[string]bool{}

	for ci, connIdx := range m.order {
		c := m.cfg.Connections[connIdx]
		group := c.Group
		if group == "" {
			group = "默认"
		}
		parts := strings.Split(group, "/")
		for d := 1; d <= len(parts); d++ {
			key := strings.Join(parts[:d], "/")
			if !seenGroups[key] {
				seenGroups[key] = true
				rows = append(rows, listRow{isGroup: true, depth: d - 1, label: parts[d-1]})
			}
		}
		rows = append(rows, listRow{isGroup: false, depth: len(parts), idx: ci})
	}
	return rows
}

func (m Model) renderConnectionList() string {
	rows := m.buildListRows()

	maxVisible := m.height - 6
	if maxVisible < 5 {
		maxVisible = 5
	}

	selectedRowIdx := -1
	for i, r := range rows {
		if !r.isGroup && r.idx == m.cursor {
			selectedRowIdx = i
			break
		}
	}

	startIdx, endIdx := 0, len(rows)
	if len(rows) > maxVisible {
		halfVisible := maxVisible / 2
		if selectedRowIdx >= 0 {
			startIdx = selectedRowIdx - halfVisible
			if startIdx < 0 {
				startIdx = 0
			}
			endIdx = startIdx + maxVisible
			if endIdx > len(rows) {
				endIdx = len(rows)
				startIdx = endIdx - maxVisible
				if startIdx < 0 {
					startIdx = 0
				}
			}
		}
	}

	var sb strings.Builder

	if startIdx > 0 {
		sb.WriteString(styleMuted.Render("  ▲ 更多...") + "\n")
	}

	for i := startIdx; i < endIdx; i++ {
		r := rows[i]
		indent := strings.Repeat("  ", r.depth)
		if r.isGroup {
			sb.WriteString(indent)
			sb.WriteString(styleGroup.Render("▸ " + r.label))
			sb.WriteString("\n")
			continue
		}

		connIdx := m.order[r.idx]
		c := m.cfg.Connections[connIdx]
		port := c.Port
		if port == 0 {
			port = 22
		}

		isSelected := r.idx == m.cursor
		cursor := "  "
		if isSelected {
			cursor = styleAccentBold("❯ ")
		}

		nameW := 22
		name := truncate(c.Name, nameW)
		hostStr := fmt.Sprintf("%s@%s:%d", c.User, c.Host, port)

		if isSelected {
			paddedNamePlain := lipgloss.PlaceHorizontal(nameW, lipgloss.Left, name)
			line := fmt.Sprintf("%s%s  %s%s%s",
				indent+"❯ ",
				paddedNamePlain,
				hostStr,
				func() string {
					if c.ProxyJump != "" {
						return " [J]"
					}
					return ""
				}(),
				func() string {
					if c.Auth == "key" {
						return " [key]"
					}
					return ""
				}(),
			)
			sb.WriteString(styleSelected.Render(line))
		} else {
			jump := ""
			if c.ProxyJump != "" {
				jump = " " + styleJump.Render("[J]")
			}
			auth := ""
			if c.Auth == "key" {
				auth = " " + styleMuted.Render("[key]")
			}

			styledName := styleConnName.Render(name)
			paddedName := lipgloss.PlaceHorizontal(nameW, lipgloss.Left, styledName)
			line := fmt.Sprintf("%s%s%s  %s%s%s",
				indent, cursor,
				paddedName,
				styleHost.Render(hostStr),
				jump, auth,
			)
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	if endIdx < len(rows) {
		sb.WriteString(styleMuted.Render("  ▼ 更多...") + "\n")
	}

	return sb.String()
}

func (m *Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.order)-1 {
			m.cursor++
		}
	case "enter", " ":
		if len(m.order) > 0 {
			m.selConn = m.order[m.cursor]
			m.page = pageAction
		}
	case "a":
		m.page = pageAdd
		m.inputs = makeAddInputs()
		m.inputIdx = 0
		m.inputs[0].Focus()
	case "e":
		if len(m.order) > 0 {
			m.editIdx = m.order[m.cursor]
			m.page = pageEdit
			m.inputs = makeEditInputs(&m.cfg.Connections[m.editIdx])
			m.inputIdx = 0
			m.inputs[0].Focus()
		}
	case "d":
		if len(m.order) > 0 {
			m.selConn = m.order[m.cursor]
			m.page = pageDelete
		}
	case "i":
		m.page = pageImport
		m.inputs = makeImportInputs()
		m.inputIdx = 0
		m.inputs[0].Focus()
		m.importStatus = ""
	}
	return m, nil
}
