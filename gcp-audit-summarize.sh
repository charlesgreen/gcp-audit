#!/usr/bin/env bash
# Thin wrapper around the Go summarizer (cmd/gcp-audit-summarize).
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
if [[ -x "${DIR}/bin/gcp-audit-summarize" ]]; then
  exec "${DIR}/bin/gcp-audit-summarize" "$@"
fi
exec go run "${DIR}/cmd/gcp-audit-summarize" "$@"
