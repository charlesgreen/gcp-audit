package gcpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Fake is an in-memory Client. No network.
type Fake struct {
	mu        sync.Mutex
	Ident     Identity
	Regions   []string
	Bodies    map[string]json.RawMessage
	Errs      map[string]error
	Calls     []Dump
	IdentErr  error
	RegionErr error
}

func key(d Dump) string {
	return d.Location + "|" + d.Service + "|" + d.Operation
}

func (f *Fake) Identity(context.Context, string) (Identity, error) {
	if f.IdentErr != nil {
		return Identity{}, f.IdentErr
	}
	if f.Ident.ProjectID == "" {
		return Identity{}, fmt.Errorf("not authenticated")
	}
	return f.Ident, nil
}

func (f *Fake) ListRegions(context.Context, string) ([]string, error) {
	if f.RegionErr != nil {
		return nil, f.RegionErr
	}
	if f.Regions == nil {
		return nil, fmt.Errorf("compute regions.list unavailable")
	}
	return f.Regions, nil
}

func (f *Fake) Dump(_ context.Context, req Dump) (json.RawMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, req)
	k := key(req)
	if err, ok := f.Errs[k]; ok {
		return nil, err
	}
	if b, ok := f.Bodies[k]; ok {
		return b, nil
	}
	return json.RawMessage(`{}`), nil
}
