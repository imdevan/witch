package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/imdevan/witch/internal/config"
	"github.com/imdevan/witch/internal/domain"
)

func TestConfigTemplateUncommentedParses(t *testing.T) {
	rendered := renderConfigTemplate(domain.DefaultConfig())
	for _, want := range []string{"index_path", "[[sources]]", "[[ignore]]"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("template missing %q", want)
		}
	}
	// Uncomment "# key = value" and "# [[table]]" lines and make sure the
	// result is a valid config.
	uncomment := regexp.MustCompile(`(?m)^# ((?:[a-z_]+ = .*)|(?:\[\[[a-z]+\]\]))$`)
	content := uncomment.ReplaceAllString(rendered, "$1")
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.NewManager(t.TempDir()).LoadWithOverride(path)
	if err != nil {
		t.Fatalf("uncommented template does not parse: %v\n%s", err, content)
	}
	if len(cfg.Sources) != 2 || len(cfg.Ignore) != 2 {
		t.Fatalf("expected example sources and ignores, got %+v %+v", cfg.Sources, cfg.Ignore)
	}
}

func TestExampleConfigParses(t *testing.T) {
	cfg, err := config.NewManager(t.TempDir()).LoadWithOverride(filepath.Join("..", "..", "example-config.toml"))
	if err != nil {
		t.Fatalf("example-config.toml does not parse: %v", err)
	}
	if len(cfg.Sources) == 0 || len(cfg.Ignore) == 0 || !cfg.IndexPath {
		t.Fatalf("unexpected example config: %+v", cfg)
	}
}
