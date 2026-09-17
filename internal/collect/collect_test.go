package collect

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charlesgreen/gcp-audit/internal/gcpapi"
)

func testOpts(t *testing.T, fake *gcpapi.Fake) Options {
	t.Helper()
	if fake.Ident.ProjectID == "" {
		fake.Ident = gcpapi.Identity{ProjectID: "example-project", ProjectNumber: "1", Principal: "auditor@example.com"}
	}
	if fake.Regions == nil {
		fake.Regions = []string{"europe-west1", "us-central1"}
	}
	return Options{
		Project:  "example-project",
		OutDir:   t.TempDir(),
		Parallel: 2,
		Client:   fake,
		Formats:  []string{"md", "csv", "json"},
	}
}

func TestUnknownLocationRejectedWithoutDump(t *testing.T) {
	fake := &gcpapi.Fake{}
	opts := testOpts(t, fake)
	opts.RequestedCSV = "us-east-1"
	err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "us-east-1") {
		t.Fatalf("got %v", err)
	}
	if len(fake.Calls) != 0 {
		t.Fatalf("must not call GCP before rejecting unknown codes")
	}
}

func TestWritesMetaAndReports(t *testing.T) {
	fake := &gcpapi.Fake{}
	opts := testOpts(t, fake)
	opts.RequestedCSV = "europe-west1"
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(opts.OutDir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("mode %o", st.Mode().Perm())
	}
	raw, _ := os.ReadFile(filepath.Join(opts.OutDir, "meta.json"))
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	if meta["project_id"] != "example-project" || meta["cloud"] != "gcp" {
		t.Fatalf("%s", raw)
	}
	for _, name := range []string{"summary.md", "summary.csv", "summary.json"} {
		if _, err := os.Stat(filepath.Join(opts.OutDir, name)); err != nil {
			t.Fatal(name, err)
		}
	}
}

func TestRedactsVPNSecret(t *testing.T) {
	fake := &gcpapi.Fake{
		Bodies: map[string]json.RawMessage{
			"|compute|AggregatedListVpnTunnels": json.RawMessage(`{"items":{"europe-west1":{"vpnTunnels":[{"sharedSecret":"supersecret"}]}}}`),
		},
	}
	opts := testOpts(t, fake)
	opts.RequestedCSV = "europe-west1"
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(opts.OutDir, "global", "compute", "vpn-tunnels-aggregated.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "supersecret") {
		t.Fatalf("secret leaked: %s", raw)
	}
}

func TestDumpFailureWritesEmptyObject(t *testing.T) {
	fake := &gcpapi.Fake{
		Errs: map[string]error{
			"|cloudresourcemanager|GetIamPolicy": fmtErr("403"),
		},
	}
	opts := testOpts(t, fake)
	opts.RequestedCSV = "europe-west1"
	if err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(opts.OutDir, "global", "iam-policy.json"))
	if strings.TrimSpace(string(raw)) != "{}" {
		t.Fatalf("got %s", raw)
	}
	elog, _ := os.ReadFile(filepath.Join(opts.OutDir, "errors.log"))
	if !strings.Contains(string(elog), "403") {
		t.Fatalf("errors.log: %s", elog)
	}
}

type fmtErr string

func (e fmtErr) Error() string { return string(e) }
