# GCP Configuration Audit

Read-only CLI that snapshots a Google Cloud project and writes findings as markdown, CSV, and JSON. Use it for monthly security reviews and as evidence toward ISO 27001 and SOC 2 technical controls.

Sibling of [aws-audit](https://github.com/charlesgreen/aws-audit): same dump hygiene and report idea, separate collector, no shared cloud SDK. Implemented in Go with Google APIs.

## Install

From a [GitHub Release](https://github.com/charlesgreen/gcp-audit/releases) (Linux amd64; replace the version):

```bash
curl -L https://github.com/charlesgreen/gcp-audit/releases/download/v0.1.0/gcp-audit_0.1.0_Linux_x86_64.tar.gz | tar xz
sudo mv gcp-audit gcp-audit-summarize /usr/local/bin/
```

From source:

```bash
go install github.com/charlesgreen/gcp-audit/cmd/gcp-audit@latest
go install github.com/charlesgreen/gcp-audit/cmd/gcp-audit-summarize@latest
```

A `v*` tag on `main` runs [GoReleaser](https://goreleaser.com) and publishes archives, checksums, and SBOMs.

## Prerequisites

- Go 1.26+ (from source; see `go.mod`)
- [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials) with read-only access to the target project

```bash
gcloud auth application-default login
gcloud config set project YOUR_PROJECT
```

`gcloud` is only for login and setting the default project. The collector uses Google APIs through ADC.

## Quick start

```bash
go run ./cmd/gcp-audit --list-locations
go run ./cmd/gcp-audit --project YOUR_PROJECT
```

After `make build`, use `./bin/gcp-audit`. After `go install` or a release archive, use `gcp-audit` on your `PATH`.

## Usage

```bash
gcp-audit --project YOUR_PROJECT
```

Common overrides:

```bash
gcp-audit --project YOUR_PROJECT --locations europe-west1,asia-northeast1
gcp-audit --project YOUR_PROJECT --out /tmp/gcp-audit --parallel 8 --format md,csv,json
```

`--project` can also come from `GOOGLE_CLOUD_PROJECT`, `CLOUDSDK_CORE_PROJECT`, or `GCP_PROJECT`.

Print every valid location code (no GCP credentials):

```bash
gcp-audit --list-locations
```

Re-run reports against an existing dump:

```bash
gcp-audit-summarize ./gcp-audit-example-project-20260101-120000 --format md
gcp-audit-summarize ./gcp-audit-example-project-20260101-120000 --format csv
gcp-audit-summarize ./gcp-audit-example-project-20260101-120000 --format json
```

Default `--format` on a collection run writes all three: `summary.md`, `summary.csv`, `summary.json`.

## Audit output

Each run writes a directory of live project metadata. Treat it as confidential. Do not commit it.

This repo gitignores `/gcp-audit-*/` at the repo root (the default output path). If you pass `--out`, keep that directory outside the working tree.

VPN shared secrets and service-account private-key material are replaced with `[REDACTED]` before write. Secret payloads (`access secret versions`) are not requested.

## Locations

By default the collector scans **every compute region available to the project**, not only `us-*`. `--locations` restricts to a catalog-validated subset. `--list-locations` prints the official codes (regions plus `us` / `eu` / `asia` multi-regions and `global`).

## Output layout

```text
gcp-audit-<project>-<UTC-timestamp>/
  meta.json
  errors.log
  summary.md
  summary.csv
  summary.json
  global/           # project IAM, networks, firewalls, buckets, DNS, logging, ...
  locations/<loc>/  # Cloud SQL, GKE, Cloud Run, Functions, KMS, Secret Manager, ...
```

## What the summary checks

- Primitive `roles/owner` and `roles/editor` on the project
- `allUsers` / `allAuthenticatedUsers` on the project IAM policy
- Firewall rules with `0.0.0.0/0` ingress
- Default VPC network
- Cloud SQL instances with public IPv4
- GKE legacy ABAC and non-private nodes
- User-managed service account keys
- Missing Cloud Logging sinks

## Build and test

```bash
make check    # gofmt, go vet, go test (no GCP credentials)
make build    # ./bin/gcp-audit and ./bin/gcp-audit-summarize
```

Tests use a fake GCP client; they never call live APIs.

Optional repo-root wrappers (`gcp-audit.sh`, `gcp-audit-summarize.sh`) exec `./bin/*` if present, otherwise `go run ./cmd/...`. Releases and `PATH` installs use the Go binaries, not the wrappers.

## License

MIT. See [LICENSE](LICENSE).
