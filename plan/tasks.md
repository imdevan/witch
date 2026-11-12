## Feature 0: Finish template → witch conversion

- [x] 0.1 Package metadata
  - [x] 0.1.1 Update `internal/package/package.toml` description/short ("which, but better")
  - [x] 0.1.2 Module path → `github.com/imdevan/witch`
  - [x] 0.1.3 Sync `package.toml` `module` with `go.mod` and run `just sync`
- [ ] 0.2 Remove template-only code
  - [ ] 0.2.1 Strip directory-listing demo + stub actions from `cmd/witch/root.go`
  - [ ] 0.2.2 Remove unused adapters (icon/tty/shell alias formatter) or keep only what witch uses
- [ ] 0.3 Docs
  - [ ] 0.3.1 Rewrite `README.md`, `INSTALL.md`, `example-config.toml` for witch
  - [ ] 0.3.2 Update `config_init.go` defaults
- [ ] 0.4 `just build` and `just test` pass after cleanup

## Feature 1: Sources 
- [ ] 1.1 Domain model
  - [ ] 1.1.1 `domain.Entry` { Name, Description, File, Line }
  - [ ] 1.1.2 Alias sub-types: regular, global (`-g`), suffix (`-s`)
- [ ] 1.2 Config sources
  - [ ] 1.2.1 `[[sources]]` array in config.toml with `path` (file or dir)
  - [ ] 1.2.2 No default location. Default behavior is just which fallback.
  - [ ] 1.2.3 `[[ignore]]` array in config.toml can be direct file or blob pattern
  - [ ] 1.3.1 Parse `alias [-g|-s] name="value" # comment`; comment → description
  - [ ] 1.3.2 Expand brace suffix aliases (`alias -s {js,ts,...}=...`) into one entry each
  - [ ] 1.3.3 Record file + line number
- [ ] 1.4 Function parser 
  - [ ] 1.4.1 Definitions: `function name()`, `function name {`, `name() {` → name + line
  - [ ] 1.4.2 Aliases declared inside function files (e.g. `alias fd="fzf_cd"`) → alias entries
- [ ] 1.5 PATH executables 
  - [ ] 1.5.1 Walk `$PATH` in order, collect executable files
  - [ ] 1.5.2 Keep every match; mark later duplicates as shadowed (like `which -a`)
  - [ ] 1.5.3 Resolve symlinks for the preview
  - [ ] 1.5.4 Config toggle to disable PATH indexing
- [ ] 1.6 Collector merges all sources; precedence alias > function > PATH for exact-name lookups
- [ ] 1.7 Unit tests per parser with fixture files copied from real dotfiles formats

## Feature 2: Split UI
- [ ] 2.1 Split layout component in `internal/ui/split.go`
  - [ ] 2.1.1 List on left, preview viewport on right (lipgloss `JoinHorizontal`)
  - [ ] 2.1.2 Split only at `BreakpointLG`+; below that, list only + enter opens preview
  - [ ] 2.1.3 Inline (not alt-screen) per AGENTS.md
- [ ] 2.2 List pane
  - [ ] 2.2.1 Reuse `ui.NewListModel` / delegate; title = name, description = type + description
  - [ ] 2.2.2 Type badges colored from theme (`tags`, `flags`, `accent`)
  - [ ] 2.2.3 Fuzzy filtering (bubbles list built-in), pre-filled with CLI arg
  - [ ] 2.2.4 Type filter keys (e.g. `tab` to cycle all/alias/function/bookmark/bin)
- [ ] 2.3 Preview pane (bubbles `viewport`)
  - [ ] 2.3.1 preview pane should show file preview of given alias / function
  - [ ] 2.3.2 the user should see the contents of the file, with the alias / function highlighted so that it is clear what alias the user has searched and where it is in the file

- [ ] 2.4 Actions
  - [ ] 2.4.1 `enter` print selection and exit (shell-friendly)
  - [ ] 2.4.2 `y` copy name/path/definition (clipboard adapter)
  - [ ] 2.4.3 `o` open defining file at line in editor (editor adapter)
- [ ] 2.5 Responsive help keys, matching template patterns

## Feature 3: CLI behavior
- [ ] 3.1 `witch <name>` non-interactive when exact match and stdout is not a TTY (drop-in for `which`)
- [ ] 3.2 `witch` with no args / ambiguous match → interactive UI
- [ ] 3.3 Flags: `-a/--all`, `-t/--type`, `--json`, `-i/--interactive`
- [ ] 3.4 Exit code 1 when nothing found (match `which`)
- [ ] 3.5 Update `README.md`, `example-config.toml`, `config_init.go` for new flags and `[[sources]]`
- [ ] 3.6 cmd tests for non-interactive output

## Feature 4: Release
- [ ] 4.1 Docs site via `just docs-generate`
- [ ] 4.2 `just release` / homebrew
