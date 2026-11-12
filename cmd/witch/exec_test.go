package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/imdevan/witch/internal/domain"
	"github.com/imdevan/witch/internal/ui"
)

func stubShell(t *testing.T, err error) *[]string {
	t.Helper()
	var calls []string
	orig := runInShell
	runInShell = func(line string) error {
		calls = append(calls, line)
		return err
	}
	t.Cleanup(func() { runInShell = orig })
	return &calls
}

func testCmd() (*cobra.Command, *bytes.Buffer) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	return cmd, &out
}

var (
	aliasEntry  = domain.Entry{Name: "asdf", Type: domain.TypeAlias, AliasKind: domain.AliasRegular, Value: "cowsay 'hi'", Definition: `alias asdf="cowsay 'hi'"`, File: "/dots/aliases.zsh", Line: 447}
	funcEntry   = domain.Entry{Name: "greet", Type: domain.TypeFunction, Definition: "greet() { echo hi; }", File: "/dots/f.sh", Line: 3}
	execEntry   = domain.Entry{Name: "my tool", Type: domain.TypeExecutable, File: "/opt/my tool/bin/it's"}
	globalEntry = domain.Entry{Name: "G", Type: domain.TypeAlias, AliasKind: domain.AliasGlobal, Value: "| rg", Definition: `alias -g G="| rg"`, File: "/dots/aliases.zsh", Line: 39}
	suffixEntry = domain.Entry{Name: "md", Type: domain.TypeAlias, AliasKind: domain.AliasSuffix, Value: "nvim", Definition: "alias -s md=nvim", File: "/dots/aliases.zsh", Line: 40}
)

func TestCommandFor(t *testing.T) {
	tests := map[string]struct {
		entry domain.Entry
		want  string
	}{
		"alias runs by name":           {aliasEntry, "asdf"},
		"function runs by name":        {funcEntry, "greet"},
		"executable runs by path":      {execEntry, `'/opt/my tool/bin/it'\''s'`},
		"plain path is not quoted":     {domain.Entry{Type: domain.TypeExecutable, File: "/usr/bin/ls"}, "/usr/bin/ls"},
		"global alias is not runnable": {globalEntry, ""},
		"suffix alias is not runnable": {suffixEntry, ""},
	}
	for name, tt := range tests {
		if got := commandFor(tt.entry); got != tt.want {
			t.Errorf("%s: commandFor = %q, want %q", name, got, tt.want)
		}
	}
}

func TestExecuteWithShellIntegration(t *testing.T) {
	calls := stubShell(t, nil)
	file := filepath.Join(t.TempDir(), "cmd")
	t.Setenv(cmdFileEnv, file)

	cmd, out := testCmd()
	if err := executeEntry(cmd, aliasEntry); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != "asdf\n" {
		t.Fatalf("command file = %q, %v", data, err)
	}
	if out.Len() != 0 || len(*calls) != 0 {
		t.Errorf("with integration, nothing should print or run directly: out=%q calls=%v", out.String(), *calls)
	}
}

func TestExecuteWithoutIntegrationRunsInShell(t *testing.T) {
	t.Setenv(cmdFileEnv, "")
	calls := stubShell(t, nil)
	cmd, out := testCmd()
	if err := executeEntry(cmd, funcEntry); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 || (*calls)[0] != "greet" || out.Len() != 0 {
		t.Fatalf("calls=%v out=%q", *calls, out.String())
	}
}

func TestExecutePropagatesExitStatus(t *testing.T) {
	t.Setenv(cmdFileEnv, "")
	exitErr := exec.Command("sh", "-c", "exit 7").Run()
	stubShell(t, exitErr)
	cmd, _ := testCmd()
	err := executeEntry(cmd, aliasEntry)
	if code, msg := exitCode(err); code != 7 || msg != "" {
		t.Fatalf("exit = %d %q, want 7 and no message", code, msg)
	}

	stubShell(t, errors.New("boom"))
	if err := executeEntry(cmd, aliasEntry); err == nil || err.Error() != "boom" {
		t.Fatalf("other errors should pass through, got %v", err)
	}
}

func TestExecuteNonRunnablePrints(t *testing.T) {
	calls := stubShell(t, nil)
	file := filepath.Join(t.TempDir(), "cmd")
	t.Setenv(cmdFileEnv, file)
	cmd, out := testCmd()
	if err := executeEntry(cmd, globalEntry); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `alias -g G="| rg"`) || len(*calls) != 0 {
		t.Fatalf("global alias should print its definition, out=%q", out.String())
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("nothing should be written for the shell to run")
	}
}

