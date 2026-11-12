package sources

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxSourceFileSize skips files too large to plausibly be shell config.
const maxSourceFileSize = 2 << 20

// SourceFiles expands configured source paths (files or directories) into
// the list of shell files to parse, in order, without duplicates.
// Directories are walked recursively; hidden directories and binary files
// are skipped. Errors for missing or unreadable sources are returned
// alongside whatever files could be found.
func SourceFiles(paths []string, ignore *Matcher) ([]string, []error) {
	var (
		files []string
		errs  []error
		seen  = map[string]bool{}
	)
	add := func(path string) {
		key := path
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			key = resolved
		}
		if seen[key] || ignore.Match(path) || !IsTextFile(path) {
			return
		}
		seen[key] = true
		files = append(files, path)
	}

	for _, root := range paths {
		if strings.TrimSpace(root) == "" {
			continue
		}
		if ignore.Match(root) {
			continue
		}
		info, err := os.Stat(root)
		if err != nil {
			errs = append(errs, fmt.Errorf("source %s: %w", root, err))
			continue
		}
		if !info.IsDir() {
			add(root)
			continue
		}
		walkRoot := root
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			walkRoot = resolved
		}
		err = filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			// Report paths under the configured root, not the symlink target.
			display := filepath.Join(root, strings.TrimPrefix(path, walkRoot))
			if d.IsDir() {
				if path != walkRoot && (strings.HasPrefix(d.Name(), ".") || ignore.Match(display)) {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type()&fs.ModeSymlink != 0 {
				target, err := os.Stat(path)
				if err != nil || !target.Mode().IsRegular() {
					return nil
				}
			} else if !d.Type().IsRegular() {
				return nil
			}
			add(display)
			return nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("source %s: %w", root, err))
		}
	}
	return files, errs
}

// IsTextFile reports whether path is a reasonably sized, non-binary file.
func IsTextFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxSourceFileSize {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	return !bytes.Contains(buf[:n], []byte{0})
}
