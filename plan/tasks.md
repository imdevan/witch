## Feature 1: Finish template → witch conversion

- [x] 1.1 Package metadata
  - [x] 1.1.1 Update `internal/package/package.toml` description/short ("which, but better")
  - [x] 1.1.2 Module path → `github.com/imdevan/witch`
  - [x] 1.1.3 Sync `package.toml` `module` with `go.mod` and run `just sync`
- [x] 1.2 Remove template-only code
  - [x] 1.2.1 Strip directory-listing demo + stub actions from `cmd/witch/root.go`
  - [x] 1.2.2 Remove unused adapters (icon/tty/shell alias formatter) or keep only what witch uses
- [x] 1.3 Docs
  - [x] 1.3.1 Rewrite `README.md`, `INSTALL.md`, `example-config.toml` for witch
  - [x] 1.3.2 Update `config_init.go` defaults
- [x] 1.4 `just build` and `just test` pass after cleanup

## Feature 2: Sources 
- [x] 2.1 Domain model
  - [x] 2.1.1 `domain.Entry` { Name, Description, File, Line }
  - [x] 2.1.2 Alias sub-types: regular, global (`-g`), suffix (`-s`)
- [x] 2.2 Config sources
  - [x] 2.2.1 `[[sources]]` array in config.toml with `path` (file or dir)
  - [x] 2.2.2 No default location. Default behavior is just which fallback.
  - [x] 2.2.3 `[[ignore]]` array in config.toml can be direct file or blob pattern
  - [x] 2.2.4 Parse `alias [-g|-s] name="value" # comment`; comment → description
  - [x] 2.2.5 Expand brace suffix aliases (`alias -s {js,ts,...}=...`) into one entry each
  - [x] 2.2.6 Record file + line number
- [x] 2.3 Function parser 
  - [x] 2.3.1 Definitions: `function name()`, `function name {`, `name() {` → name + line
  - [x] 2.3.2 Aliases declared inside function files (e.g. `alias fd="fzf_cd"`) → alias entries
- [x] 2.4 PATH executables 
  - [x] 2.4.1 Walk `$PATH` in order, collect executable files
  - [x] 2.4.2 Keep every match; mark later duplicates as shadowed (like `which -a`)
  - [x] 2.4.3 Resolve symlinks for the preview
  - [x] 2.4.4 Config toggle to disable PATH indexing
- [x] 2.5 Collector merges all sources; precedence alias > function > PATH for exact-name lookups
- [x] 2.6 Unit tests per parser with fixture files copied from real dotfiles formats

## Feature 3: CLI behavior
- [x] 3.1 `witch <name>` non-interactive when exact match and stdout is not a TTY (drop-in for `which`)
- [x] 3.2 `witch` with no args / ambiguous match → interactive UI
- [x] 3.3 Flags: `-a/--all`, `-t/--type`, `--json`, `-i/--interactive`
- [x] 3.4 Exit code 1 when nothing found (match `which`)
- [x] 3.5 Update `README.md`, `example-config.toml`, `config_init.go` for new flags and `[[sources]]`
- [x] 3.6 cmd tests for non-interactive output
- [x] 3.7 add -o/--open flag to open the alias file at the given location of the alias
  - [x] 3.7.1 `-o/--open` opens the winning alias/function's file at its line via the editor adapter (`editor` config, then `$VISUAL`/`$EDITOR`)
  - [x] 3.7.2 Respects `-t`; works from the picker (`witch -o`, or a fuzzy miss on a TTY) by opening the selection
  - [x] 3.7.3 Errors: executables (no definition line), more than one name, `--json`; not found exits 1
  - [x] 3.7.4 cmd tests with a stubbed editor; README/CONFIG docs updated

## Feature 4: Split UI
- [x] 4.1 Split layout component in `internal/ui/split.go`
  - [x] 4.1.1 List on left, preview viewport on right (lipgloss `JoinHorizontal`)
  - [x] 4.1.2 Split only at `BreakpointLG`+; below that, list only + enter opens preview
  - [x] 4.1.3 Inline (not alt-screen) per AGENTS.md
  - [x] 4.1.4 Rounded bordered panes; left pane 40% of width clamped to 30–60 cols; height capped at 40 lines / terminal height
  - [x] 4.1.5 Browser model (`internal/ui/browser.go`) replaces the Feature 3 single-list picker; wired into `witch`, `-i`, `-o`, and fuzzy misses
- [x] 4.2 List pane
  - [x] 4.2.1 Reuse `ui.NewListModel` / delegate; title = name, description = type + description
  - [x] 4.2.2 Type badges colored from theme (alias `tags`, function `flags`, executable `primary`, shadowed `muted`) via a delegate wrapping `list.DefaultDelegate`
  - [x] 4.2.3 Fuzzy filtering (bubbles list built-in), pre-filled with CLI arg
  - [x] 4.2.4 Type filter keys: `tab`/`shift+tab` cycle all/alias/function/executable; list title shows the active type
    - [ ] 4.2.4.1 bookmark type (no bookmark source exists yet; see open questions)
- [x] 4.3 Preview pane (bubbles `viewport`)
  - [x] 4.3.1 preview pane should show file preview of given alias / function
  - [x] 4.3.2 the user should see the contents of the file, with the alias / function highlighted so that it is clear what alias the user has searched and where it is in the file
  - [x] 4.3.3 Header with name, type badge, `~/path:line`, symlink target, shadowed marker
  - [x] 4.3.4 Line-number gutter; definition lines marked `▌` and highlighted; scrolled into view with context above
  - [x] 4.3.5 Executables: scripts previewed as text; binaries show path/target/mode/size; missing files show "file not found"
  - [x] 4.3.6 File contents cached per path; `J`/`K`, `ctrl+d`/`ctrl+u` scroll; horizontal scroll via viewport
- [x] 4.4 Actions
  - [x] 4.4.1 `enter` print selection and exit (shell-friendly)
  - [x] 4.4.2 `y` copy name/path/definition (clipboard adapter): `y` definition (executable path), `Y` path:line, `c` name; status line confirms
  - [x] 4.4.3 `o` open defining file at line in editor (editor adapter); executables show a status instead
- [x] 4.5 Responsive help keys, matching template patterns
  - [x] 4.5.1 bubbles `help` short view trimmed by breakpoint (XS 2, SM 3, MD 4, LG+ all), `?` always kept; `?` toggles full help and re-lays out panes
- [x] 4.6 Tests: split layout sizing, preview highlight/scroll/cache/binary, browser keys (split vs narrow, enter, quit, prefilled filter, type cycle, copy, open, help)

## Feature 5: Release
- [ ] 5.1 Docs site via `just docs-generate`
- [ ] 5.2 `just release` / homebrew
