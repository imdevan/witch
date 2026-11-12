package sources

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/imdevan/witch/internal/domain"
)

func findAll(entries []domain.Entry, name string) []domain.Entry {
	var out []domain.Entry
	for _, e := range entries {
		if e.Name == name {
			out = append(out, e)
		}
	}
	return out
}

func findOne(t *testing.T, entries []domain.Entry, name string) domain.Entry {
	t.Helper()
	found := findAll(entries, name)
	if len(found) != 1 {
		t.Fatalf("expected exactly one entry named %q, got %d: %+v", name, len(found), found)
	}
	return found[0]
}

func TestParseAliasFixture(t *testing.T) {
	path := filepath.Join("testdata", "aliases.zsh")
	entries, err := ParseFile(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tests := []struct {
		name, value, description string
		kind                     domain.AliasKind
		line                     int
	}{
		{"..", "cd ..", "", domain.AliasRegular, 17},
		{"...", "cd ../..", "", domain.AliasRegular, 18},
		{"o", "open ./", "Open - in finder", domain.AliasRegular, 20},
		{"rf", "rm -rf", "Remove recursive", domain.AliasRegular, 21},
		{"gs", "git status", "", domain.AliasRegular, 22},
		{"git", "git_clone", "open git urls with custom function", domain.AliasSuffix, 24},
		{"G", "| rg", "grep output", domain.AliasGlobal, 25},
		{"T", "2>&1 | tee output.log", "copy output to output.log", domain.AliasGlobal, 26},
		{"C", "", "", domain.AliasGlobal, 27},
		{"hash", "echo '#not a comment'", "real comment", domain.AliasRegular, 28},
		{"indented", "echo indented", "", domain.AliasRegular, 29},
		{"ll", "ls -la", "", domain.AliasRegular, 31},
		{"la", "ls -A", "", domain.AliasRegular, 31},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := findOne(t, entries, tt.name)
			if e.Type != domain.TypeAlias {
				t.Errorf("type = %q, want alias", e.Type)
			}
			if e.Value != tt.value {
				t.Errorf("value = %q, want %q", e.Value, tt.value)
			}
			if e.Description != tt.description {
				t.Errorf("description = %q, want %q", e.Description, tt.description)
			}
			if e.AliasKind != tt.kind {
				t.Errorf("kind = %q, want %q", e.AliasKind, tt.kind)
			}
			if e.Line != tt.line {
				t.Errorf("line = %d, want %d", e.Line, tt.line)
			}
			if e.File != path {
				t.Errorf("file = %q, want %q", e.File, path)
			}
		})
	}

	t.Run("definition is the source line", func(t *testing.T) {
		if got := findOne(t, entries, "o").Definition; got != `alias o='open ./' # Open - in finder` {
			t.Errorf("definition = %q", got)
		}
		if got := findOne(t, entries, "indented").Definition; got != `alias indented="echo indented"` {
			t.Errorf("indented definition should be trimmed, got %q", got)
		}
		if got := findOne(t, entries, "ts").Definition; got != `alias -s {js,mjs,json,ts}="$EDITOR"` {
			t.Errorf("brace definition = %q", got)
		}
	})

	t.Run("redefinitions are all kept", func(t *testing.T) {
		rs := findAll(entries, "r")
		if len(rs) != 2 || rs[0].Line != 19 || rs[1].Line != 32 {
			t.Fatalf("expected r on lines 19 and 32, got %+v", rs)
		}
	})

	t.Run("commented aliases are skipped", func(t *testing.T) {
		if got := findAll(entries, "commented"); len(got) != 0 {
			t.Fatalf("expected no commented alias, got %+v", got)
		}
	})

	t.Run("brace suffix aliases expand", func(t *testing.T) {
		for _, ext := range []string{"js", "mjs", "json", "ts"} {
			e := findOne(t, entries, ext)
			if e.AliasKind != domain.AliasSuffix || e.Value != "$EDITOR" || e.Line != 23 {
				t.Errorf("%s: got %+v", ext, e)
			}
		}
	})

	t.Run("function in alias file", func(t *testing.T) {
		e := findOne(t, entries, "aliases")
		if e.Type != domain.TypeFunction || e.Line != 6 {
			t.Fatalf("got %+v", e)
		}
		if e.Definition != e.Value {
			t.Errorf("function definition should be its body")
		}
		if !strings.HasPrefix(e.Value, "aliases() {") || !strings.HasSuffix(e.Value, "\n}") {
			t.Errorf("unexpected body: %q", e.Value)
		}
		want := "Function wrapper to auto-source aliases after running alias commands This ensures new/updated aliases are immediately available in your shell"
		if e.Description != want {
			t.Errorf("description = %q", e.Description)
		}
	})
}

