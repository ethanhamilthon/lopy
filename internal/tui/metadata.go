package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"lopy/internal/agent"
)

const markdownStyle = `{"document":{"margin":0},"heading":{"bold":true,"block_suffix":"\n"},"h1":{"prefix":"# ","color":"#ff79c6"},"h2":{"prefix":"## ","color":"#79d9ff"},"h3":{"prefix":"### ","color":"#ffd866"},"h4":{"color":"#a9dc76"},"h5":{"color":"#ab9df2"},"h6":{"color":"#fc9867"},"strong":{"bold":true},"emph":{"italic":true},"code":{"color":"#8aadff","prefix":"` + "`" + `","suffix":"` + "`" + `"},"code_block":{"margin":0},"list":{"level_indent":2},"item":{"block_prefix":"• "},"enumeration":{"block_prefix":". "},"block_quote":{"indent":1,"indent_token":"│ "},"link":{"underline":true,"color":"#8aadff","format":"\u2063{{.text}}\u2064"},"link_text":{"bold":true},"table":{"center_separator":"┼","column_separator":"│","row_separator":"─"}}`

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
func formatTokens(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fm", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprint(n)
}
func formatMetadata(meta *agent.Metadata, kind string) string {
	if kind != "assistant" && kind != "code" && kind != "output" && kind != "error" {
		return ""
	}
	duration := "?"
	if meta != nil {
		duration = formatDuration(meta.Duration)
	}
	if kind == "output" || kind == "error" {
		return "  " + duration
	}
	input, output, cache := "?", "?", "—"
	if meta != nil && meta.Usage != nil {
		u := meta.Usage
		input, output = formatTokens(u.Input), formatTokens(u.Output)
		if u.Input > 0 && u.Cached >= 0 && u.Cached <= u.Input {
			cache = fmt.Sprintf("%.0f%%", 100*float64(u.Cached)/float64(u.Input))
		}
	}
	return fmt.Sprintf("  ↑%s ↓%s · %s · cache %s", input, output, duration, cache)
}
func formatContext(n int64) string {
	if n >= 999500 {
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1000000), ".0") + "M"
	}
	if n >= 1000 {
		return fmt.Sprintf("%.0fK", float64(n)/1000)
	}
	return fmt.Sprint(n)
}
func (m model) bottomBar() string {
	settings := m.a.Settings()
	model := settings.Model
	if settings.Effort != "" {
		model += "/" + settings.Effort
	}
	ctx := "ctx ?"
	if m.contextInput != nil {
		ctx = "ctx ~" + formatContext(*m.contextInput)
	}
	left := settings.Provider + " · " + model + " · " + ctx
	if settings.Provider == "" || settings.Model == "" {
		left = "no model selected · " + ctx
	}
	if m.a.ReadOnly {
		left = "READ-ONLY · " + left
	}
	right := ansi.Strip(strings.ReplaceAll(m.cwd, "\n", " "))
	left = ansi.Strip(strings.ReplaceAll(left, "\n", " "))
	width := max(1, m.width)
	if ansi.StringWidth(left)+ansi.StringWidth(right)+1 > width {
		right = ansi.TruncateLeft(right, max(0, width/3), "…")
		left = ansi.Truncate(left, max(0, width-ansi.StringWidth(right)-1), "…")
	}
	gap := max(0, width-ansi.StringWidth(left)-ansi.StringWidth(right))
	return ansi.Truncate(left+strings.Repeat(" ", gap)+right, width, "")
}
