// Package sources discovers aliases, functions, and executables that witch
// can resolve.
package sources

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/imdevan/witch/internal/domain"
)

// maxFunctionLines caps how far a function body is scanned for its closing brace.
const maxFunctionLines = 500

var (
	// function name, function name(), function name {, function name() {
	reFunctionKeyword = regexp.MustCompile(`^\s*function\s+([^\s(){}=;#$]+)\s*(\(\s*\))?\s*([{(].*)?$`)
	// name() {, name () {, name()
	rePosixFunction = regexp.MustCompile(`^\s*([A-Za-z0-9_][A-Za-z0-9_.:+@-]*)\s*\(\s*\)\s*([{(].*)?$`)
	reAliasLine     = regexp.MustCompile(`^\s*alias\s+`)
	reDecoration    = regexp.MustCompile(`^[\s#=\-*_~+/\\|.]*$`)
)

// ParseFile parses a shell file for alias and function definitions.
func ParseFile(path string) ([]domain.Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f, path)
}

// Parse reads shell source and returns alias and function entries.
// file is recorded on every entry as the defining location.
func Parse(r io.Reader, file string) ([]domain.Entry, error) {
	lines, err := readLines(r)
	if err != nil {
		return nil, err
	}

	var entries []domain.Entry
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if reAliasLine.MatchString(line) {
			for _, entry := range parseAliasLine(trimmed) {
				entry.File = file
				entry.Line = i + 1
				entry.Definition = trimmed
				entries = append(entries, entry)
			}
			continue
		}

		if name, ok := matchFunction(line); ok {
			body := functionBody(lines, i)
			description := trailingComment(lines[i])
			if description == "" {
				description = precedingComment(lines, i)
			}
			entries = append(entries, domain.Entry{
				Name:        name,
				Type:        domain.TypeFunction,
				Value:       body,
				Description: description,
				Definition:  body,
				File:        file,
				Line:        i + 1,
			})
		}
	}
	return entries, nil
}

func readLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func matchFunction(line string) (string, bool) {
	if m := reFunctionKeyword.FindStringSubmatch(line); m != nil {
		return m[1], true
	}
	if m := rePosixFunction.FindStringSubmatch(line); m != nil {
		return m[1], true
	}
	return "", false
}

// parseAliasLine parses `alias [-g|-s|-r] name=value [name=value...] [# comment]`.
func parseAliasLine(line string) []domain.Entry {
	words, comment := shellWords(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "alias")))
	kind := domain.AliasRegular
	flagsDone := false
	var entries []domain.Entry
	for _, w := range words {
		if !flagsDone && strings.HasPrefix(w.raw, "-") {
			if w.raw == "--" {
				flagsDone = true
				continue
			}
			if strings.Contains(w.raw, "g") {
				kind = domain.AliasGlobal
			}
			if strings.Contains(w.raw, "s") {
				kind = domain.AliasSuffix
			}
			continue
		}
		flagsDone = true

		eq := strings.Index(w.raw, "=")
		if eq <= 0 {
			// `alias name` prints an alias; it does not define one.
			continue
		}
		rawName := w.raw[:eq]
		_, value, _ := strings.Cut(w.value, "=")
		for _, name := range expandBraces(unquote(rawName)) {
			// Names built at runtime (e.g. alias ${name}=...) can't be resolved statically.
			if name == "" || strings.ContainsAny(name, "$`()") {
				continue
			}
			entries = append(entries, domain.Entry{
				Name:        name,
				Type:        domain.TypeAlias,
				AliasKind:   kind,
				Value:       value,
				Description: comment,
			})
		}
	}
	return entries
}

// word is a shell word with both its raw source text and its unquoted value.
type word struct {
	raw   string
	value string
}

