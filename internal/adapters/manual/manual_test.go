package manual

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// noMan is a name with no man page, so Lookup falls through to --help.
const noMan = "witch-test-no-such-page"

func writeScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestClean(t *testing.T) {
	in := "N\bNA\bAM\bME\bE\n  _\bl_\bs - list\x1b[1m bold\x1b[0m  \r\n\x07"
	want := "NAME\n  ls - list bold"
	if got := Clean(in); got != want {
		t.Errorf("Clean() = %q, want %q", got, want)
	}
}

func TestLookupFallsBackToHelp(t *testing.T) {
	path := writeScript(t, `case "$1" in --help) echo "usage: tool [--help]"; echo "cols=$COLUMNS"; exit 2;; esac`+"\n")
	page, err := Lookup(noMan, path, 72)
	if err != nil {
		t.Fatal(err)
	}
	if page.Source != noMan+" --help" {
		t.Errorf("source = %q", page.Source)
	}
	if !strings.Contains(page.Text, "usage: tool [--help]") || !strings.Contains(page.Text, "cols=72") {
		t.Errorf("text = %q", page.Text)
	}
}

func TestLookupSkipsProgramsWithoutHelp(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ran")
	path := writeScript(t, "touch "+marker+"\n")
	if _, err := Lookup(noMan, path, 80); !errors.Is(err, ErrNoHelp) {
		t.Fatalf("err = %v, want ErrNoHelp", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("a program that never mentions --help must not be run")
	}
}

func TestLookupTimesOut(t *testing.T) {
	old := Timeout
	Timeout = 200 * time.Millisecond
	t.Cleanup(func() { Timeout = old })

	path := writeScript(t, "# --help\nsleep 5\n")
	start := time.Now()
	if _, err := Lookup(noMan, path, 80); err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("lookup took %s, expected it to be killed", elapsed)
	}
}

func TestLookupPrefersManPage(t *testing.T) {
	if exec.Command("man", "-w", "ls").Run() != nil {
		t.Skip("no man page for ls")
	}
	path := writeScript(t, "echo should not run --help\n")
	page, err := Lookup("ls", path, 80)
	if err != nil {
		t.Fatal(err)
	}
	if page.Source != "man ls" || !strings.Contains(page.Text, "NAME") || strings.Contains(page.Text, "\b") {
		t.Errorf("page = %q / %.200q", page.Source, page.Text)
	}
}

func TestMentionsHelpAcrossReads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bin")
	data := make([]byte, 64*1024-3)
	data = append(data, []byte("--help")...)
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatal(err)
	}
	if !mentionsHelp(path) {
		t.Error("expected a match spanning two reads")
	}
}
