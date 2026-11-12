package sources

import (
	"fmt"
	"sort"

	"github.com/imdevan/witch/internal/domain"
)

// Options configures what Collect indexes.
type Options struct {
	// Sources are shell files or directories to parse for aliases/functions.
	Sources []string
	// Ignore holds paths or glob patterns to skip while reading sources.
	Ignore []string
	// IndexPath enables scanning $PATH for executables.
	IndexPath bool
	// PathEnv is the $PATH value to scan.
	PathEnv string
}

// Index holds every collected entry with shadowing resolved.
type Index struct {
	entries []domain.Entry
}

// Collect gathers aliases and functions from sources and executables from
// PATH into one index. Non-fatal problems (missing sources, unreadable
// files) are returned as warnings.
func Collect(opts Options) (*Index, []error) {
	var (
		entries  []domain.Entry
		warnings []error
	)
	files, errs := SourceFiles(opts.Sources, NewMatcher(opts.Ignore))
	warnings = append(warnings, errs...)
	for _, file := range files {
		parsed, err := ParseFile(file)
		if err != nil {
			warnings = append(warnings, fmt.Errorf("parse %s: %w", file, err))
			continue
		}
		entries = append(entries, parsed...)
	}
	if opts.IndexPath {
		entries = append(entries, PathExecutables(opts.PathEnv)...)
	}
	return NewIndex(entries), warnings
}

// NewIndex builds an index from entries in definition order (source files in
// configured order, then PATH order) and resolves shadowing.
func NewIndex(entries []domain.Entry) *Index {
	ranked := make([]rankedEntry, len(entries))
	for i, e := range entries {
		ranked[i] = rankedEntry{Entry: e, order: i}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.before(b)
	})

	out := make([]domain.Entry, len(ranked))
	winner := map[string]bool{}
	for i, r := range ranked {
		key := r.Name
		if r.AliasKind == domain.AliasSuffix {
			key = "suffix:" + r.Name
		}
		r.Entry.Shadowed = winner[key]
		winner[key] = true
		out[i] = r.Entry
	}
	return &Index{entries: out}
}

// All returns every entry, grouped by name with the winning entry first.
func (idx *Index) All() []domain.Entry {
	return append([]domain.Entry(nil), idx.entries...)
}

// Lookup returns all entries for an exact name, winner first:
// alias > function > PATH executable, then suffix aliases (which only
// apply to file names and never shadow commands).
func (idx *Index) Lookup(name string) []domain.Entry {
	i := sort.Search(len(idx.entries), func(i int) bool { return idx.entries[i].Name >= name })
	var out []domain.Entry
	for ; i < len(idx.entries) && idx.entries[i].Name == name; i++ {
		out = append(out, idx.entries[i])
	}
	return out
}

type rankedEntry struct {
	domain.Entry
	order int
}

func (a rankedEntry) group() int {
	if a.AliasKind == domain.AliasSuffix {
		return 3
	}
	return a.Type.Precedence()
}

// before reports whether a wins over b for the same name.
func (a rankedEntry) before(b rankedEntry) bool {
	ga, gb := a.group(), b.group()
	if ga != gb {
		return ga < gb
	}
	if a.Type == domain.TypeExecutable {
		// First directory on PATH wins.
		return a.order < b.order
	}
	// For shell definitions, the last one sourced wins.
	return a.order > b.order
}
