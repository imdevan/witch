package domain

import (
	"os"
	"path/filepath"
)

// Config describes the resolved configuration.
type Config struct {
	Editor               string `toml:"editor"`
	Primary              string `toml:"primary"`
	Secondary            string `toml:"secondary"`
	Headings             string `toml:"headings"`
	Text                 string `toml:"text"`
	TextHighlight        string `toml:"text_highlight"`
	DescriptionHighlight string `toml:"description_highlight"`
	Tags                 string `toml:"tags"`
	Flags                string `toml:"flags"`
	Muted                string `toml:"muted"`
	Accent               string `toml:"accent"`
	Border               string `toml:"border"`
	// Highlight colors matches of the name and search term in the preview.
	Highlight          string `toml:"highlight"`
	InteractiveDefault bool   `toml:"interactive_default"`
	ListSpacing        string `toml:"list_spacing"`
	// EnterAction is what enter does in the interactive browser:
	// "execute" (run the selection) or "print" (print it, like witch <name>).
	EnterAction string `toml:"enter_action"`
	// IndexPath controls whether executables on $PATH are indexed.
	IndexPath bool `toml:"index_path"`
	// Sources are files or directories parsed for aliases and functions.
	Sources []Source `toml:"sources"`
	// Ignore lists files, directories, or glob patterns to skip in sources.
	Ignore []Ignore `toml:"ignore"`
}

// Enter actions for the interactive browser.
const (
	EnterExecute = "execute"
	EnterPrint   = "print"
)

// Source is a file or directory to parse for aliases and functions.
type Source struct {
	Path string `toml:"path"`
}

// Ignore is a file, directory, or glob pattern excluded from sources.
type Ignore struct {
	Path string `toml:"path"`
}

// DefaultConfig returns the default configuration values.
func DefaultConfig() Config {
	return Config{
		Editor:               "nvim",
		Headings:             "15",
		Primary:              "02",
		Secondary:            "06",
		Text:                 "07",
		TextHighlight:        "06",
		DescriptionHighlight: "05",
		Tags:                 "13",
		Flags:                "12",
		Muted:                "08",
		Accent:               "13",
		Border:               "08",
		Highlight:            "03",
		InteractiveDefault:   true,
		ListSpacing:          "space",
		EnterAction:          EnterExecute,
		IndexPath:            true,
	}
}

func xdgHome(envKey, fallbackSuffix string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, fallbackSuffix)
}
