#!/usr/bin/env bash
# Optional local helper: exec ./bin/gcp-audit-summarize or go run ./cmd/gcp-audit-summarize.
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
if [[ -x "${DIR}/bin/gcp-audit-summarize" ]]; then
  exec "${DIR}/bin/gcp-audit-summarize" "$@"
fi
exec go run "${DIR}/cmd/gcp-audit-summarize" "$@"
