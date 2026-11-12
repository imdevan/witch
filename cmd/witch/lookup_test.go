package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imdevan/witch/internal/domain"
	"github.com/imdevan/witch/internal/testutil"
)

type cliEnv struct {
	bin1, bin2 string
	aliases    string
	functions  string
	configPath string
}

// setupCLI builds a fake PATH and a config with [[sources]], isolated from
// the user's real environment.
func setupCLI(t *testing.T, extraConfig string) cliEnv {
	t.Helper()
	configDir, _, _ := testutil.WithTempXDG(t)
	root := t.TempDir()
	env := cliEnv{
		bin1:      filepath.Join(root, "bin1"),
		bin2:      filepath.Join(root, "bin2"),
		aliases:   filepath.Join(root, "dotfiles", "aliases.zsh"),
		functions: filepath.Join(root, "dotfiles", "functions"),
	}
	for _, exe := range []string{
		filepath.Join(env.bin1, "ls"),
		filepath.Join(env.bin1, "cat"),
		filepath.Join(env.bin2, "ls"),
		filepath.Join(env.bin2, "fd"),
	} {
		writeFile(t, exe, "#!/bin/sh\n", 0o755)
	}
	writeFile(t, env.aliases, strings.Join([]string{
		`alias ll="ls -la" # long listing`,
		`alias -g G="| rg"`,
		`alias -s md=nvim`,
		`alias cat="bat"`,
	}, "\n")+"\n", 0o644)
	writeFile(t, filepath.Join(env.functions, "fcd.sh"), "# fuzzy cd\nfd() {\n  cd \"$(fzf)\"\n}\n", 0o644)
	t.Setenv("PATH", env.bin1+string(os.PathListSeparator)+env.bin2)

	env.configPath = filepath.Join(configDir, "witch", "config.toml")
	writeFile(t, env.configPath, extraConfig+`
[[sources]]
path = "`+env.aliases+`"

[[sources]]
path = "`+env.functions+`"
`, 0o644)
	return env
}

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func runWitch(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := newRootCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestLookupNonInteractive(t *testing.T) {
	env := setupCLI(t, "")
	alias := func(line int, def string) string { return fmt.Sprintf("%s:%d\n%s\n", env.aliases, line, def) }
	fdFunc := filepath.Join(env.functions, "fcd.sh") + ":2\nfd() {\n  cd \"$(fzf)\"\n}\n"

	tests := []struct {
		name    string
		args    []string
		stdout  string
		stderr  string
		missing bool
	}{
		{"executable first on PATH", []string{"ls"}, filepath.Join(env.bin1, "ls") + "\n", "", false},
		{"all shows shadowed PATH entries", []string{"-a", "ls"}, filepath.Join(env.bin1, "ls") + "\n" + filepath.Join(env.bin2, "ls") + "\n", "", false},
		{"alias", []string{"ll"}, alias(1, `alias ll="ls -la" # long listing`), "", false},
		{"global alias", []string{"G"}, alias(2, `alias -g G="| rg"`), "", false},
		{"suffix alias", []string{"md"}, alias(3, `alias -s md=nvim`), "", false},
		{"alias shadows executable", []string{"cat"}, alias(4, `alias cat="bat"`), "", false},
		{"all shows alias then executable", []string{"--all", "cat"}, alias(4, `alias cat="bat"`) + filepath.Join(env.bin1, "cat") + "\n", "", false},
		{"function shadows executable", []string{"fd"}, fdFunc, "", false},
		{"type filter skips shadowing entry", []string{"-t", "executable", "cat"}, filepath.Join(env.bin1, "cat") + "\n", "", false},
		{"type shorthand", []string{"-t", "bin", "fd"}, filepath.Join(env.bin2, "fd") + "\n", "", false},
		{"multiple types", []string{"-a", "-t", "alias,function", "fd", "ll"}, fdFunc + alias(1, `alias ll="ls -la" # long listing`), "", false},
		{"not found", []string{"nope"}, "", "witch: nope not found\n", true},
		{"filtered out counts as not found", []string{"-t", "alias", "ls"}, "", "witch: ls not found\n", true},
		{"mixed found and missing", []string{"ls", "nope", "ll"}, filepath.Join(env.bin1, "ls") + "\n" + alias(1, `alias ll="ls -la" # long listing`), "witch: nope not found\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runWitch(t, tt.args...)
			if tt.missing {
				if !errors.Is(err, errNotFound) {
					t.Fatalf("expected errNotFound, got %v", err)
				}
				if code, msg := exitCode(err); code != 1 || msg != "" {
					t.Fatalf("exit code = %d %q, want 1 and no message", code, msg)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v (stderr %q)", err, stderr)
			}
			if stdout != tt.stdout {
				t.Errorf("stdout = %q, want %q", stdout, tt.stdout)
			}
			if stderr != tt.stderr {
				t.Errorf("stderr = %q, want %q", stderr, tt.stderr)
			}
		})
	}
}

func TestLookupJSON(t *testing.T) {
	env := setupCLI(t, "")

	stdout, _, err := runWitch(t, "--json", "-a", "cat")
	if err != nil {
		t.Fatal(err)
	}
	var entries []domain.Entry
	if err := json.Unmarshal([]byte(stdout), &entries); err != nil {
		t.Fatalf("invalid json %q: %v", stdout, err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %+v", entries)
	}
	alias, exe := entries[0], entries[1]
	if alias.Type != domain.TypeAlias || alias.Value != "bat" || alias.File != env.aliases || alias.Line != 4 || alias.Shadowed {
		t.Errorf("unexpected alias entry %+v", alias)
	}
	if exe.Type != domain.TypeExecutable || exe.File != filepath.Join(env.bin1, "cat") || !exe.Shadowed {
		t.Errorf("unexpected executable entry %+v", exe)
	}

	t.Run("not found prints empty array", func(t *testing.T) {
		stdout, _, err := runWitch(t, "--json", "nope")
		if !errors.Is(err, errNotFound) {
			t.Fatalf("expected errNotFound, got %v", err)
		}
		if strings.TrimSpace(stdout) != "[]" {
			t.Fatalf("stdout = %q", stdout)
		}
	})
}

func TestNoArgsListsEntries(t *testing.T) {
	env := setupCLI(t, "")

	stdout, _, err := runWitch(t)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	names := map[string]string{}
	for _, line := range lines {
		fields := strings.Fields(line)
		names[fields[0]] = line
	}
	for _, want := range []string{"ll", "G", "md", "cat", "fd", "ls"} {
		if _, ok := names[want]; !ok {
			t.Errorf("missing %q in list output:\n%s", want, stdout)
		}
	}
	if len(lines) != 6 {
		t.Errorf("expected one row per winning name (6), got %d:\n%s", len(lines), stdout)
	}
	if !strings.Contains(names["ll"], env.aliases+":1") {
		t.Errorf("expected location in row, got %q", names["ll"])
	}

	t.Run("all includes shadowed", func(t *testing.T) {
		stdout, _, err := runWitch(t, "-a")
		if err != nil {
			t.Fatal(err)
		}
		if got := len(strings.Split(strings.TrimSpace(stdout), "\n")); got != 9 {
			t.Errorf("expected 9 rows, got %d:\n%s", got, stdout)
		}
	})
}

func TestIndexPathDisabled(t *testing.T) {
	setupCLI(t, "index_path = false\n")

	if _, _, err := runWitch(t, "ls"); !errors.Is(err, errNotFound) {
		t.Fatalf("expected ls to be unresolved without PATH indexing, got %v", err)
	}
	stdout, _, err := runWitch(t, "ll")
	if err != nil || !strings.HasSuffix(stdout, "\nalias ll=\"ls -la\" # long listing\n") {
		t.Fatalf("aliases should still resolve: %q %v", stdout, err)
	}
}

func TestNoSourcesBehavesLikeWhich(t *testing.T) {
	testutil.WithTempXDG(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "tool"), "#!/bin/sh\n", 0o755)
	t.Setenv("PATH", dir)

	stdout, _, err := runWitch(t, "tool")
	if err != nil || stdout != filepath.Join(dir, "tool")+"\n" {
		t.Fatalf("got %q %v", stdout, err)
	}
}

func TestConfigFlagOverride(t *testing.T) {
	env := setupCLI(t, "")
	override := filepath.Join(t.TempDir(), "other.toml")
	writeFile(t, override, "index_path = true\n[[sources]]\npath = \""+env.functions+"\"\n", 0o644)

	// The override replaces the global config, so aliases are not loaded.
	if _, _, err := runWitch(t, "--config", override, "ll"); !errors.Is(err, errNotFound) {
		t.Fatalf("expected ll to be missing with override config, got %v", err)
	}
	stdout, _, err := runWitch(t, "--config", override, "fd")
	if err != nil || !strings.Contains(stdout, "\nfd() {") {
		t.Fatalf("got %q %v", stdout, err)
	}
}

func TestMissingSourceWarns(t *testing.T) {
	setupCLI(t, "[[sources]]\npath = \"/definitely/not/here\"\n")

	stdout, stderr, err := runWitch(t, "ll")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(stdout, "\nalias ll=\"ls -la\" # long listing\n") {
		t.Errorf("stdout = %q", stdout)
	}
	if !strings.Contains(stderr, "warning: source /definitely/not/here") {
		t.Errorf("expected warning on stderr, got %q", stderr)
	}
}

func TestInvalidType(t *testing.T) {
	setupCLI(t, "")
	_, _, err := runWitch(t, "-t", "bogus", "ls")
	if err == nil || !strings.Contains(err.Error(), `unknown type "bogus"`) {
		t.Fatalf("expected unknown type error, got %v", err)
	}
}

func TestRootFlags(t *testing.T) {
	cmd := newRootCmd()
	for name, short := range map[string]string{"all": "a", "type": "t", "json": "", "interactive": "i"} {
		f := cmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("missing --%s flag", name)
			continue
		}
		if f.Shorthand != short {
			t.Errorf("--%s shorthand = %q, want %q", name, f.Shorthand, short)
		}
	}
}

func TestDisplayPathAbbreviatesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	tests := map[string]string{
		filepath.Join(home, ".aliases", "aliases.zsh") + ":447": "~/.aliases/aliases.zsh:447",
		home:                   "~",
		home + "other/file.sh": home + "other/file.sh",
		"/etc/zsh/zshrc:3":     "/etc/zsh/zshrc:3",
	}
	for in, want := range tests {
		if got := displayPath(in); got != want {
			t.Errorf("displayPath(%q) = %q, want %q", in, got, want)
		}
	}

	e := domain.Entry{Name: "asdf", Type: domain.TypeAlias, File: filepath.Join(home, ".aliases", "aliases.zsh"), Line: 447, Definition: `alias asdf="cowsay 'hi'"`}
	if got, want := formatEntry(e), "~/.aliases/aliases.zsh:447\nalias asdf=\"cowsay 'hi'\""; got != want {
		t.Errorf("formatEntry = %q, want %q", got, want)
	}
}