func TestParseFunctionFixture(t *testing.T) {
	entries, err := ParseFile(filepath.Join("testdata", "functions", "fcd.sh"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	tests := []struct {
		name        string
		line        int
		description string
		bodyEnd     string
	}{
		{"fzf_cd", 6, "Change directory with fzf", "cd \"$dir\"\n}"},
		{"keyword_brace", 13, "", "}\"\n}"},
		{"next_line_brace", 17, "", "echo next\n}"},
		{"posix_func", 22, "", `echo "one-liner"; }`},
		{"subshell_func", 24, "", "ls\n)"},
		{"open", 28, "Open files with xdg-open", "&\n}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := findOne(t, entries, tt.name)
			if e.Type != domain.TypeFunction {
				t.Fatalf("type = %q", e.Type)
			}
			if e.Line != tt.line {
				t.Errorf("line = %d, want %d", e.Line, tt.line)
			}
			if e.Description != tt.description {
				t.Errorf("description = %q, want %q", e.Description, tt.description)
			}
			if !strings.HasSuffix(e.Value, tt.bodyEnd) {
				t.Errorf("body %q does not end with %q", e.Value, tt.bodyEnd)
			}
		})
	}

	t.Run("alias inside function file", func(t *testing.T) {
		e := findOne(t, entries, "fd")
		if e.Type != domain.TypeAlias || e.Value != "fzf_cd" || e.Line != 11 {
			t.Fatalf("got %+v", e)
		}
	})

	t.Run("no false positives", func(t *testing.T) {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name)
		}
		want := []string{"fzf_cd", "fd", "keyword_brace", "next_line_brace", "posix_func", "subshell_func", "open"}
		if !reflect.DeepEqual(names, want) {
			t.Fatalf("names = %v, want %v", names, want)
		}
	})
}

func TestExpandBraces(t *testing.T) {
	tests := map[string][]string{
		"plain":        {"plain"},
		"{a,b}":        {"a", "b"},
		"x{a,b}y":      {"xay", "xby"},
		"{a,b}{1,2}":   {"a1", "a2", "b1", "b2"},
		"{a,{b,c}}":    {"a", "b", "c"},
		"{single}":     {"{single}"},
		"{unclosed,":   {"{unclosed,"},
		"{env,md,env}": {"env", "md"},
	}
	for in, want := range tests {
		if got := expandBraces(in); !reflect.DeepEqual(got, want) {
			t.Errorf("expandBraces(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseAliasLine(t *testing.T) {
	tests := []struct {
		line string
		want []domain.Entry
	}{
		{`alias x='a "b"'`, []domain.Entry{{Name: "x", Type: domain.TypeAlias, AliasKind: domain.AliasRegular, Value: `a "b"`}}},
		{`alias x="a \"b\""`, []domain.Entry{{Name: "x", Type: domain.TypeAlias, AliasKind: domain.AliasRegular, Value: `a "b"`}}},
		{`alias -- -x=y`, []domain.Entry{{Name: "-x", Type: domain.TypeAlias, AliasKind: domain.AliasRegular, Value: "y"}}},
		{`alias x=y; echo hi`, []domain.Entry{{Name: "x", Type: domain.TypeAlias, AliasKind: domain.AliasRegular, Value: "y"}}},
		{`alias ll`, nil},
		{`alias ${alias_name}="$cmd"`, nil},
		{"alias `echo x`=y", nil},
	}
	for _, tt := range tests {
		if got := parseAliasLine(tt.line); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseAliasLine(%q) = %+v, want %+v", tt.line, got, tt.want)
		}
	}
}
