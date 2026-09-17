#!/usr/bin/env bash
# Optional local helper: exec ./bin/gcp-audit or go run ./cmd/gcp-audit.
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
if [[ -x "${DIR}/bin/gcp-audit" ]]; then
  exec "${DIR}/bin/gcp-audit" "$@"
fi
exec go run "${DIR}/cmd/gcp-audit" "$@"
