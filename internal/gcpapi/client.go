package gcpapi

import (
	"context"
	"encoding/json"
)

// Identity is the audited project plus the caller principal.
type Identity struct {
	ProjectID     string
	ProjectNumber string
	Principal     string
}

// Dump is one List/Get call written into the audit tree.
type Dump struct {
	Location  string
	Service   string
	Operation string
	Input     map[string]string
}

// Client is the injectable GCP surface. Production uses Live; tests use Fake.
type Client interface {
	Identity(ctx context.Context, project string) (Identity, error)
	ListRegions(ctx context.Context, project string) ([]string, error)
	Dump(ctx context.Context, req Dump) (json.RawMessage, error)
}
