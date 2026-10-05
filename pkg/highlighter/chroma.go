package highlighter

import (
	"bytes"
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Highlight highlights text using chroma with the specified lexer name (e.g. "ini", "diff", "bash", "python")
func Highlight(content, lexerName string) string {
	lexer := lexers.Get(lexerName)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return content
	}

	return buf.String()
}

// HighlightUnit highlights systemd unit content
func HighlightUnit(content string) string {
	return Highlight(content, "ini")
}

// HighlightDiff highlights a diff string
func HighlightDiff(diff string) string {
	return Highlight(diff, "diff")
}

// SimpleDiff computes a line-based visual diff representation
func SimpleDiff(oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var sb strings.Builder
	sb.WriteString("--- Original\n+++ Generated\n")

	// If oldContent was empty, everything is +
	if strings.TrimSpace(oldContent) == "" {
		for _, line := range newLines {
			sb.WriteString("+ " + line + "\n")
		}
		return HighlightDiff(sb.String())
	}

	// Simple line matching
	max := len(oldLines)
	if len(newLines) > max {
		max = len(newLines)
	}

	for i := 0; i < max; i++ {
		var oldL, newL string
		if i < len(oldLines) {
			oldL = oldLines[i]
		}
		if i < len(newLines) {
			newL = newLines[i]
		}

		if oldL == newL {
			sb.WriteString("  " + oldL + "\n")
		} else {
			if i < len(oldLines) {
				sb.WriteString("- " + oldL + "\n")
			}
			if i < len(newLines) {
				sb.WriteString("+ " + newL + "\n")
			}
		}
	}

	return HighlightDiff(sb.String())
}

// FprintfHighlight writes highlighted content to an io.Writer
func FprintfHighlight(w io.Writer, content, lexerName string) {
	io.WriteString(w, Highlight(content, lexerName))
}