// shellWords splits a command line into words, honoring quotes and
// backslash escapes. An unquoted # at the start of a word begins a comment,
// which is returned separately. Parsing stops at an unquoted ; or &&.
func shellWords(s string) ([]word, string) {
	var (
		words   []word
		raw     strings.Builder
		value   strings.Builder
		inWord  bool
		quote   rune
		escaped bool
	)
	flush := func() {
		if inWord {
			words = append(words, word{raw: raw.String(), value: value.String()})
		}
		raw.Reset()
		value.Reset()
		inWord = false
	}

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if escaped {
			raw.WriteRune(r)
			value.WriteRune(r)
			escaped = false
			continue
		}
		switch {
		case quote != 0:
			raw.WriteRune(r)
			if r == quote {
				quote = 0
				continue
			}
			if r == '\\' && quote == '"' && i+1 < len(runes) {
				next := runes[i+1]
				if next == '"' || next == '\\' || next == '$' || next == '`' {
					raw.WriteRune(next)
					value.WriteRune(next)
					i++
					continue
				}
			}
			value.WriteRune(r)
		case r == '\\':
			inWord = true
			raw.WriteRune(r)
			escaped = true
		case r == '\'' || r == '"':
			inWord = true
			quote = r
			raw.WriteRune(r)
		case r == ' ' || r == '\t':
			flush()
		case r == '#' && !inWord:
			flush()
			return words, strings.TrimSpace(string(runes[i+1:]))
		case r == ';' || (r == '&' && i+1 < len(runes) && runes[i+1] == '&'):
			flush()
			return words, ""
		default:
			inWord = true
			raw.WriteRune(r)
			value.WriteRune(r)
		}
	}
	flush()
	return words, ""
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// expandBraces expands shell brace lists: "{js,ts}x" -> ["jsx", "tsx"].
func expandBraces(s string) []string {
	open := strings.Index(s, "{")
	if open < 0 {
		return []string{s}
	}
	depth := 0
	closeIdx := -1
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				closeIdx = i
			}
		}
		if closeIdx >= 0 {
			break
		}
	}
	if closeIdx < 0 {
		return []string{s}
	}
	inner := s[open+1 : closeIdx]
	options := splitTopLevel(inner)
	if len(options) < 2 {
		return []string{s}
	}
	prefix, suffix := s[:open], s[closeIdx+1:]
	seen := map[string]bool{}
	var out []string
	for _, opt := range options {
		for _, expanded := range expandBraces(prefix + opt + suffix) {
			if !seen[expanded] {
				seen[expanded] = true
				out = append(out, expanded)
			}
		}
	}
	return out
}

func splitTopLevel(s string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

// functionBody returns the source of the function starting at line idx,
// through its matching closing brace or paren. If the body cannot be
// delimited, only the definition line is returned.
func functionBody(lines []string, idx int) string {
	depth := 0
	opened := false
	var closer, opener rune
	var st scanState
	end := -1

	for i := idx; i < len(lines) && i < idx+maxFunctionLines && end < 0; i++ {
		line := lines[i]
		if i == idx {
			// Skip past the "name()" so its parens don't count.
			if p := strings.Index(line, ")"); p >= 0 && rePosixFunction.MatchString(line) {
				line = line[p+1:]
			} else if m := reFunctionKeyword.FindStringSubmatchIndex(line); m != nil && m[6] >= 0 {
				line = line[m[6]:]
			} else if m != nil {
				line = ""
			}
		}
		st.comment = false
		for _, r := range line {
			if !st.step(r) {
				continue
			}
			if !opened {
				if r == '{' || r == '(' {
					opener = r
					closer = map[rune]rune{'{': '}', '(': ')'}[r]
					opened = true
					depth = 1
				} else if r != ' ' && r != '\t' {
					return lines[idx]
				}
				continue
			}
			switch r {
			case opener:
				depth++
			case closer:
				depth--
				if depth == 0 {
					end = i
				}
			}
			if end >= 0 {
				break
			}
		}
	}
	if end < 0 {
		return lines[idx]
	}
	return strings.Join(lines[idx:end+1], "\n")
}

// scanState tracks quoting while scanning shell source rune by rune.
type scanState struct {
	quote   rune
	escaped bool
	comment bool
	prev    rune
}

// step advances the state and reports whether r is significant
// (unquoted, unescaped, not in a comment).
func (s *scanState) step(r rune) bool {
	defer func() { s.prev = r }()
	if s.comment {
		return false
	}
	if s.escaped {
		s.escaped = false
		return false
	}
	if r == '\\' && s.quote != '\'' {
		s.escaped = true
		return false
	}
	if s.quote != 0 {
		if r == s.quote {
			s.quote = 0
		}
		return false
	}
	switch r {
	case '\'', '"':
		s.quote = r
		return false
	case '#':
		if s.prev == 0 || s.prev == ' ' || s.prev == '\t' || s.prev == ';' {
			s.comment = true
			return false
		}
	}
	return true
}

// trailingComment returns the text of an unquoted # comment on the line.
func trailingComment(line string) string {
	var st scanState
	for i, r := range line {
		st.step(r)
		if st.comment {
			return strings.TrimSpace(line[i+1:])
		}
	}
	return ""
}

// precedingComment joins the contiguous comment lines directly above idx,
// skipping shebangs and decorative separator lines.
func precedingComment(lines []string, idx int) string {
	var parts []string
	for i := idx - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "#!") {
			break
		}
		if reDecoration.MatchString(trimmed) {
			if len(parts) > 0 {
				break
			}
			continue
		}
		parts = append([]string{strings.TrimSpace(strings.TrimLeft(trimmed, "#"))}, parts...)
	}
	return strings.Join(parts, " ")
}
