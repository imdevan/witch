package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/imdevan/witch/internal/adapters/editor"
	"github.com/imdevan/witch/internal/config"
	"github.com/imdevan/witch/internal/domain"
	"github.com/imdevan/witch/internal/utils"
)

type configInitOptions struct {
	force        bool
	openInEditor bool
}

func newConfigInitCmd() *cobra.Command {
	opts := &configInitOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a default config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigInit(cmd, opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "overwrite existing config")
	cmd.Flags().BoolVarP(&opts.openInEditor, "editor", "e", false, "open config in editor after creation")
	return cmd
}

func runConfigInit(cmd *cobra.Command, opts *configInitOptions) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	manager := config.NewManager(cwd)
	exists, err := manager.Exists()
	if err != nil {
		return err
	}
	if exists && !opts.force {
		return fmt.Errorf("config already exists at %s (use --force to overwrite)", utils.ConfigPathGlobal())
	}
	cfg := domain.DefaultConfig()
	path := utils.ConfigPathGlobal()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := renderConfigTemplate(cfg)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	if opts.openInEditor {
		editorAdapter := editor.New(cfg.Editor)
		if err := editorAdapter.Open(path); err != nil {
			return err
		}
	}
	cmd.Printf("Wrote config to %s\n", utils.ConfigPathGlobal())
	return nil
}

func renderConfigTemplate(cfg domain.Config) string {
	var builder strings.Builder
	builder.WriteString("# witch configuration\n\n")
	builder.WriteString("# General\n")
	builder.WriteString(fmt.Sprintf("# editor = %q\n", cfg.Editor))
	builder.WriteString("\n# CLI behavior\n")
	builder.WriteString("# Open the interactive list when run with no name (or no exact match) on a terminal.\n")
	builder.WriteString(fmt.Sprintf("# interactive_default = %t\n", cfg.InteractiveDefault))
	builder.WriteString("\n# What enter does in the interactive browser: \"execute\" runs the selection\n")
	builder.WriteString("# (in your current shell with `eval \"$(witch init zsh)\"`), \"print\" prints it like witch <name>.\n")
	builder.WriteString(fmt.Sprintf("# enter_action = %q\n", cfg.EnterAction))
	builder.WriteString("\n# Index executables on $PATH (like which). Set false to only search sources.\n")
	builder.WriteString(fmt.Sprintf("# index_path = %t\n", cfg.IndexPath))
	builder.WriteString("\n# UI\n")
	builder.WriteString("# Render the interactive browser fullscreen (alternate screen, whole terminal pane)\n")
	builder.WriteString("# instead of inline below the prompt.\n")
	builder.WriteString(fmt.Sprintf("# fullscreen = %t\n", cfg.Fullscreen))
	builder.WriteString("# list_spacing options: compact (title only), tight (title + description, no margin), space (default, with spacing)\n")
	builder.WriteString(fmt.Sprintf("# list_spacing = %q\n", cfg.ListSpacing))
	builder.WriteString("\n# Colors\n")
	builder.WriteString("# Colors support named, numeric, or hex values (ex: 7, 13, \"#ff8800\").\n")
	builder.WriteString(fmt.Sprintf("# headings = %q\n", cfg.Headings))
	builder.WriteString(fmt.Sprintf("# primary = %q\n", cfg.Primary))
	builder.WriteString(fmt.Sprintf("# secondary = %q\n", cfg.Secondary))
	builder.WriteString(fmt.Sprintf("# text = %q\n", cfg.Text))
	builder.WriteString(fmt.Sprintf("# text_highlight = %q\n", cfg.TextHighlight))
	builder.WriteString(fmt.Sprintf("# description_highlight = %q\n", cfg.DescriptionHighlight))
	builder.WriteString(fmt.Sprintf("# tags = %q\n", cfg.Tags))
	builder.WriteString(fmt.Sprintf("# flags = %q\n", cfg.Flags))
	builder.WriteString(fmt.Sprintf("# accent = %q\n", cfg.Accent))
	builder.WriteString(fmt.Sprintf("# muted = %q\n", cfg.Muted))
	builder.WriteString(fmt.Sprintf("# border = %q\n", cfg.Border))
	builder.WriteString("# Background for the entry name and search term in the preview\n")
	builder.WriteString(fmt.Sprintf("# highlight = %q\n", cfg.Highlight))
	builder.WriteString("\n# Sources\n")
	builder.WriteString("# Shell files or directories to parse for aliases and functions.\n")
	builder.WriteString("# Directories are searched recursively. There are no default sources:\n")
	builder.WriteString("# without any, witch only searches $PATH, like which.\n")
	builder.WriteString("# Paths support ~ and $VARS; relative paths are relative to this file.\n")
	builder.WriteString("#\n")
	builder.WriteString("# [[sources]]\n")
	builder.WriteString("# path = \"~/.zshrc\"\n")
	builder.WriteString("#\n")
	builder.WriteString("# [[sources]]\n")
	builder.WriteString("# path = \"~/.config/zsh\"\n")
	builder.WriteString("\n# Ignore\n")
	builder.WriteString("# Files, directories, or glob patterns to skip inside sources.\n")
	builder.WriteString("# A glob without a slash (like \"*.zwc\") matches file names anywhere.\n")
	builder.WriteString("#\n")
	builder.WriteString("# [[ignore]]\n")
	builder.WriteString("# path = \"*.zwc\"\n")
	builder.WriteString("#\n")
	builder.WriteString("# [[ignore]]\n")
	builder.WriteString("# path = \"~/.config/zsh/plugins/**\"\n")
	return builder.String()
}
