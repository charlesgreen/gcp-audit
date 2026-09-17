package locations

import (
	"strings"
	"testing"
)

func TestLookupKnown(t *testing.T) {
	l, ok := Lookup("europe-west1")
	if !ok {
		t.Fatal("europe-west1 should be in the catalog")
	}
	if l.Name != "Belgium" || l.Class != ClassRegion {
		t.Fatalf("%+v", l)
	}
}

func TestLookupUnknown(t *testing.T) {
	if _, ok := Lookup("us-east-1"); ok {
		t.Fatal("AWS Region codes are not GCP locations")
	}
}

func TestParseRequested(t *testing.T) {
	got, err := ParseRequested(" EUROPE-WEST1 , asia-northeast1 ")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "europe-west1" || got[1] != "asia-northeast1" {
		t.Fatalf("%v", got)
	}
}

func TestParseRequestedRejectsUnknown(t *testing.T) {
	_, err := ParseRequested("us-central1,not-a-place")
	if err == nil || !strings.Contains(err.Error(), "not-a-place") {
		t.Fatalf("got %v", err)
	}
}

func TestFilterUnavailable(t *testing.T) {
	_, err := FilterRequested([]string{"africa-south1"}, []string{"us-central1", "europe-west1"})
	if err == nil {
		t.Fatal("expected unavailable error")
	}
}

func TestCatalogIsWorldwide(t *testing.T) {
	all := Codes(ClassRegion)
	need := []string{"asia-northeast1", "europe-west1", "southamerica-east1", "us-central1", "me-west1", "africa-south1"}
	set := map[string]bool{}
	for _, c := range all {
		set[c] = true
	}
	for _, c := range need {
		if !set[c] {
			t.Fatalf("catalog missing %s", c)
		}
	}
}

func TestFormatCatalog(t *testing.T) {
	s := FormatCatalog()
	for _, n := range []string{"europe-west1", "Belgium", "Multi-region", "cloud.google.com"} {
		if !strings.Contains(s, n) {
			t.Fatalf("missing %q", n)
		}
	}
}
