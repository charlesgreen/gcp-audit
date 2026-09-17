package report

import (
	"strings"
	"testing"
)

func TestExporters(t *testing.T) {
	r := New("example-project", "auditor@example.com", "/tmp/dump")
	r.Add("HIGH", "Public bucket", "allUsers has objectViewer", "gs://logs")
	r.Note("Locations scanned (1): europe-west1")
	r.Errors = 2

	md := r.Markdown()
	for _, n := range []string{"Project example-project", "HIGH: Public bucket", "gs://logs", "Permission / API Errors"} {
		if !strings.Contains(md, n) {
			t.Fatalf("markdown missing %q\n%s", n, md)
		}
	}
	csv, err := r.CSV()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csv, "gcp,example-project,HIGH,Public bucket") {
		t.Fatalf("csv: %s", csv)
	}
	js, err := r.JSONBytes()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), `"cloud": "gcp"`) {
		t.Fatalf("json: %s", js)
	}
}
