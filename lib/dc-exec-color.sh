# Color profile for dc-exec interactive shells. Copied to /tmp/dc-cli.bashrc.
# shellcheck shell=bash
if [[ -f /etc/bash.bashrc ]]; then
  # shellcheck source=/dev/null
  . /etc/bash.bashrc
fi
if [[ -f "$HOME/.bashrc" ]]; then
  # shellcheck source=/dev/null
  . "$HOME/.bashrc"
fi

export TERM="${TERM:-xterm-256color}"
export COLORTERM="${COLORTERM:-truecolor}"
export CLICOLOR="${CLICOLOR:-1}"
export FORCE_COLOR="${FORCE_COLOR:-1}"
export LESS="${LESS:--R}"

if command -v ls >/dev/null 2>&1 && ls --color=auto / >/dev/null 2>&1; then
  alias ls='ls --color=auto'
fi
alias grep='grep --color=auto' 2>/dev/null || true

case "${PS1:-}" in
  *\\[*) ;;
  *) PS1='\[\e[36m\]\u@\h\[\e[0m\]:\[\e[34m\]\w\[\e[0m\]\$ ' ;;
esac

# Highlight Apache/app log lines (pipe: tail -f /var/log/... | hl)
hl() {
  sed -E \
    -e $'s/\\b(GET|HEAD|OPTIONS)\\b/\033[1;36m&\033[0m/g' \
    -e $'s/\\b(POST|PUT|PATCH)\\b/\033[1;33m&\033[0m/g' \
    -e $'s/\\b(DELETE)\\b/\033[1;31m&\033[0m/g' \
    -e $'s/(HTTP\\/[0-9.]+"[[:space:]]+)(2[0-9]{2})\\b/\\1\033[1;32m\\2\033[0m/g' \
    -e $'s/(HTTP\\/[0-9.]+"[[:space:]]+)(3[0-9]{2})\\b/\\1\033[1;36m\\2\033[0m/g' \
    -e $'s/(HTTP\\/[0-9.]+"[[:space:]]+)(4[0-9]{2})\\b/\\1\033[1;33m\\2\033[0m/g' \
    -e $'s/(HTTP\\/[0-9.]+"[[:space:]]+)(5[0-9]{2})\\b/\\1\033[1;31m\\2\033[0m/g' \
    -e $'s/\\b([0-9]{1,3}\\.){3}[0-9]{1,3}\\b/\033[90m&\033[0m/g' \
    -e $'s/\\b(ERROR|ERR|FATAL)\\b/\033[1;31m&\033[0m/g' \
    -e $'s/\\b(WARN(ING)?)\\b/\033[1;33m&\033[0m/g' \
    -e $'s/\\b(INFO|DEBUG)\\b/\033[32m&\033[0m/g'
}
