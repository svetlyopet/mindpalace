#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

if [ ! -d .git ]; then
  echo "error: not a git repository (expected .git in ${REPO_ROOT})" >&2
  exit 1
fi

git config core.hooksPath .githooks

for hook in .githooks/*; do
  [ -f "$hook" ] || continue
  chmod +x "$hook"
done

if ! command -v gitleaks >/dev/null 2>&1; then
  echo "warning: gitleaks not found on PATH; pre-commit secrets scan will fail until installed" >&2
  echo "         https://github.com/gitleaks/gitleaks" >&2
fi

if ! command -v semgrep >/dev/null 2>&1; then
  echo "warning: semgrep not found on PATH; pre-commit semgrep scan will fail until installed" >&2
  echo "         pip install semgrep  (or see https://semgrep.dev/docs/getting-started/)" >&2
fi

echo "Git hooks enabled (core.hooksPath=.githooks)"
for hook in .githooks/*; do
  [ -f "$hook" ] || continue
  echo "  - $(basename "$hook")"
done
echo "To disable: git config --unset core.hooksPath"