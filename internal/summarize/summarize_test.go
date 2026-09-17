package summarize

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildFixture(t *testing.T) {
	rep, err := Build(filepath.Join("testdata", "sample"))
	if err != nil {
		t.Fatal(err)
	}
	md := rep.Markdown()
	needles := []string{
		"Project example-project",
		"Primitive project IAM role",
		"Firewall allows ingress from the public internet",
		"Cloud SQL instance has a public IPv4 address",
		"Locations scanned (1): europe-west1",
	}
	for _, n := range needles {
		if !strings.Contains(md, n) {
			t.Errorf("missing %q\n%s", n, md)
		}
	}
	csv, err := rep.CSV()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csv, "gcp,example-project,HIGH") {
		t.Fatalf("csv: %s", csv)
	}
}

func TestMissingDir(t *testing.T) {
	if _, err := Build("/no/such/dump"); err == nil {
		t.Fatal("expected error")
	}
}
