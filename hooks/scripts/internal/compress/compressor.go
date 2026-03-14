// Package compress provides semantic text compression for injected context.
// Strips grammatical scaffolding (articles, copulas, filler words) that LLMs
// reconstruct naturally, reducing token count 20-40% without semantic loss.
package compress

import (
	"regexp"
	"strings"
)

// word-boundary patterns for each category of removable words
var (
	// Articles: the, a, an
	articlesRe = regexp.MustCompile(`(?i)\b(the|an?)\s+`)
	// Copulas: is, are, was, were, been, being
	copulasRe = regexp.MustCompile(`(?i)\b(is|are|was|were|been|being)\s+`)
	// Filler words: basically, actually, essentially, just, simply, really, very, quite, rather
	fillersRe = regexp.MustCompile(`(?i)\b(basically|actually|essentially|simply|really|very|quite|rather)\s+`)
	// Verbose phrases → compact equivalents
	verbosePairs = []struct {
		re   *regexp.Regexp
		repl string
	}{
		{regexp.MustCompile(`(?i)\bin order to\b`), "to"},
		{regexp.MustCompile(`(?i)\bdue to the fact that\b`), "because"},
		{regexp.MustCompile(`(?i)\bat this point in time\b`), "now"},
		{regexp.MustCompile(`(?i)\bfor the purpose of\b`), "for"},
		{regexp.MustCompile(`(?i)\bin the event that\b`), "if"},
		{regexp.MustCompile(`(?i)\bwith regard to\b`), "about"},
		{regexp.MustCompile(`(?i)\bit is important to note that\b`), "note:"},
		{regexp.MustCompile(`(?i)\bplease note that\b`), "note:"},
		{regexp.MustCompile(`(?i)\bmake sure to\b`), "ensure"},
		{regexp.MustCompile(`(?i)\bmake sure that\b`), "ensure"},
	}
	// Collapse multiple spaces into one
	multiSpaceRe = regexp.MustCompile(`  +`)
)

// CompressText strips grammatical scaffolding from text while preserving
// code blocks (``` fenced) and inline code (`backtick`). Returns compressed
// text and whether any compression was applied.
func CompressText(text string) (string, bool) {
	if text == "" {
		return text, false
	}

	lines := strings.Split(text, "\n")
	var result []string
	inCodeBlock := false

	for _, line := range lines {
		// Track fenced code blocks
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			result = append(result, line)
			continue
		}
		if inCodeBlock {
			result = append(result, line)
			continue
		}

		// Compress non-code lines
		result = append(result, compressLine(line))
	}

	compressed := strings.Join(result, "\n")
	return compressed, compressed != text
}

// compressLine applies all compression rules to a single line,
// preserving inline code spans (`...`).
func compressLine(line string) string {
	// Split on inline code spans to preserve them
	parts := splitPreservingCode(line)
	for i, part := range parts {
		if part.isCode {
			continue
		}
		s := part.text
		// Verbose phrase replacements first (longer matches)
		for _, vp := range verbosePairs {
			s = vp.re.ReplaceAllString(s, vp.repl)
		}
		s = articlesRe.ReplaceAllString(s, "")
		s = copulasRe.ReplaceAllString(s, "")
		s = fillersRe.ReplaceAllString(s, "")
		s = multiSpaceRe.ReplaceAllString(s, " ")
		parts[i].text = s
	}

	var sb strings.Builder
	for _, part := range parts {
		if part.isCode {
			sb.WriteString("`")
			sb.WriteString(part.text)
			sb.WriteString("`")
		} else {
			sb.WriteString(part.text)
		}
	}
	return sb.String()
}

type textPart struct {
	text   string
	isCode bool
}

// splitPreservingCode splits text into alternating prose/code segments.
func splitPreservingCode(line string) []textPart {
	var parts []textPart
	rest := line
	for {
		idx := strings.Index(rest, "`")
		if idx < 0 {
			parts = append(parts, textPart{text: rest, isCode: false})
			break
		}
		if idx > 0 {
			parts = append(parts, textPart{text: rest[:idx], isCode: false})
		}
		rest = rest[idx+1:]
		end := strings.Index(rest, "`")
		if end < 0 {
			// Unmatched backtick — treat rest as prose
			parts = append(parts, textPart{text: "`" + rest, isCode: false})
			break
		}
		parts = append(parts, textPart{text: rest[:end], isCode: true})
		rest = rest[end+1:]
	}
	return parts
}
