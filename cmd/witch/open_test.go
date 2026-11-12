package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

type openCall struct {
	editor, path string
	line         int
}

// stubEditor replaces the editor launcher and records calls.
func stubEditor(t *testing.T) *[]openCall {
	t.Helper()
	var calls []openCall
	orig := openInEditor
	openInEditor = func(editorCmd, path string, line int) error {
		calls = append(calls, openCall{editorCmd, path, line})
		return nil
	}
	t.Cleanup(func() { openInEditor = orig })
	return &calls
}

func TestOpenFlag(t *testing.T) {
	env := setupCLI(t, "editor = \"vim\"\n")

	tests := []struct {
		name string
		args []string
		want openCall
	}{
		{"alias", []string{"-o", "ll"}, openCall{"vim", env.aliases, 1}},
		{"long flag", []string{"--open", "md"}, openCall{"vim", env.aliases, 3}},
		{"winning alias over executable", []string{"-o", "cat"}, openCall{"vim", env.aliases, 4}},
		{"function", []string{"-o", "fd"}, openCall{"vim", filepath.Join(env.functions, "fcd.sh"), 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := stubEditor(t)
			stdout, _, err := runWitch(t, tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stdout != "" {
				t.Errorf("expected no stdout when opening, got %q", stdout)
			}
			if len(*calls) != 1 || (*calls)[0] != tt.want {
				t.Fatalf("calls = %+v, want [%+v]", *calls, tt.want)
			}
		})
	}

	t.Run("executable is rejected", func(t *testing.T) {
		calls := stubEditor(t)
		_, _, err := runWitch(t, "-o", "ls")
		code, msg := exitCode(err)
		if code != 1 || !strings.Contains(msg, "ls is an executable") {
			t.Fatalf("got code %d msg %q", code, msg)
		}
		if len(*calls) != 0 {
			t.Fatalf("editor should not open, got %+v", *calls)
		}
	})

	t.Run("type filter can leave only an executable", func(t *testing.T) {
		stubEditor(t)
		if _, _, err := runWitch(t, "-o", "-t", "executable", "cat"); err == nil {
			t.Fatal("expected error opening an executable")
		}
	})

	t.Run("not found", func(t *testing.T) {
		calls := stubEditor(t)
		_, stderr, err := runWitch(t, "-o", "nope")
		if !errors.Is(err, errNotFound) || stderr != "witch: nope not found\n" {
			t.Fatalf("err %v stderr %q", err, stderr)
		}
		if len(*calls) != 0 {
			t.Fatalf("editor should not open, got %+v", *calls)
		}
	})

	t.Run("rejects multiple names", func(t *testing.T) {
		stubEditor(t)
		_, _, err := runWitch(t, "-o", "ll", "cat")
		if err == nil || !strings.Contains(err.Error(), "single name") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("rejects json", func(t *testing.T) {
		stubEditor(t)
		_, _, err := runWitch(t, "-o", "--json", "ll")
		if err == nil || !strings.Contains(err.Error(), "--json") {
			t.Fatalf("got %v", err)
		}
	})
}
