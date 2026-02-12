package textnorm

import "strings"

// LeadingBlankLineTrimmer drops leading blank lines from a streaming text response.
// It buffers initial whitespace chunks until non-whitespace content appears.
type LeadingBlankLineTrimmer struct {
	seenContent bool
	pending     strings.Builder
}

// Push ingests one text delta and returns a normalized delta to emit.
func (t *LeadingBlankLineTrimmer) Push(delta string) string {
	if delta == "" {
		return ""
	}
	if t.seenContent {
		return delta
	}

	t.pending.WriteString(delta)
	pending := t.pending.String()
	if strings.TrimSpace(pending) == "" {
		return ""
	}

	normalized := TrimLeadingBlankLines(pending)
	t.pending.Reset()
	t.seenContent = true
	return normalized
}

// TrimLeadingBlankLines removes leading blank lines while preserving
// intentional leading spaces on the first non-empty line.
func TrimLeadingBlankLines(text string) string {
	i := 0
	for i < len(text) {
		j := i
		for j < len(text) && (text[j] == ' ' || text[j] == '\t') {
			j++
		}
		if j >= len(text) {
			return text
		}

		switch text[j] {
		case '\n':
			i = j + 1
		case '\r':
			if j+1 < len(text) && text[j+1] == '\n' {
				i = j + 2
			} else {
				i = j + 1
			}
		default:
			return text[i:]
		}
	}
	return text[i:]
}
