## Feature 1: Finish template → witch conversion

- [x] 1.1 Package metadata
  - [x] 1.1.1 Update `internal/package/package.toml` description/short ("which, but better")
  - [x] 1.1.2 Module path → `github.com/imdevan/witch`
  - [x] 1.1.3 Sync `package.toml` `module` with `go.mod` and run `just sync`
- [ ] 1.2 Remove template-only code
  - [ ] 1.2.1 Strip directory-listing demo + stub actions from `cmd/witch/root.go`
  - [ ] 1.2.2 Remove unused adapters (icon/tty/shell alias formatter) or keep only what witch uses
- [ ] 1.3 Docs
  - [ ] 1.3.1 Rewrite `README.md`, `INSTALL.md`, `example-config.toml` for witch
  - [ ] 1.3.2 Update `config_init.go` defaults
- [ ] 1.4 `just build` and `just test` pass after cleanup

## Feature 2: Sources 
- [ ] 2.1 Domain model
  - [ ] 2.1.1 `domain.Entry` { Name, Description, File, Line }
  - [ ] 2.1.2 Alias sub-types: regular, global (`-g`), suffix (`-s`)
- [ ] 2.2 Config sources
  - [ ] 2.2.1 `[[sources]]` array in config.toml with `path` (file or dir)
  - [ ] 2.2.2 No default location. Default behavior is just which fallback.
  - [ ] 2.2.3 `[[ignore]]` array in config.toml can be direct file or blob pattern
  - [ ] 2.2.4 Parse `alias [-g|-s] name="value" # comment`; comment → description
  - [ ] 2.2.5 Expand brace suffix aliases (`alias -s {js,ts,...}=...`) into one entry each
  - [ ] 2.2.6 Record file + line number
- [ ] 2.3 Function parser 
  - [ ] 2.3.1 Definitions: `function name()`, `function name {`, `name() {` → name + line
  - [ ] 2.3.2 Aliases declared inside function files (e.g. `alias fd="fzf_cd"`) → alias entries
- [ ] 2.4 PATH executables 
  - [ ] 2.4.1 Walk `$PATH` in order, collect executable files
  - [ ] 2.4.2 Keep every match; mark later duplicates as shadowed (like `which -a`)
  - [ ] 2.4.3 Resolve symlinks for the preview
  - [ ] 2.4.4 Config toggle to disable PATH indexing
- [ ] 2.5 Collector merges all sources; precedence alias > function > PATH for exact-name lookups
- [ ] 2.6 Unit tests per parser with fixture files copied from real dotfiles formats

## Feature 3: CLI behavior
- [ ] 3.1 `witch <name>` non-interactive when exact match and stdout is not a TTY (drop-in for `which`)
- [ ] 3.2 `witch` with no args / ambiguous match → interactive UI
- [ ] 3.3 Flags: `-a/--all`, `-t/--type`, `--json`, `-i/--interactive`
- [ ] 3.4 Exit code 1 when nothing found (match `which`)
- [ ] 3.5 Update `README.md`, `example-config.toml`, `config_init.go` for new flags and `[[sources]]`
- [ ] 3.6 cmd tests for non-interactive output

## Feature 4: Split UI
- [ ] 4.1 Split layout component in `internal/ui/split.go`
  - [ ] 4.1.1 List on left, preview viewport on right (lipgloss `JoinHorizontal`)
  - [ ] 4.1.2 Split only at `BreakpointLG`+; below that, list only + enter opens preview
  - [ ] 4.1.3 Inline (not alt-screen) per AGENTS.md
- [ ] 4.2 List pane
  - [ ] 4.2.1 Reuse `ui.NewListModel` / delegate; title = name, description = type + description
  - [ ] 4.2.2 Type badges colored from theme (`tags`, `flags`, `accent`)
  - [ ] 4.2.3 Fuzzy filtering (bubbles list built-in), pre-filled with CLI arg
  - [ ] 4.2.4 Type filter keys (e.g. `tab` to cycle all/alias/function/bookmark/bin)
- [ ] 4.3 Preview pane (bubbles `viewport`)
  - [ ] 4.3.1 preview pane should show file preview of given alias / function
  - [ ] 4.3.2 the user should see the contents of the file, with the alias / function highlighted so that it is clear what alias the user has searched and where it is in the file

- [ ] 4.4 Actions
  - [ ] 4.4.1 `enter` print selection and exit (shell-friendly)
  - [ ] 4.4.2 `y` copy name/path/definition (clipboard adapter)
  - [ ] 4.4.3 `o` open defining file at line in editor (editor adapter)
- [ ] 4.5 Responsive help keys, matching template patterns

## Feature 5: Release
- [ ] 5.1 Docs site via `just docs-generate`
- [ ] 5.2 `just release` / homebrew
