package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/spf13/cobra"

	"github.com/imdevan/witch/internal/adapters/clipboard"
	"github.com/imdevan/witch/internal/adapters/editor"
	"github.com/imdevan/witch/internal/adapters/tty"
	"github.com/imdevan/witch/internal/config"
	"github.com/imdevan/witch/internal/domain"
	pkg "github.com/imdevan/witch/internal/package"
	"github.com/imdevan/witch/internal/sources"
	"github.com/imdevan/witch/internal/ui"
)

// Metadata loaded from package.toml at build time
var (
	version = pkg.Version()
	name    = pkg.Name()
	short   = pkg.Short()
)

type rootOptions struct {
	configPath  string
	showVersion bool
	all         bool
	json        bool
	interactive bool
	open        bool
	types       []string
}

// openInEditor opens path at line in the configured editor. It is a
// variable so tests can stub out launching a real editor.
var openInEditor = func(editorCmd, path string, line int) error {
	return editor.New(editorCmd).OpenAtLine(path, line)
}

var rootCmd = newRootCmd()

// Execute is the CLI entrypoint.
func Execute() error {
	return rootCmd.Execute()
}

// exitError carries a process exit code. An empty message prints nothing.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

// errNotFound mirrors `which`: exit 1 when a name cannot be resolved.
var errNotFound = &exitError{code: 1}

func newRootCmd() *cobra.Command {
	opts := &rootOptions{}
	cmd := &cobra.Command{
		Use:   name + " [name...]",
		Short: short,
		Long: short + `.

Resolve names the way your shell does: aliases, then functions, then
executables on $PATH. Aliases and functions are read from the files and
directories listed under [[sources]] in your config; with no sources
configured, witch behaves like which.

With a name and an exact match, witch prints the result and exits.
With no name, or no exact match on a terminal, it opens an interactive list.`,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.showVersion {
				cmd.Printf("%s\n", resolvedVersion())
				return nil
			}
			return runRoot(cmd, opts, args)
		},
	}

	// @docs-flag-group
	//
	// 	name: Lookup
	// 	description:
	//
	// 		Control which entries are printed and how.
	//	flags: all, type, json, interactive, open
	cmd.Flags().BoolVarP(&opts.all, "all", "a", false, "print all matches, including shadowed ones")
	cmd.Flags().StringSliceVarP(&opts.types, "type", "t", nil, "only match these types: alias, function, executable (repeatable or comma-separated)")
	cmd.Flags().BoolVar(&opts.json, "json", false, "print matches as JSON")
	cmd.Flags().BoolVarP(&opts.interactive, "interactive", "i", false, "always open the interactive list")
	cmd.Flags().BoolVarP(&opts.open, "open", "o", false, "open the file defining the alias or function in your editor, at its line")

	// @docs-flag-group
	//
	// 	name: Config
	// 	description:
	//
	// 		Define a config.toml file to use instead of the global one.
	//	flags: config
	cmd.Flags().StringVarP(&opts.configPath, "config", "c", "", "config file path")

	// @docs-flag-group
	//
	// 	name: Meta
	//	flags: version
	cmd.Flags().BoolVarP(&opts.showVersion, "version", "v", false, "print version information")

	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newInitCmd())

	return cmd
}

func resolvedVersion() string {
	ver := version
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ver
	}
	if ver == "dev" && strings.TrimSpace(info.Main.Version) != "" && info.Main.Version != "(devel)" {
		ver = info.Main.Version
	}
	return ver
}

func loadConfig(path string) (domain.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return domain.Config{}, fmt.Errorf("failed to get working directory: %w", err)
	}
	manager := config.NewManager(cwd)
	if path != "" {
		return manager.LoadWithOverride(path)
	}
	return manager.Load()
}

func runRoot(cmd *cobra.Command, opts *rootOptions, args []string) error {
	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		return err
	}
	if err := validateConfig(cfg); err != nil {
		return err
	}
	types, err := parseTypes(opts.types)
	if err != nil {
		return err
	}
	if opts.open {
		if opts.json {
			return errors.New("--open cannot be used with --json")
		}
		if len(args) > 1 {
			return errors.New("--open takes a single name")
		}
	}

	idx, warnings := sources.Collect(collectOptions(cfg))
	for _, w := range warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s: warning: %v\n", name, w)
	}

	stdout := cmd.OutOrStdout()
	isTTY := tty.IsTerminalWriter(stdout)
	all := filterTypes(idx.All(), types)

	if opts.interactive {
		return runPicker(cmd, cfg, opts, all, strings.Join(args, " "), len(args) > 0)
	}

	if len(args) == 0 {
		if opts.open || (!opts.json && isTTY && cfg.InteractiveDefault) {
			return runPicker(cmd, cfg, opts, all, "", false)
		}
		if !opts.all {
			all = winners(all)
		}
		if opts.json {
			return writeJSON(stdout, all)
		}
		writeList(stdout, all)
		return nil
	}

	var (
		found   []domain.Entry
		missing []string
	)
	for _, arg := range args {
		matches := filterTypes(idx.Lookup(arg), types)
		if len(matches) == 0 {
			missing = append(missing, arg)
			continue
		}
		if !opts.all {
			matches = matches[:1]
		}
		found = append(found, matches...)
	}

	// A single unresolved name on a terminal opens the picker pre-filtered,
	// as long as something fuzzy-matches it.
	if len(args) == 1 && len(missing) == 1 && isTTY && !opts.json && cfg.InteractiveDefault && hasFuzzyMatch(all, args[0]) {
		return runPicker(cmd, cfg, opts, all, args[0], true)
	}

	if opts.open {
		if len(missing) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s not found\n", name, missing[0])
			return errNotFound
		}
		return openEntry(cfg, found[0])
	}

	if opts.json {
		if err := writeJSON(stdout, found); err != nil {
			return err
		}
	} else {
		for _, e := range found {
			fmt.Fprintln(stdout, formatEntry(e))
		}
	}
	for _, m := range missing {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s not found\n", name, m)
	}
	if len(missing) > 0 {
		return errNotFound
	}
	return nil
}

