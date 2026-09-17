package summarize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charlesgreen/gcp-audit/internal/report"
)

// Build reads a gcp-audit dump directory and returns a portable report.
func Build(dir string) (*report.Report, error) {
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return nil, fmt.Errorf("Usage: gcp-audit-summarize <audit-output-dir>")
	}
	meta := filepath.Join(dir, "meta.json")
	project := sj(meta, "project_id")
	principal := sj(meta, "principal")
	r := report.New(project, principal, dir)

	var m struct {
		LocationsScanned []string `json:"locations_scanned"`
	}
	_ = readJSON(meta, &m)
	if len(m.LocationsScanned) > 0 {
		r.Note(fmt.Sprintf("Locations scanned (%d): %s", len(m.LocationsScanned), strings.Join(m.LocationsScanned, ",")))
	}

	iamPolicy(dir, r)
	firewalls(dir, r)
	networks(dir, r)
	sql(dir, r)
	gke(dir, r)
	saKeys(dir, r)
	logging(dir, r)
	r.Errors = errorLines(filepath.Join(dir, "errors.log"))
	return r, nil
}

func iamPolicy(dir string, r *report.Report) {
	var doc struct {
		Bindings []struct {
			Role    string   `json:"role"`
			Members []string `json:"members"`
		} `json:"bindings"`
	}
	_ = readJSON(filepath.Join(dir, "global", "iam-policy.json"), &doc)
	for _, b := range doc.Bindings {
		if b.Role == "roles/owner" || b.Role == "roles/editor" {
			r.Add("HIGH", "Primitive project IAM role",
				fmt.Sprintf("%s granted to %s — prefer predefined/custom roles.", b.Role, strings.Join(b.Members, ", ")),
				b.Role)
		}
		for _, mem := range b.Members {
			if strings.HasPrefix(mem, "allUsers") || strings.HasPrefix(mem, "allAuthenticatedUsers") {
				r.Add("HIGH", "Public IAM member on project", mem+" has "+b.Role, b.Role)
			}
		}
	}
}

func firewalls(dir string, r *report.Report) {
	var doc struct {
		Items []struct {
			Name         string   `json:"name"`
			SourceRanges []string `json:"sourceRanges"`
			Allowed      []struct {
				IPProtocol string   `json:"IPProtocol"`
				Ports      []string `json:"ports"`
			} `json:"allowed"`
			Disabled bool `json:"disabled"`
		} `json:"items"`
	}
	_ = readJSON(filepath.Join(dir, "global", "compute", "firewalls.json"), &doc)
	for _, fw := range doc.Items {
		if fw.Disabled {
			continue
		}
		open := false
		for _, cidr := range fw.SourceRanges {
			if cidr == "0.0.0.0/0" || cidr == "::/0" {
				open = true
			}
		}
		if !open {
			continue
		}
		var ports []string
		for _, a := range fw.Allowed {
			if len(a.Ports) == 0 {
				ports = append(ports, a.IPProtocol+"/all")
			} else {
				for _, p := range a.Ports {
					ports = append(ports, a.IPProtocol+"/"+p)
				}
			}
		}
		r.Add("HIGH", "Firewall allows ingress from the public internet",
			strings.Join(ports, ", "), fw.Name)
	}
}

func networks(dir string, r *report.Report) {
	var doc struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	_ = readJSON(filepath.Join(dir, "global", "compute", "networks.json"), &doc)
	for _, n := range doc.Items {
		if n.Name == "default" {
			r.Add("LOW", "Default VPC network present", "Delete unused default networks.", n.Name)
		}
	}
}

func sql(dir string, r *report.Report) {
	matches, _ := filepath.Glob(filepath.Join(dir, "locations", "*", "sql", "instances.json"))
	for _, f := range matches {
		var doc struct {
			Items []struct {
				Name     string `json:"name"`
				Settings struct {
					IpConfiguration struct {
						Ipv4Enabled bool `json:"ipv4Enabled"`
					} `json:"ipConfiguration"`
				} `json:"settings"`
			} `json:"items"`
		}
		_ = readJSON(f, &doc)
		for _, inst := range doc.Items {
			if inst.Settings.IpConfiguration.Ipv4Enabled {
				r.Add("HIGH", "Cloud SQL instance has a public IPv4 address",
					"Disable ipv4Enabled and use private IP or Cloud SQL Auth Proxy.", inst.Name)
			}
		}
	}
}

func gke(dir string, r *report.Report) {
	matches, _ := filepath.Glob(filepath.Join(dir, "locations", "*", "container", "clusters.json"))
	for _, f := range matches {
		var doc struct {
			Clusters []struct {
				Name       string `json:"name"`
				LegacyAbac struct {
					Enabled bool `json:"enabled"`
				} `json:"legacyAbac"`
				PrivateClusterConfig struct {
					EnablePrivateNodes bool `json:"enablePrivateNodes"`
				} `json:"privateClusterConfig"`
			} `json:"clusters"`
		}
		_ = readJSON(f, &doc)
		for _, c := range doc.Clusters {
			if c.LegacyAbac.Enabled {
				r.Add("HIGH", "GKE cluster has legacy ABAC enabled", "Disable ABAC; use RBAC only.", c.Name)
			}
			if !c.PrivateClusterConfig.EnablePrivateNodes {
				r.Add("MEDIUM", "GKE cluster nodes are not private", "Enable private nodes.", c.Name)
			}
		}
	}
}

func saKeys(dir string, r *report.Report) {
	matches, _ := filepath.Glob(filepath.Join(dir, "global", "iam", "keys", "*.json"))
	for _, f := range matches {
		var doc struct {
			Keys []struct {
				KeyType        string `json:"keyType"`
				ValidAfterTime string `json:"validAfterTime"`
				Name           string `json:"name"`
			} `json:"keys"`
		}
		_ = readJSON(f, &doc)
		n := 0
		for _, k := range doc.Keys {
			if k.KeyType == "USER_MANAGED" {
				n++
			}
		}
		if n > 0 {
			email := strings.TrimSuffix(filepath.Base(f), ".json")
			r.Add("MEDIUM", "User-managed service account keys present",
				fmt.Sprintf("%d user-managed key(s); prefer Workload Identity.", n), email)
		}
	}
}

func logging(dir string, r *report.Report) {
	var doc struct {
		Sinks []struct {
			Name string `json:"name"`
		} `json:"sinks"`
	}
	_ = readJSON(filepath.Join(dir, "global", "logging", "sinks.json"), &doc)
	if len(doc.Sinks) == 0 {
		r.Add("MEDIUM", "No Cloud Logging sinks found",
			"Export audit logs to a locked destination for SOC 2 / ISO 27001 evidence.", "")
	}
}

func readJSON(path string, v any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

func sj(path, key string) string {
	var m map[string]any
	if readJSON(path, &m) != nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func errorLines(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}
