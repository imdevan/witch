package sources

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/imdevan/witch/internal/domain"
)

func writeExec(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPathExecutables(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	writeExec(t, first, "tool")
	target := writeExec(t, second, "tool")
	writeExec(t, second, "other")
	if err := os.WriteFile(filepath.Join(second, "notexec"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(second, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "links")
	if err := os.MkdirAll(link, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(link, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing"), filepath.Join(link, "dangling")); err != nil {
		t.Fatal(err)
	}

	pathEnv := strings.Join([]string{first, filepath.Join(root, "nope"), second, first, link}, string(os.PathListSeparator))
	entries := PathExecutables(pathEnv)

	type row struct {
		name, file string
		shadowed   bool
	}
	var got []row
	for _, e := range entries {
		if e.Type != domain.TypeExecutable {
			t.Errorf("type = %q", e.Type)
		}
		got = append(got, row{e.Name, e.File, e.Shadowed})
	}
	want := []row{
		{"tool", filepath.Join(first, "tool"), false},
		{"other", filepath.Join(second, "other"), false},
		{"tool", filepath.Join(second, "tool"), true},
		{"linked", filepath.Join(link, "linked"), false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	resolvedTarget, _ := filepath.EvalSymlinks(target)
	if entries[3].Target != resolvedTarget {
		t.Errorf("symlink target = %q, want %q", entries[3].Target, resolvedTarget)
	}
	if entries[0].Target != "" {
		t.Errorf("regular file should have no target, got %q", entries[0].Target)
	}
}

func TestSourceFilesWalksAndIgnores(t *testing.T) {
	dir := filepath.Join("testdata", "functions")
	ignored := filepath.Join(dir, "shared", "ignored.sh")

	files, errs := SourceFiles([]string{dir, filepath.Join("testdata", "missing")}, NewMatcher([]string{ignored}))
	if len(errs) != 1 {
		t.Fatalf("expected one error for the missing source, got %v", errs)
	}
	want := []string{
		filepath.Join(dir, "fcd.sh"),
		filepath.Join(dir, "shared", "colors.sh"),
	}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("files = %v, want %v", files, want)
	}

	t.Run("glob ignore", func(t *testing.T) {
		files, _ := SourceFiles([]string{dir}, NewMatcher([]string{"colors.*", "**/ignored.sh"}))
		if !reflect.DeepEqual(files, []string{filepath.Join(dir, "fcd.sh")}) {
			t.Fatalf("files = %v", files)
		}
	})

	t.Run("binary files skipped and duplicates removed", func(t *testing.T) {
		tmp := t.TempDir()
		bin := filepath.Join(tmp, "functions.zwc")
		if err := os.WriteFile(bin, []byte{0, 1, 2, 3}, 0o644); err != nil {
			t.Fatal(err)
		}
		script := filepath.Join(tmp, "a.sh")
		if err := os.WriteFile(script, []byte("a() { :; }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		files, errs := SourceFiles([]string{tmp, script}, nil)
		if len(errs) != 0 || !reflect.DeepEqual(files, []string{script}) {
			t.Fatalf("files = %v errs = %v", files, errs)
		}
	})
}

func TestMatcher(t *testing.T) {
	m := NewMatcher([]string{"/home/u/.functions/old", "*.zwc", "/home/u/**/secret*.sh", "/etc/[ab]*.conf"})
	tests := map[string]bool{
		"/home/u/.functions/old":            true,
		"/home/u/.functions/old/x.sh":       true,
		"/home/u/.functions/older.sh":       false,
		"/anywhere/index.zwc":               true,
		"/home/u/a/b/secret_keys.sh":        true,
		"/home/u/secret.sh":                 true,
		"/home/u/public.sh":                 false,
		"/etc/a1.conf":                      true,
		"/etc/c1.conf":                      false,
		"/home/u/.functions/fcd.sh":         false,
		"/home/u/.functions/fcd.sh.zwc.bak": false,
	}
	for path, want := range tests {
		if got := m.Match(path); got != want {
			t.Errorf("Match(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestCollectPrecedence(t *testing.T) {
	root := t.TempDir()
	bin1 := filepath.Join(root, "bin1")
	bin2 := filepath.Join(root, "bin2")
	writeExec(t, bin1, "ls")
	writeExec(t, bin2, "ls")
	writeExec(t, bin1, "go")
	writeExec(t, bin1, "fd")

	first := filepath.Join(root, "first.zsh")
	second := filepath.Join(root, "second.zsh")
	if err := os.WriteFile(first, []byte("alias ls='ls -G'\nls() { command ls \"$@\"; }\nfd() { :; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("alias ls='ls --color' # colors\nalias -s go=nvim\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	idx, warnings := Collect(Options{
		Sources:   []string{first, second},
		IndexPath: true,
		PathEnv:   bin1 + string(os.PathListSeparator) + bin2,
	})
	if len(warnings) != 0 {
		t.Fatalf("warnings: %v", warnings)
	}

	type row struct {
		typ      domain.EntryType
		value    string
		file     string
		shadowed bool
	}
	rows := func(name string) []row {
		var out []row
		for _, e := range idx.Lookup(name) {
			out = append(out, row{e.Type, e.Value, e.File, e.Shadowed})
		}
		return out
	}

	t.Run("alias beats function beats PATH; last alias wins", func(t *testing.T) {
		want := []row{
			{domain.TypeAlias, "ls --color", second, false},
			{domain.TypeAlias, "ls -G", first, true},
			{domain.TypeFunction, `ls() { command ls "$@"; }`, first, true},
			{domain.TypeExecutable, "", filepath.Join(bin1, "ls"), true},
			{domain.TypeExecutable, "", filepath.Join(bin2, "ls"), true},
		}
		if got := rows("ls"); !reflect.DeepEqual(got, want) {
			t.Fatalf("got %+v\nwant %+v", got, want)
		}
	})

	t.Run("function shadows executable", func(t *testing.T) {
		got := rows("fd")
		if len(got) != 2 || got[0].typ != domain.TypeFunction || got[0].shadowed || !got[1].shadowed {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("suffix alias does not shadow executable", func(t *testing.T) {
		got := idx.Lookup("go")
		if len(got) != 2 || got[0].Type != domain.TypeExecutable || got[0].Shadowed {
			t.Fatalf("got %+v", got)
		}
		if got[1].AliasKind != domain.AliasSuffix || got[1].Shadowed {
			t.Fatalf("suffix alias should be its own winner, got %+v", got[1])
		}
	})

	t.Run("missing name", func(t *testing.T) {
		if got := idx.Lookup("nope"); len(got) != 0 {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("PATH indexing can be disabled", func(t *testing.T) {
		idx, _ := Collect(Options{Sources: []string{first}, IndexPath: false, PathEnv: bin1})
		for _, e := range idx.All() {
			if e.Type == domain.TypeExecutable {
				t.Fatalf("unexpected executable %+v", e)
			}
		}
	})
}
