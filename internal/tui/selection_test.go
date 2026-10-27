package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestSelectionSpanAcrossRows(t *testing.T) {
	sel := selection{start: point{2, 1}, end: point{4, 3}}
	if _, _, ok := sel.span(0, 10); ok {
		t.Fatal("row before the selection must be untouched")
	}
	if from, to, ok := sel.span(1, 10); !ok || from != 2 || to != 10 {
		t.Fatalf("first row span = %d..%d %v", from, to, ok)
	}
	if from, to, ok := sel.span(2, 10); !ok || from != 0 || to != 10 {
		t.Fatalf("middle row span = %d..%d %v", from, to, ok)
	}
	if from, to, ok := sel.span(3, 10); !ok || from != 0 || to != 5 {
		t.Fatalf("last row span = %d..%d %v", from, to, ok)
	}
	if _, _, ok := sel.span(4, 10); ok {
		t.Fatal("row after the selection must be untouched")
	}
}

func TestSelectionNormalizesBackwardsDrag(t *testing.T) {
	sel := selection{start: point{5, 4}, end: point{1, 2}}
	if from, to, ok := sel.span(3, 20); !ok || from != 0 || to != 20 {
		t.Fatalf("middle span = %d..%d %v", from, to, ok)
	}
	if from, to, ok := sel.span(2, 20); !ok || from != 1 || to != 20 {
		t.Fatalf("first span = %d..%d %v", from, to, ok)
	}
}

func TestSelectionTextAndHighlight(t *testing.T) {
	rows := []string{"alpha line", "bravo line", "charlie line"}
	sel := selection{start: point{6, 0}, end: point{4, 1}}
	if got := sel.text(rows, 20); got != "line\nbravo" {
		t.Fatalf("text = %q", got)
	}
	marked := sel.highlight(rows, 20)
	if !strings.Contains(marked[0], "\x1b[7m") || !strings.Contains(marked[1], "\x1b[7m") {
		t.Fatal("selected cells must be reversed")
	}
	if marked[2] != rows[2] {
		t.Fatal("untouched rows must keep their content")
	}
	if !strings.HasPrefix(ansi.Strip(marked[1]), "bravo") {
		t.Fatalf("highlight changed the row text: %q", ansi.Strip(marked[1]))
	}
}
