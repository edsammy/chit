package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	boldStyle      = lipgloss.NewStyle().Bold(true)
	italicStyle    = lipgloss.NewStyle().Italic(true)
	codeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	codeBlockStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	headingStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
)

func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var result []string
	for _, line := range strings.Split(s, "\n") {
		if len(line) <= width {
			result = append(result, line)
			continue
		}
		for len(line) > width {
			// Find last space before width.
			cut := strings.LastIndex(line[:width], " ")
			if cut <= 0 {
				cut = width
			}
			result = append(result, line[:cut])
			line = line[cut:]
			line = strings.TrimLeft(line, " ")
		}
		if len(line) > 0 {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func renderMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			out = append(out, codeBlockStyle.Render(line))
			continue
		}

		if inCodeBlock {
			out = append(out, codeBlockStyle.Render(line))
			continue
		}

		// Headings.
		trimmed := strings.TrimLeft(line, " ")
		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			out = append(out, headingStyle.Render(line))
			continue
		}

		line = renderInline(line)
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

func renderInline(s string) string {
	s = renderDelimited(s, "**", boldStyle)
	s = renderDelimited(s, "`", codeStyle)
	s = renderDelimited(s, "*", italicStyle)
	return s
}

// renderDelimited finds pairs of delimiter and applies the style.
func renderDelimited(s, delim string, style lipgloss.Style) string {
	for {
		start := strings.Index(s, delim)
		if start == -1 {
			break
		}
		end := strings.Index(s[start+len(delim):], delim)
		if end == -1 {
			break
		}
		end += start + len(delim)
		inner := s[start+len(delim) : end]
		styled := style.Render(inner)
		s = s[:start] + styled + s[end+len(delim):]
	}
	return s
}