func TestHandleSelection(t *testing.T) {
	t.Setenv(cmdFileEnv, "")
	selected := ui.BrowserResult{Entry: &aliasEntry, Action: ui.ActionSelect}

	t.Run("enter executes by default", func(t *testing.T) {
		calls := stubShell(t, nil)
		cmd, out := testCmd()
		if err := handleSelection(cmd, domain.DefaultConfig(), &rootOptions{}, selected, false); err != nil {
			t.Fatal(err)
		}
		if len(*calls) != 1 || (*calls)[0] != "asdf" || out.Len() != 0 {
			t.Fatalf("calls=%v out=%q", *calls, out.String())
		}
	})

	t.Run("enter_action print", func(t *testing.T) {
		calls := stubShell(t, nil)
		cfg := domain.DefaultConfig()
		cfg.EnterAction = domain.EnterPrint
		cmd, out := testCmd()
		if err := handleSelection(cmd, cfg, &rootOptions{}, selected, false); err != nil {
			t.Fatal(err)
		}
		if out.String() != formatEntry(aliasEntry)+"\n" || len(*calls) != 0 {
			t.Fatalf("out=%q calls=%v", out.String(), *calls)
		}
	})

	t.Run("json prints instead of executing", func(t *testing.T) {
		calls := stubShell(t, nil)
		cmd, out := testCmd()
		if err := handleSelection(cmd, domain.DefaultConfig(), &rootOptions{json: true}, selected, false); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), `"name": "asdf"`) || len(*calls) != 0 {
			t.Fatalf("out=%q calls=%v", out.String(), *calls)
		}
	})

	t.Run("open flag opens instead of executing", func(t *testing.T) {
		calls := stubShell(t, nil)
		opened := stubEditor(t)
		cmd, _ := testCmd()
		if err := handleSelection(cmd, domain.DefaultConfig(), &rootOptions{open: true}, selected, false); err != nil {
			t.Fatal(err)
		}
		if len(*opened) != 1 || len(*calls) != 0 {
			t.Fatalf("opened=%v calls=%v", *opened, *calls)
		}
	})

	t.Run("quit without selection", func(t *testing.T) {
		cmd, _ := testCmd()
		if err := handleSelection(cmd, domain.DefaultConfig(), &rootOptions{}, ui.BrowserResult{}, false); err != nil {
			t.Fatalf("browsing then quitting should succeed, got %v", err)
		}
		if err := handleSelection(cmd, domain.DefaultConfig(), &rootOptions{}, ui.BrowserResult{}, true); !errors.Is(err, errNotFound) {
			t.Fatalf("quitting a search should be not found, got %v", err)
		}
	})
}

func TestInvalidEnterAction(t *testing.T) {
	setupCLI(t, "enter_action = \"launch\"\n")
	_, _, err := runWitch(t, "ls")
	if err == nil || !strings.Contains(err.Error(), `invalid enter_action "launch"`) {
		t.Fatalf("got %v", err)
	}
}

func TestInitCommand(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish"} {
		stdout, _, err := runWitch(t, "init", shell)
		if err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		for _, want := range []string{cmdFileEnv, "command witch", "eval"} {
			if !strings.Contains(stdout, want) {
				t.Errorf("%s init missing %q:\n%s", shell, want, stdout)
			}
		}
	}
	zsh, _, _ := runWitch(t, "init", "zsh")
	bash, _, _ := runWitch(t, "init", "bash")
	if !strings.Contains(zsh, "print -s") || !strings.Contains(bash, "history -s") {
		t.Error("wrappers should add the command to shell history")
	}
	if _, _, err := runWitch(t, "init", "pwsh"); err == nil {
		t.Error("expected an error for an unsupported shell")
	}
}

// TestInitScriptsRunInShell sources the generated wrapper in real shells
// with a fake witch that "selects" its argument, and checks the command runs
// in the current shell (cd persists) and plain output passes through.
func TestInitScriptsRunInShell(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "witch")
	script := "#!/bin/sh\nif [ \"$1\" = plain ]; then echo plain-output; exit 3; fi\nprintf '%s\\n' \"$1\" > \"$WITCH_CMD_FILE\"\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	for _, shell := range []string{"zsh", "bash"} {
		path, err := exec.LookPath(shell)
		if err != nil {
			t.Logf("%s not installed; skipping", shell)
			continue
		}
		init, err := initScript(shell)
		if err != nil {
			t.Fatal(err)
		}
		initFile := filepath.Join(dir, "init."+shell)
		if err := os.WriteFile(initFile, []byte(init), 0o644); err != nil {
			t.Fatal(err)
		}
		body := "source " + initFile + "\nwitch 'cd " + target + "'\npwd\nwitch plain\necho status=$?\n"
		c := exec.Command(path, "-c", body)
		c.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", shell, err, out)
		}
		resolved, _ := filepath.EvalSymlinks(target)
		got := string(out)
		if !strings.Contains(got, resolved+"\n") && !strings.Contains(got, target+"\n") {
			t.Errorf("%s: cd should persist in the calling shell:\n%s", shell, got)
		}
		if !strings.Contains(got, "plain-output\nstatus=3") {
			t.Errorf("%s: plain runs should pass output and status through:\n%s", shell, got)
		}
	}
}
