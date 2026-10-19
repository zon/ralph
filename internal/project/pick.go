package project

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/zon/ralph/internal/trailer"
)

// pickRefPrefix labels the reference line shown above an item in the picker
// prompt.
const pickRefPrefix = "ref: "

var (
	pickRefLineRe = regexp.MustCompile("(?i)(?:ralph[- ]?item|item|ref|hash)\\s*[:=]\\s*[\"'`]?([A-Za-z0-9]{7})[\"'`]?")
	pickRefBareRe = regexp.MustCompile("^[\"'`]?([A-Za-z0-9]{7})[\"'`]?$")
	pickTrailerRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*-([A-Za-z0-9]{7})$`)
	pickJSONRefRe = regexp.MustCompile(`(?i)"(?:ref|hash)"\s*:\s*"([A-Za-z0-9]{7})"`)
)

// PickRef returns the reference that identifies the item in the picker
// protocol: its completion hash. The reference is short, contains no quoting
// or escaping, and names the same text the completion trailer records.
func (it Item) PickRef() string {
	return it.Hash()
}

// PickRefLine renders the reference line shown above the item in the picker
// prompt.
func (it Item) PickRefLine() string {
	return pickRefPrefix + it.PickRef()
}

// DecodePickRef resolves the content the picker wrote to one of items. A
// report is expected to be a reference line, but a bare reference, a
// completion trailer, a JSON reference, or the item's text are accepted too.
// Accepting the text keeps a picker that reports what it sees from failing on
// formatting, and the escaped renderings of the text keep it from failing on
// the quoting a YAML source adds.
func DecodePickRef(items []Item, report string) (Item, bool) {
	candidates := pickCandidates(report)
	for _, candidate := range candidates {
		if item, ok := matchPickRef(items, candidate); ok {
			return item, true
		}
	}
	for _, candidate := range candidates {
		if item, ok := matchItemText(items, candidate); ok {
			return item, true
		}
	}
	return Item{}, false
}

// pickCandidates returns the strings in a picker report that may name an item:
// the whole report, each reference line, any JSON reference, and the report
// with a surrounding quote pair removed.
func pickCandidates(report string) []string {
	report = stripCodeFence(strings.TrimSpace(report))
	candidates := []string{report}
	for _, line := range strings.Split(report, "\n") {
		if ref, ok := parsePickRef(line); ok {
			candidates = append(candidates, ref)
		}
	}
	if match := pickJSONRefRe.FindStringSubmatch(report); match != nil {
		candidates = append(candidates, match[1])
	}
	if unquoted, ok := unquoteText(report); ok {
		candidates = append(candidates, unquoted)
	}
	return candidates
}

// stripCodeFence removes a surrounding Markdown code fence, which an agent
// sometimes adds around an output file's content.
func stripCodeFence(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) < 2 || !strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
		return s
	}
	lines = lines[1:]
	if last := len(lines) - 1; last >= 0 && strings.TrimSpace(lines[last]) == "```" {
		lines = lines[:last]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// parsePickRef extracts a reference from one line. It accepts a labelled line
// such as "ref: abc1234", a bare reference, and a completion trailer.
func parsePickRef(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if match := pickRefLineRe.FindStringSubmatch(line); match != nil {
		return match[1], true
	}
	if match := pickRefBareRe.FindStringSubmatch(line); match != nil {
		return match[1], true
	}
	if match := pickTrailerRe.FindStringSubmatch(line); match != nil {
		return match[1], true
	}
	return "", false
}

// matchPickRef returns the item whose completion hash is ref.
func matchPickRef(items []Item, ref string) (Item, bool) {
	for _, item := range items {
		if item.PickRef() == ref {
			return item, true
		}
	}
	return Item{}, false
}

// matchItemText returns the item whose text hashes to the candidate. Variants
// cover the item's pick text, its canonical YAML, and the quoted renderings a
// YAML source may escape.
func matchItemText(items []Item, candidate string) (Item, bool) {
	if candidate == "" {
		return Item{}, false
	}
	hash := trailer.Hash(candidate)
	for _, item := range items {
		for _, variant := range itemTextVariants(item) {
			if trailer.Hash(variant) == hash {
				return item, true
			}
		}
	}
	return Item{}, false
}

// itemTextVariants returns the renderings of an item's text a picker may copy:
// the text the picker was shown, the canonical YAML, and the double- and
// single-quoted forms a YAML source would hold, both with and without their
// surrounding quotes.
func itemTextVariants(item Item) []string {
	pickText := item.PickText()
	variants := []string{pickText, item.Text()}
	variants = append(variants, quotedVariants(strconv.Quote(pickText), `"`, `"`)...)
	singleQuoted := "'" + strings.ReplaceAll(pickText, "'", "''") + "'"
	variants = append(variants, quotedVariants(singleQuoted, "'", "'")...)
	return variants
}

// quotedVariants returns a quoted rendering and, when it differs from the
// unquoted text, the rendering with its surrounding quotes removed.
func quotedVariants(quoted, prefix, suffix string) []string {
	inner := strings.TrimSuffix(strings.TrimPrefix(quoted, prefix), suffix)
	if inner == quoted {
		return []string{quoted}
	}
	return []string{quoted, inner}
}

// unquoteText removes a matching pair of surrounding double or single quotes.
// YAML double-quoted escapes are decoded; single-quoted apostrophes are
// undoubled.
func unquoteText(s string) (string, bool) {
	if len(s) < 2 {
		return "", false
	}
	switch {
	case strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`):
		unquoted, err := strconv.Unquote(s)
		return unquoted, err == nil
	case strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'"):
		return strings.ReplaceAll(s[1:len(s)-1], "''", "'"), true
	}
	return "", false
}
