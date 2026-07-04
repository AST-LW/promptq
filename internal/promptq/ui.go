package promptq

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/mattn/go-runewidth"
)

var (
	stErr = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	stOK  = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true)
)

func ttyColor() bool { return isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsTerminal(os.Stdout.Fd()) }

// errMark colors a marker red on a terminal, plain otherwise.
func errMark(s string) string {
	if ttyColor() {
		return stErr.Render(s)
	}
	return s
}

// okMark colors a marker green on a terminal, plain otherwise.
func okMark(s string) string {
	if ttyColor() {
		return stOK.Render(s)
	}
	return s
}

func renderLogo() string {
	if !ttyColor() {
		return asciiLogo
	}
	lines := strings.Split(strings.Trim(asciiLogo, "\n"), "\n")
	var b strings.Builder
	b.WriteString("\n")
	for i, line := range lines {
		color := gradient[i%len(gradient)]
		b.WriteString(lipgloss.NewStyle().Foreground(color).Bold(true).Render(line))
		b.WriteString("\n")
	}
	return b.String()
}

func formatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = runewidth.StringWidth(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && runewidth.StringWidth(cell) > widths[i] {
				widths[i] = runewidth.StringWidth(cell)
			}
		}
	}

	border := func(left, middle, right, fill string) string {
		parts := make([]string, len(widths))
		for i, width := range widths {
			parts[i] = strings.Repeat(fill, width+2)
		}
		return left + strings.Join(parts, middle) + right
	}
	line := func(cells []string) string {
		parts := make([]string, len(widths))
		for i, width := range widths {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			parts[i] = " " + cell + strings.Repeat(" ", width-runewidth.StringWidth(cell)+1)
		}
		return "│" + strings.Join(parts, "│") + "│"
	}

	lines := []string{
		border("╭", "┬", "╮", "─"),
		line(headers),
		border("├", "┼", "┤", "─"),
	}
	for _, row := range rows {
		lines = append(lines, line(row))
	}
	lines = append(lines, border("╰", "┴", "╯", "─"))
	return strings.Join(lines, "\n")
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if runewidth.StringWidth(s) <= n {
		return s
	}
	if n <= 3 {
		return strings.Repeat(".", n)
	}
	return runewidth.Truncate(s, n-3, "") + "..."
}