func collectOptions(cfg domain.Config) sources.Options {
	opts := sources.Options{IndexPath: cfg.IndexPath, PathEnv: os.Getenv("PATH")}
	for _, s := range cfg.Sources {
		opts.Sources = append(opts.Sources, s.Path)
	}
	for _, i := range cfg.Ignore {
		opts.Ignore = append(opts.Ignore, i.Path)
	}
	return opts
}

func runPicker(cmd *cobra.Command, cfg domain.Config, opts *rootOptions, entries []domain.Entry, filter string, searched bool) error {
	if !opts.all {
		entries = winners(entries)
	}
	if len(entries) == 0 {
		return &exitError{code: 1, msg: fmt.Sprintf("%s: nothing to show (configure [[sources]] or enable index_path)", name)}
	}
	clip := clipboard.Adapter{}
	result, err := ui.RunBrowser(entries, ui.ThemeFromConfig(cfg), ui.BrowserOptions{
		Title:   name,
		Filter:  filter,
		Spacing: cfg.ListSpacing,
		Copy:    clip.WriteText,
	}, tty.GetProgramOptions()...)
	if err != nil {
		return err
	}
	return handleSelection(cmd, cfg, opts, result, searched)
}

// handleSelection acts on what the user chose in the browser. Enter runs
// the selection by default (enter_action = "execute"); --open, --json, and
// enter_action = "print" change that.
func handleSelection(cmd *cobra.Command, cfg domain.Config, opts *rootOptions, result ui.BrowserResult, searched bool) error {
	if result.Entry == nil || result.Action == ui.ActionNone {
		if searched {
			return errNotFound
		}
		return nil
	}
	e := *result.Entry
	switch {
	case result.Action == ui.ActionOpen || opts.open:
		return openEntry(cfg, e)
	case opts.json:
		return writeJSON(cmd.OutOrStdout(), []domain.Entry{e})
	case cfg.EnterAction == domain.EnterExecute:
		return executeEntry(cmd, e)
	default:
		fmt.Fprintln(cmd.OutOrStdout(), formatEntry(e))
		return nil
	}
}

func validateConfig(cfg domain.Config) error {
	switch cfg.EnterAction {
	case domain.EnterExecute, domain.EnterPrint:
		return nil
	default:
		return fmt.Errorf("invalid enter_action %q (expected %q or %q)", cfg.EnterAction, domain.EnterExecute, domain.EnterPrint)
	}
}

// openEntry opens the file that defines an alias or function at its line.
func openEntry(cfg domain.Config, e domain.Entry) error {
	if e.Type == domain.TypeExecutable || e.Line <= 0 {
		return &exitError{code: 1, msg: fmt.Sprintf("%s: %s is an executable (%s), not defined in a source file", name, e.Name, e.File)}
	}
	return openInEditor(cfg.Editor, e.File, e.Line)
}

func parseTypes(values []string) (map[domain.EntryType]bool, error) {
	if len(values) == 0 {
		return nil, nil
	}
	types := map[domain.EntryType]bool{}
	for _, v := range values {
		t, err := domain.ParseEntryType(v)
		if err != nil {
			return nil, err
		}
		types[t] = true
	}
	return types, nil
}

func filterTypes(entries []domain.Entry, types map[domain.EntryType]bool) []domain.Entry {
	if len(types) == 0 {
		return entries
	}
	out := make([]domain.Entry, 0, len(entries))
	for _, e := range entries {
		if types[e.Type] {
			out = append(out, e)
		}
	}
	return out
}

// winners keeps the first entry per name (suffix aliases are their own
// namespace). Entries must be grouped by name with the winner first, as
// returned by the index; this respects type filters, so `-t executable ls`
// still shows the executable even when an alias shadows it.
func winners(entries []domain.Entry) []domain.Entry {
	seen := map[string]bool{}
	out := make([]domain.Entry, 0, len(entries))
	for _, e := range entries {
		key := e.Name
		if e.AliasKind == domain.AliasSuffix {
			key = "suffix:" + key
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e)
	}
	return out
}

func hasFuzzyMatch(entries []domain.Entry, term string) bool {
	targets := make([]string, len(entries))
	for i, e := range entries {
		targets[i] = e.Name
	}
	return len(list.DefaultFilter(term, targets)) > 0
}

// exitCode extracts the process exit code for err.
func exitCode(err error) (int, string) {
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code, ee.msg
	}
	return 1, err.Error()
}
