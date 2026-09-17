package locations

import (
	"fmt"
	"sort"
	"strings"
)

// Class is a GCP location kind.
type Class string

const (
	ClassRegion      Class = "region"
	ClassMultiRegion Class = "multi-region"
)

// Info is one GCP location.
// https://cloud.google.com/about/locations
type Info struct {
	Code  string
	Name  string
	Class Class
}

var catalog = []Info{
	{"africa-south1", "Johannesburg", ClassRegion},
	{"asia-east1", "Taiwan", ClassRegion},
	{"asia-east2", "Hong Kong", ClassRegion},
	{"asia-northeast1", "Tokyo", ClassRegion},
	{"asia-northeast2", "Osaka", ClassRegion},
	{"asia-northeast3", "Seoul", ClassRegion},
	{"asia-south1", "Mumbai", ClassRegion},
	{"asia-south2", "Delhi", ClassRegion},
	{"asia-southeast1", "Singapore", ClassRegion},
	{"asia-southeast2", "Jakarta", ClassRegion},
	{"australia-southeast1", "Sydney", ClassRegion},
	{"australia-southeast2", "Melbourne", ClassRegion},
	{"europe-central2", "Warsaw", ClassRegion},
	{"europe-north1", "Finland", ClassRegion},
	{"europe-north2", "Stockholm", ClassRegion},
	{"europe-southwest1", "Madrid", ClassRegion},
	{"europe-west1", "Belgium", ClassRegion},
	{"europe-west2", "London", ClassRegion},
	{"europe-west3", "Frankfurt", ClassRegion},
	{"europe-west4", "Netherlands", ClassRegion},
	{"europe-west6", "Zurich", ClassRegion},
	{"europe-west8", "Milan", ClassRegion},
	{"europe-west9", "Paris", ClassRegion},
	{"europe-west10", "Berlin", ClassRegion},
	{"europe-west12", "Turin", ClassRegion},
	{"me-central1", "Doha", ClassRegion},
	{"me-central2", "Dammam", ClassRegion},
	{"me-west1", "Tel Aviv", ClassRegion},
	{"northamerica-northeast1", "Montréal", ClassRegion},
	{"northamerica-northeast2", "Toronto", ClassRegion},
	{"northamerica-south1", "Mexico", ClassRegion},
	{"southamerica-east1", "São Paulo", ClassRegion},
	{"southamerica-west1", "Santiago", ClassRegion},
	{"us-central1", "Iowa", ClassRegion},
	{"us-east1", "South Carolina", ClassRegion},
	{"us-east4", "Northern Virginia", ClassRegion},
	{"us-east5", "Columbus", ClassRegion},
	{"us-south1", "Dallas", ClassRegion},
	{"us-west1", "Oregon", ClassRegion},
	{"us-west2", "Los Angeles", ClassRegion},
	{"us-west3", "Salt Lake City", ClassRegion},
	{"us-west4", "Las Vegas", ClassRegion},
	{"asia", "Asia (multi-region)", ClassMultiRegion},
	{"eu", "European Union (multi-region)", ClassMultiRegion},
	{"us", "United States (multi-region)", ClassMultiRegion},
	{"global", "Global", ClassMultiRegion},
}

var byCode = func() map[string]Info {
	m := make(map[string]Info, len(catalog))
	for _, l := range catalog {
		m[l.Code] = l
	}
	return m
}()

// Lookup returns catalog data for a location code.
func Lookup(code string) (Info, bool) {
	l, ok := byCode[strings.ToLower(strings.TrimSpace(code))]
	return l, ok
}

// Codes returns location codes of a class.
func Codes(class Class) []string {
	var out []string
	for _, l := range catalog {
		if l.Class == class {
			out = append(out, l.Code)
		}
	}
	return out
}

// RegionCodes is every regional location (not multi-region or global).
func RegionCodes() []string {
	return Codes(ClassRegion)
}

// ParseRequested splits --locations and rejects unknown codes.
func ParseRequested(csv string) ([]string, error) {
	var out []string
	var unknown []string
	seen := map[string]bool{}
	for _, p := range strings.Split(csv, ",") {
		code := strings.ToLower(strings.TrimSpace(p))
		if code == "" {
			continue
		}
		if _, ok := Lookup(code); !ok {
			unknown = append(unknown, code)
			continue
		}
		if !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown location code(s): %s", strings.Join(unknown, ","))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("--locations did not contain any location codes")
	}
	return out, nil
}

// FilterRequested keeps codes that are available on the project.
func FilterRequested(requested, available []string) ([]string, error) {
	have := map[string]bool{}
	for _, c := range available {
		have[c] = true
	}
	var out, missing []string
	for _, code := range requested {
		if !have[code] {
			info, _ := Lookup(code)
			missing = append(missing, fmt.Sprintf("%s  %s", code, info.Name))
			continue
		}
		out = append(out, code)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("location(s) not available on this project:\n  %s", strings.Join(missing, "\n  "))
	}
	sort.Strings(out)
	return out, nil
}

// FormatCatalog prints the valid location list.
func FormatCatalog() string {
	var b strings.Builder
	b.WriteString("Valid GCP location codes\n")
	b.WriteString("https://cloud.google.com/about/locations\n")
	b.WriteString("Pass a comma-separated subset with --locations (example: --locations europe-west1,asia-northeast1).\n\n")
	b.WriteString("Regions:\n")
	for _, l := range catalog {
		if l.Class == ClassRegion {
			fmt.Fprintf(&b, "  %-24s  %s\n", l.Code, l.Name)
		}
	}
	b.WriteString("\nMulti-region / global:\n")
	for _, l := range catalog {
		if l.Class == ClassMultiRegion {
			fmt.Fprintf(&b, "  %-24s  %s\n", l.Code, l.Name)
		}
	}
	return b.String()
}
