package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"sshm/internal/config"
	"sshm/internal/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[错误] %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(
		tui.New(cfg),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
		os.Exit(1)
	}
}
