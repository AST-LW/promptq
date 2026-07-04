package tests

import (
	"strings"
	"testing"

	promptq "github.com/ast-lw/promptq/internal/promptq"
	"github.com/mattn/go-runewidth"
)

func TestFormatTable(t *testing.T) {
	rows := [][]string{{"abc/@abc", "abc {{xyz}}", "pqr"}, {"@pqr", "pqr", "pqr"}}
	want := "╭──────────┬─────────────┬──────╮\n" +
		"│ NAME     │ DESCRIPTION │ TAGS │\n" +
		"├──────────┼─────────────┼──────┤\n" +
		"│ abc/@abc │ abc {{xyz}} │ pqr  │\n" +
		"│ @pqr     │ pqr         │ pqr  │\n" +
		"╰──────────┴─────────────┴──────╯"
	if got := promptq.FormatTable([]string{"NAME", "DESCRIPTION", "TAGS"}, rows); got != want {
		t.Fatalf("FormatTable() =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatTableAlignsUnicodeByDisplayWidth(t *testing.T) {
	got := promptq.FormatTable([]string{"NAME", "DESCRIPTION"}, [][]string{
		{"@日本語", "wide"},
		{"@plain", "🙂"},
	})
	lines := strings.Split(got, "\n")
	wantWidth := runewidth.StringWidth(lines[0])
	for i, line := range lines {
		if width := runewidth.StringWidth(line); width != wantWidth {
			t.Fatalf("line %d width = %d, want %d: %q", i, width, wantWidth, line)
		}
	}
}
