#!/bin/zsh
# fuzzy cd into a directory
# ================================================================================

# Change directory with fzf
function fzf_cd() {
  local dir
  dir=$(fd --type d | fzf --preview 'ls {}') && cd "$dir"
}

alias fd="fzf_cd"

function keyword_brace {
  echo "brace on keyword line }"
}

function next_line_brace
{
  echo next
}

posix_func () { echo "one-liner"; }

subshell_func() (
  cd /tmp && ls
)

open() { # Open files with xdg-open
  xdg-open "$@" >/dev/null 2>&1 &
}
