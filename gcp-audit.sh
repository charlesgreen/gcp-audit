#!/usr/bin/env bash
# Thin wrapper around the Go collector (cmd/gcp-audit).
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
if [[ -x "${DIR}/bin/gcp-audit" ]]; then
  exec "${DIR}/bin/gcp-audit" "$@"
fi
exec go run "${DIR}/cmd/gcp-audit" "$@"
