package sources

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Matcher decides whether a path should be skipped while walking sources.
type Matcher struct {
	exact    []string
	patterns []*regexp.Regexp
	base     []*regexp.Regexp
}

// NewMatcher builds a matcher from absolute paths or glob patterns.
// Plain paths match the file itself or anything beneath that directory.
// Globs support *, ?, [...], and ** (any number of directories). A glob
// without a slash (e.g. "*.zwc") matches against the file's base name.
func NewMatcher(patterns []string) *Matcher {
	m := &Matcher{}
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.ContainsAny(p, "*?[") {
			m.exact = append(m.exact, filepath.Clean(p))
			continue
		}
		re, err := globToRegexp(p)
		if err != nil {
			continue
		}
		if strings.Contains(p, "/") {
			m.patterns = append(m.patterns, re)
		} else {
			m.base = append(m.base, re)
		}
	}
	return m
}

// Match reports whether path is ignored.
func (m *Matcher) Match(path string) bool {
	if m == nil {
		return false
	}
	path = filepath.Clean(path)
	for _, e := range m.exact {
		if path == e || strings.HasPrefix(path, e+string(filepath.Separator)) {
			return true
		}
	}
	for _, re := range m.patterns {
		if re.MatchString(path) {
			return true
		}
	}
	base := filepath.Base(path)
	for _, re := range m.base {
		if re.MatchString(base) {
			return true
		}
	}
	return false
}

func globToRegexp(glob string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); i++ {
		c := glob[i]
		switch c {
		case '*':
			if i+1 < len(glob) && glob[i+1] == '*' {
				i++
				if i+1 < len(glob) && glob[i+1] == '/' {
					i++
					b.WriteString("(?:.*/)?")
				} else {
					b.WriteString(".*")
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '[':
			end := strings.IndexByte(glob[i:], ']')
			if end < 0 {
				b.WriteString(`\[`)
				continue
			}
			class := glob[i+1 : i+end]
			if strings.HasPrefix(class, "!") {
				class = "^" + class[1:]
			}
			b.WriteString("[" + class + "]")
			i += end
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	// A pattern matching a directory also matches everything beneath it.
	b.WriteString("(?:/.*)?$")
	return regexp.Compile(b.String())
}
