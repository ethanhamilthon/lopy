package tui

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// point is a screen cell inside the timeline viewport.
type point struct{ x, y int }

type selection struct{ start, end point }

// span returns the selected cells of one row and whether the row is touched at all.
func (s selection) span(row, width int) (int, int, bool) {
	first, last := s.start, s.end
	if first.y > last.y || first.y == last.y && first.x > last.x {
		first, last = last, first
	}
	if row < first.y || row > last.y {
		return 0, 0, false
	}
	from, to := 0, width
	if row == first.y {
		from = first.x
	}
	if row == last.y {
		to = last.x + 1
	}
	return max(0, min(from, width)), max(0, min(to, width)), true
}

// cells returns the plain selected text of one row.
func (s selection) cells(line string, row, width int) (string, bool) {
	from, to, ok := s.span(row, width)
	if !ok {
		return "", false
	}
	plain := []rune(strings.TrimRight(ansi.Strip(line), " "))
	if from, to = min(from, len(plain)), min(to, len(plain)); from >= to {
		return "", false
	}
	return strings.TrimRight(string(plain[from:to]), " "), true
}

// highlight reverses the selected cells while leaving other rows untouched.
func (s selection) highlight(rows []string, width int) []string {
	out := make([]string, len(rows))
	for row, line := range rows {
		text, ok := s.cells(line, row, width)
		if !ok {
			out[row] = line
			continue
		}
		from := strings.Index(ansi.Strip(line), text)
		plain := ansi.Strip(line)
		out[row] = plain[:from] + "\x1b[7m" + text + "\x1b[0m" + plain[from+len(text):]
	}
	return out
}

// text extracts the selected rows as plain text.
func (s selection) text(rows []string, width int) string {
	var out []string
	for row, line := range rows {
		if text, ok := s.cells(line, row, width); ok {
			out = append(out, text)
		}
	}
	return strings.Join(out, "\n")
}

// copyText puts text on the system clipboard.
func copyText(text string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		name, args := "xclip", []string{"-selection", "clipboard"}
		if runtime.GOOS == "darwin" {
			name, args = "pbcopy", nil
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			name, args = "wl-copy", nil
		}
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			return clipboardResult{err: err}
		}
		return clipboardCopied{lines: strings.Count(text, "\n") + 1}
	}
}

type clipboardCopied struct{ lines int }

// mouseSelection handles drag selection; wheel and other buttons keep scrolling.
func (m *model) mouseSelection(msg tea.MouseMsg) tea.Cmd {
	if msg.Button != tea.MouseButtonLeft {
		return nil
	}
	cell := point{msg.X, msg.Y}
	switch msg.Action {
	case tea.MouseActionPress:
		m.sel = &selection{start: cell, end: cell}
	case tea.MouseActionMotion:
		if m.sel != nil {
			m.sel.end = cell
		}
	case tea.MouseActionRelease:
		if m.sel == nil {
			return nil
		}
		text := m.sel.text(strings.Split(m.viewport.View(), "\n"), m.viewport.Width)
		m.sel = nil
		if strings.TrimSpace(text) != "" {
			return copyText(text)
		}
	}
	return nil
}
