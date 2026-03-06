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
	tableStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	hrStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
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

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			out = append(out, codeBlockStyle.Render(line))
			continue
		}

		if inCodeBlock {
			out = append(out, codeBlockStyle.Render(line))
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			out = append(out, hrStyle.Render("────────────────────"))
			continue
		}

		if strings.HasPrefix(trimmed, "|") {
			var tableLines []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
				tableLines = append(tableLines, lines[i])
				i++
			}
			i-- // back up for the outer loop increment
			out = append(out, renderTable(tableLines)...)
			continue
		}

		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			out = append(out, headingStyle.Render(line))
			continue
		}

		line = renderInline(line)
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

var (
	tableHeaderStyle = lipgloss.NewStyle().Bold(true)
	tableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func renderTable(lines []string) []string {
	var rows [][]string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "|")
		cells := strings.Split(line, "|")
		for j := range cells {
			cells[j] = strings.TrimSpace(cells[j])
		}
		rows = append(rows, cells)
	}

	if len(rows) == 0 {
		return nil
	}

	var dataRows [][]string
	for _, row := range rows {
		isSep := true
		for _, cell := range row {
			cleaned := strings.Trim(cell, "-: ")
			if cleaned != "" {
				isSep = false
				break
			}
		}
		if !isSep {
			dataRows = append(dataRows, row)
		}
	}

	if len(dataRows) == 0 {
		return nil
	}

	numCols := len(dataRows[0])
	widths := make([]int, numCols)
	for _, row := range dataRows {
		for j := 0; j < len(row) && j < numCols; j++ {
			visible := stripMarkdown(row[j])
			if len(visible) > widths[j] {
				widths[j] = len(visible)
			}
		}
	}

	var borderParts []string
	for _, w := range widths {
		borderParts = append(borderParts, strings.Repeat("─", w+2))
	}
	topBorder := tableBorderStyle.Render("┌" + strings.Join(borderParts, "┬") + "┐")
	midBorder := tableBorderStyle.Render("├" + strings.Join(borderParts, "┼") + "┤")
	botBorder := tableBorderStyle.Render("└" + strings.Join(borderParts, "┴") + "┘")
	sep := tableBorderStyle.Render("│")

	var out []string
	out = append(out, topBorder)

	for i, row := range dataRows {
		if i > 0 {
			out = append(out, midBorder)
		}
		var parts []string
		for j := 0; j < numCols; j++ {
			cell := ""
			if j < len(row) {
				cell = row[j]
			}
			visible := stripMarkdown(cell)
			pad := strings.Repeat(" ", widths[j]-len(visible))
			styled := renderInline(cell)
			padded := " " + styled + pad + " "
			if i == 0 {
				parts = append(parts, tableHeaderStyle.Render(padded))
			} else {
				parts = append(parts, padded)
			}
		}
		out = append(out, sep+strings.Join(parts, sep)+sep)
	}

	out = append(out, botBorder)
	return out
}

func stripMarkdown(s string) string {
	s = stripDelim(s, "**")
	s = stripDelim(s, "`")
	s = stripDelim(s, "*")
	s = stripDelim(s, "_")
	return s
}

func stripDelim(s, delim string) string {
	return strings.ReplaceAll(s, delim, "")
}

func renderInline(s string) string {
	s = renderDelimited(s, "**", boldStyle)
	s = renderDelimited(s, "`", codeStyle)
	s = renderDelimited(s, "*", italicStyle)
	s = renderDelimited(s, "_", italicStyle)
	return s
}

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
