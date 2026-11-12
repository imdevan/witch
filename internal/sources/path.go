package sources

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/imdevan/witch/internal/domain"
)

// PathExecutables walks the directories in pathEnv (a $PATH-style list) in
// order and returns every executable found. Like `which -a`, all matches
// are kept; any executable whose name already appeared in an earlier
// directory is marked Shadowed. Symlinks are resolved into Target.
func PathExecutables(pathEnv string) []domain.Entry {
	var (
		entries []domain.Entry
		seen    = map[string]bool{}
		dirs    = map[string]bool{}
	)
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			dir = "."
		}
		dir = filepath.Clean(dir)
		if dirs[dir] {
			continue
		}
		dirs[dir] = true

		items, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Name() < items[j].Name() })
		for _, item := range items {
			full := filepath.Join(dir, item.Name())
			info, err := os.Stat(full) // follows symlinks
			if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
				continue
			}
			entry := domain.Entry{
				Name:     item.Name(),
				Type:     domain.TypeExecutable,
				File:     full,
				Shadowed: seen[item.Name()],
			}
			if item.Type()&fs.ModeSymlink != 0 {
				if target, err := filepath.EvalSymlinks(full); err == nil && target != full {
					entry.Target = target
				}
			}
			seen[item.Name()] = true
			entries = append(entries, entry)
		}
	}
	return entries
}
