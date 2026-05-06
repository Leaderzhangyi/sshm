package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	clrPrimary = lipgloss.Color("#6B21A8")
	clrAccent  = lipgloss.Color("#0891B2")
	clrSuccess = lipgloss.Color("#059669")
	clrWarn    = lipgloss.Color("#D97706")
	clrDanger  = lipgloss.Color("#DC2626")
	clrMuted   = lipgloss.Color("#4B5563")
	clrText    = lipgloss.Color("#1F2937")
	clrBg      = lipgloss.Color("#FFFFFF")
	clrBgAlt   = lipgloss.Color("#F3F4F6")
)

var (
	styleLogo = lipgloss.NewStyle().
			Foreground(clrPrimary).
			Bold(true)

	styleTitle = lipgloss.NewStyle().
			Foreground(clrBg).
			Background(clrPrimary).
			Padding(0, 2).
			Bold(true)

	styleSectionHeader = lipgloss.NewStyle().
				Foreground(clrAccent).
				Bold(true)

	styleSelected = lipgloss.NewStyle().
			Foreground(clrBg).
			Background(clrPrimary).
			Bold(true).
			Padding(0, 1)

	styleNormal = lipgloss.NewStyle().
			Foreground(clrText).
			Padding(0, 1)

	styleGroup = lipgloss.NewStyle().
			Foreground(clrPrimary).
			Bold(true)

	styleConnName = lipgloss.NewStyle().
			Foreground(clrText).
			Bold(true)

	styleHost = lipgloss.NewStyle().
			Foreground(clrAccent)

	styleJump = lipgloss.NewStyle().
			Foreground(clrWarn)

	styleKey = lipgloss.NewStyle().
			Foreground(clrMuted)

	styleSuccess = lipgloss.NewStyle().
			Foreground(clrSuccess).
			Bold(true)

	styleDanger = lipgloss.NewStyle().
			Foreground(clrDanger).
			Bold(true)

	styleWarn = lipgloss.NewStyle().
			Foreground(clrWarn)

	styleMuted = lipgloss.NewStyle().
			Foreground(clrMuted)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrPrimary).
			Padding(1, 2)

	styleStatusBar = lipgloss.NewStyle().
			Background(clrBgAlt).
			Foreground(clrText).
			Padding(0, 1)

	styleHint = lipgloss.NewStyle().
			Foreground(clrMuted).
			Italic(true)

	styleProgress = lipgloss.NewStyle().
			Foreground(clrSuccess).
			Background(lipgloss.Color("#E5E7EB"))
)

func renderLogo() string {
	lines := []string{
		` ███████╗███████╗██╗  ██╗███╗   ███╗`,
		` ██╔════╝██╔════╝██║  ██║████╗ ████║`,
		` ███████╗███████╗███████║██╔████╔██║`,
		` ╚════██║╚════██║██╔══██║██║╚██╔╝██║`,
		` ███████║███████║██║  ██║██║ ╚═╝ ██║`,
		` ╚══════╝╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝`,
		``,
		styleMuted.Render(" SSH Connection Manager  数据二室开发v2.0"),
	}
	return styleLogo.Render(strings.Join(lines, "\n"))
}

func fmtKV(k, v string) string {
	return styleMuted.Render(fmt.Sprintf("  %-10s", k+":")) + styleNormal.Render(v)
}

func styleAccentBold(s string) string {
	return lipgloss.NewStyle().Foreground(clrAccent).Bold(true).Render(s)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

func renderProgressBar(width int, percent float64) string {
	width = max(width, 10)
	filled := max(0, min(int(float64(width-2)*percent), width-2))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-2-filled)
	return fmt.Sprintf("[%s] %.0f%%", styleProgress.Render(bar), percent*100)
}
