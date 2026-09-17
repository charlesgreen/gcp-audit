# GCP Configuration Audit

Read-only Google Cloud metadata extraction and security/engineering audit. Sibling of [aws-audit](https://github.com/charlesgreen/aws-audit): same dump hygiene and report formats, separate collector, no shared cloud SDK.

Implemented in Go with Google APIs. The `.sh` files are thin wrappers around the binaries.

## Prerequisites

- Go 1.24+
- [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials) with read-only access to the target project

```bash
gcloud auth application-default login
gcloud config set project YOUR_PROJECT
```

## Usage

```bash
./gcp-audit.sh --project YOUR_PROJECT
```

Common overrides:

```bash
./gcp-audit.sh --project YOUR_PROJECT --locations europe-west1,asia-northeast1
./gcp-audit.sh --project YOUR_PROJECT --out /tmp/gcp-audit --parallel 8 --format md,csv,json
```

Print every valid location code (no GCP credentials required):

```bash
./gcp-audit.sh --list-locations
```

Re-run reports against an existing dump:

```bash
./gcp-audit-summarize.sh ./gcp-audit-example-project-20260101-120000 --format md
./gcp-audit-summarize.sh ./gcp-audit-example-project-20260101-120000 --format csv
./gcp-audit-summarize.sh ./gcp-audit-example-project-20260101-120000 --format json
```

## Handling audit output

Each run writes a directory of live project metadata. Treat it as confidential. Do not commit it.

This repo gitignores `gcp-audit-*/` (the default output path). If you pass `--out`, keep that directory outside the working tree.

VPN shared secrets and service-account private key material are replaced with `[REDACTED]` before write. Secret payloads (`access secret versions`) are not requested.

## Locations

By default the collector scans **every compute region available to the project**, not only `us-*`. `--locations` restricts to a catalog-validated subset. `--list-locations` prints the official codes (regions plus `us` / `eu` / `asia` multi-regions and `global`).

## Output layout

```bash
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
make build    # bin/gcp-audit and bin/gcp-audit-summarize
```

Tests use a fake GCP client; they never call live APIs.

## License

MIT. See [LICENSE](LICENSE).
