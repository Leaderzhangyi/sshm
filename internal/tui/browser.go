package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"sshm/internal/ssh"
)

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(KB))
	}
	return fmt.Sprintf("%d B", size)
}

func (m Model) renderPane(title, cwd string, files []ssh.FileEntry, cursor int, focused bool) string {
	w := (m.width - 4) / 2
	if w < 30 {
		w = 30
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(clrMuted).
		Width(w - 2)
	if focused {
		borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrAccent).
			Width(w - 2)
	}

	var sb strings.Builder
	sb.WriteString(styleMuted.Render(cwd))
	sb.WriteString("\n")

	visibleHeight := m.height - 8
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	start := 0
	if cursor >= visibleHeight {
		start = cursor - visibleHeight + 1
	}

	if start > 0 {
		sb.WriteString(styleMuted.Render("  ▲ 更多..."))
		sb.WriteString("\n")
	}

	for i := start; i < len(files) && i < start+visibleHeight; i++ {
		f := files[i]
		selected := i == cursor

		var icon, sizeStr string
		if f.IsDir {
			icon = "▸ "
			sizeStr = ""
		} else {
			icon = "  "
			sizeStr = formatSize(f.Size)
		}

		name := f.Name + func() string {
			if f.IsDir {
				return "/"
			}
			return ""
		}()

		line := icon + name

		if sizeStr != "" {
			pad := w - 6 - len(name) - len(sizeStr)
			if pad < 1 {
				pad = 1
			}
			line += strings.Repeat(" ", pad) + styleMuted.Render(sizeStr)
		}

		if selected {
			sb.WriteString(styleSelected.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	if start+visibleHeight < len(files) {
		sb.WriteString(styleMuted.Render("  ▼ 更多..."))
	}

	return borderStyle.Render(sb.String())
}

func (m Model) viewBrowser() string {
	c := m.cfg.Connections[m.selConn]
	port := c.Port
	if port == 0 {
		port = 22
	}

	header := styleTitle.Render(" 文件浏览器 ") + " " +
		styleMuted.Render(fmt.Sprintf("%s@%s:%d", c.User, c.Host, port))

	localFocused := m.browserFocus == "local"
	localPane := m.renderPane("本地文件", m.browserLocalCwd, m.browserLocalFiles, m.browserLocalCur, localFocused)
	remotePane := m.renderPane("远程文件", m.browserRemoteCwd, m.browserRemoteFiles, m.browserRemoteCur, !localFocused)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, localPane, remotePane)

	var statusLine string
	if m.browserTransferring {
		statusLine = styleWarn.Render("  ⟳ 传输中...") + "\n  " + renderProgressBar(40, m.browserPercent)
	} else if m.browserStatus != "" {
		statusLine = "  " + m.browserStatus
	}

	hints := styleHint.Render("  [Tab] 切换面板  [Enter] 进入目录  [u] 上传  [d] 下载  [Esc] 返回")

	return header + "\n\n" + panes + "\n" + statusLine + "\n" + hints + "\n"
}

func loadLocalDirCmd(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := ssh.ListLocalDir(path)
		return msgBrowserDir{side: "local", entries: entries, err: err}
	}
}

func (m *Model) browserEnterDir() tea.Cmd {
	var files []ssh.FileEntry
	var cwd string
	if m.browserFocus == "local" {
		files = m.browserLocalFiles
		cwd = m.browserLocalCwd
	} else {
		files = m.browserRemoteFiles
		cwd = m.browserRemoteCwd
	}

	cursor := m.browserLocalCur
	if m.browserFocus == "remote" {
		cursor = m.browserRemoteCur
	}

	if cursor >= len(files) {
		return nil
	}

	entry := files[cursor]
	if !entry.IsDir {
		return nil
	}

	newPath := cwd
	if entry.Name == ".." {
		newPath = filepath.Dir(cwd)
	} else {
		newPath = filepath.Join(cwd, entry.Name)
	}

	if m.browserFocus == "local" {
		m.browserLocalCwd = newPath
		return loadLocalDirCmd(newPath)
	}

	m.browserRemoteCwd = newPath
	conn := m.cfg.Connections[m.selConn]
	return func() tea.Msg {
		entries, err := ssh.ListRemoteDir(&conn, newPath)
		return msgBrowserDir{side: "remote", entries: entries, err: err}
	}
}

func (m *Model) browserUpload() tea.Cmd {
	if m.browserTransferring {
		return nil
	}
	if m.browserLocalCur >= len(m.browserLocalFiles) {
		return nil
	}

	entry := m.browserLocalFiles[m.browserLocalCur]
	if entry.Name == ".." {
		return nil
	}

	localPath := filepath.Join(m.browserLocalCwd, entry.Name)
	remotePath := filepath.Join(m.browserRemoteCwd, entry.Name)

	m.browserTransferring = true
	m.browserStatus = ""
	conn := m.cfg.Connections[m.selConn]

	return func() tea.Msg {
		err := ssh.Upload(&conn, localPath, remotePath, func(p float64) {})
		if err != nil {
			return msgBrowserTransferDone{err: err}
		}
		return msgBrowserTransferDone{msg: fmt.Sprintf("上传完成: %s → %s", localPath, remotePath)}
	}
}

func (m *Model) browserDownload() tea.Cmd {
	if m.browserTransferring {
		return nil
	}
	if m.browserRemoteCur >= len(m.browserRemoteFiles) {
		return nil
	}

	entry := m.browserRemoteFiles[m.browserRemoteCur]
	if entry.Name == ".." {
		return nil
	}

	remotePath := filepath.Join(m.browserRemoteCwd, entry.Name)
	localPath := filepath.Join(m.browserLocalCwd, entry.Name)

	m.browserTransferring = true
	m.browserStatus = ""
	conn := m.cfg.Connections[m.selConn]

	return func() tea.Msg {
		err := ssh.Download(&conn, remotePath, localPath, func(p float64) {})
		if err != nil {
			return msgBrowserTransferDone{err: err}
		}
		return msgBrowserTransferDone{msg: fmt.Sprintf("下载完成: %s → %s", remotePath, localPath)}
	}
}

func (m *Model) handleBrowserKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.page = pageAction
		return m, nil
	case "tab":
		if m.browserFocus == "local" {
			m.browserFocus = "remote"
		} else {
			m.browserFocus = "local"
		}
		return m, nil
	case "up", "k":
		if m.browserFocus == "local" && m.browserLocalCur > 0 {
			m.browserLocalCur--
		} else if m.browserFocus == "remote" && m.browserRemoteCur > 0 {
			m.browserRemoteCur--
		}
		return m, nil
	case "down", "j":
		if m.browserFocus == "local" && m.browserLocalCur < len(m.browserLocalFiles)-1 {
			m.browserLocalCur++
		} else if m.browserFocus == "remote" && m.browserRemoteCur < len(m.browserRemoteFiles)-1 {
			m.browserRemoteCur++
		}
		return m, nil
	case "enter":
		return m, m.browserEnterDir()
	case "u":
		return m, m.browserUpload()
	case "d":
		return m, m.browserDownload()
	}
	return m, nil
}

func initBrowser(m *Model) tea.Cmd {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}
	m.browserFocus = "local"
	m.browserLocalCwd = home
	m.browserRemoteCwd = "/"

	conn := m.cfg.Connections[m.selConn]
	return tea.Batch(
		loadLocalDirCmd(home),
		func() tea.Msg {
			entries, err := ssh.ListRemoteDir(&conn, "/")
			return msgBrowserDir{side: "remote", entries: entries, err: err}
		},
	)
}
