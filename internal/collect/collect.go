package collect

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charlesgreen/gcp-audit/internal/gcpapi"
	"github.com/charlesgreen/gcp-audit/internal/locations"
	"github.com/charlesgreen/gcp-audit/internal/redact"
	"github.com/charlesgreen/gcp-audit/internal/report"
	"github.com/charlesgreen/gcp-audit/internal/summarize"
)

const Version = "gcp-audit/1.0"

// Options configure a collection run.
type Options struct {
	Project      string
	OutDir       string
	RequestedCSV string
	Parallel     int
	Formats      []string // md, csv, json
	Client       gcpapi.Client
	Now          time.Time
}

// Run authenticates, dumps configuration JSON, and writes reports.
func Run(ctx context.Context, opts Options) error {
	if opts.Parallel <= 0 {
		opts.Parallel = 4
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if len(opts.Formats) == 0 {
		opts.Formats = []string{"md", "csv", "json"}
	}
	if opts.Client == nil {
		return fmt.Errorf("GCP client is required")
	}
	if strings.TrimSpace(opts.Project) == "" {
		return fmt.Errorf("--project is required (or set GOOGLE_CLOUD_PROJECT)")
	}

	var requested []string
	if strings.TrimSpace(opts.RequestedCSV) != "" {
		var err error
		requested, err = locations.ParseRequested(opts.RequestedCSV)
		if err != nil {
			return fmt.Errorf("%w\n\n%s", err, locations.FormatCatalog())
		}
	}

	id, err := opts.Client.Identity(ctx, opts.Project)
	if err != nil {
		return fmt.Errorf("failed to load project %q: %w", opts.Project, err)
	}

	available, src := discover(ctx, opts.Client, opts.Project)
	scanned := available
	if requested != nil {
		scanned, err = locations.FilterRequested(requested, append(available, "global", "us", "eu", "asia"))
		if err != nil {
			return err
		}
	}
	if len(scanned) == 0 {
		return fmt.Errorf("no locations resolved")
	}

	out := opts.OutDir
	if out == "" {
		out = fmt.Sprintf("./gcp-audit-%s-%s", id.ProjectID, opts.Now.Format("20060102-150405"))
	}
	if err := os.MkdirAll(out, 0o700); err != nil {
		return err
	}
	_ = os.Chmod(out, 0o700)
	opts.OutDir = out

	c := &collector{opts: opts, id: id, errLog: filepath.Join(out, "errors.log")}
	if err := os.WriteFile(c.errLog, nil, 0o600); err != nil {
		return err
	}

	meta := map[string]any{
		"cloud":               "gcp",
		"project_id":          id.ProjectID,
		"project_number":      id.ProjectNumber,
		"principal":           id.Principal,
		"started_at_utc":      opts.Now.Format("20060102-150405"),
		"tool_version":        Version,
		"locations_scanned":   scanned,
		"locations_source":    src,
		"locations_requested": requested,
	}
	if requested == nil {
		meta["locations_requested"] = []string{}
	}
	if err := c.writeJSON(filepath.Join(out, "meta.json"), meta); err != nil {
		return err
	}

	c.auditGlobal(ctx)
	c.auditLocations(ctx, scanned)

	rep, err := summarize.Build(out)
	if err != nil {
		return err
	}
	return writeReports(out, rep, opts.Formats)
}

func discover(ctx context.Context, cl gcpapi.Client, project string) ([]string, string) {
	regs, err := cl.ListRegions(ctx, project)
	if err == nil && len(regs) > 0 {
		return regs, "compute.regions.list"
	}
	return locations.RegionCodes(), "catalog"
}

type collector struct {
	opts   Options
	id     gcpapi.Identity
	errMu  sync.Mutex
	errLog string
}

func (c *collector) dump(ctx context.Context, path string, req gcpapi.Dump) json.RawMessage {
	if req.Input == nil {
		req.Input = map[string]string{}
	}
	if req.Input["Project"] == "" {
		req.Input["Project"] = c.id.ProjectID
	}
	raw, err := c.opts.Client.Dump(ctx, req)
	if err != nil {
		c.logErr(req, err)
		raw = json.RawMessage(`{}`)
	}
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if red, rerr := redact.JSON(raw); rerr == nil {
		raw = red
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, raw, 0o600)
	return raw
}

func (c *collector) writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

func (c *collector) logErr(req gcpapi.Dump, err error) {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	f, e := os.OpenFile(c.errLog, os.O_APPEND|os.O_WRONLY, 0o600)
	if e != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] FAIL location=%s :: %s %s %v\n",
		time.Now().UTC().Format(time.RFC3339), req.Location, req.Service, req.Operation, err)
}

func (c *collector) auditGlobal(ctx context.Context) {
	g := filepath.Join(c.opts.OutDir, "global")
	c.dump(ctx, filepath.Join(g, "project.json"), gcpapi.Dump{Service: "cloudresourcemanager", Operation: "GetProject"})
	c.dump(ctx, filepath.Join(g, "iam-policy.json"), gcpapi.Dump{Service: "cloudresourcemanager", Operation: "GetIamPolicy"})
	sas := c.dump(ctx, filepath.Join(g, "iam", "service-accounts.json"), gcpapi.Dump{Service: "iam", Operation: "ListServiceAccounts"})
	for _, email := range fieldList(sas, "accounts", "email") {
		c.dump(ctx, filepath.Join(g, "iam", "keys", sanitize(email)+".json"), gcpapi.Dump{
			Service: "iam", Operation: "ListServiceAccountKeys",
			Input: map[string]string{"ServiceAccount": email},
		})
	}
	c.dump(ctx, filepath.Join(g, "serviceusage", "services.json"), gcpapi.Dump{Service: "serviceusage", Operation: "ListServices"})
	c.dump(ctx, filepath.Join(g, "storage", "buckets.json"), gcpapi.Dump{Service: "storage", Operation: "ListBuckets"})
	c.dump(ctx, filepath.Join(g, "compute", "networks.json"), gcpapi.Dump{Service: "compute", Operation: "ListNetworks"})
	c.dump(ctx, filepath.Join(g, "compute", "firewalls.json"), gcpapi.Dump{Service: "compute", Operation: "ListFirewalls"})
	c.dump(ctx, filepath.Join(g, "compute", "instances-aggregated.json"), gcpapi.Dump{Service: "compute", Operation: "AggregatedListInstances"})
	c.dump(ctx, filepath.Join(g, "compute", "disks-aggregated.json"), gcpapi.Dump{Service: "compute", Operation: "AggregatedListDisks"})
	c.dump(ctx, filepath.Join(g, "compute", "vpn-tunnels-aggregated.json"), gcpapi.Dump{Service: "compute", Operation: "AggregatedListVpnTunnels"})
	c.dump(ctx, filepath.Join(g, "dns", "managed-zones.json"), gcpapi.Dump{Service: "dns", Operation: "ListManagedZones"})
	c.dump(ctx, filepath.Join(g, "logging", "sinks.json"), gcpapi.Dump{Service: "logging", Operation: "ListSinks"})
	c.dump(ctx, filepath.Join(g, "pubsub", "topics.json"), gcpapi.Dump{Service: "pubsub", Operation: "ListTopics"})
}

func (c *collector) auditLocations(ctx context.Context, scanned []string) {
	sem := make(chan struct{}, c.opts.Parallel)
	var wg sync.WaitGroup
	for _, loc := range scanned {
		if loc == "global" || loc == "us" || loc == "eu" || loc == "asia" {
			continue
		}
		loc := loc
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			c.auditLocation(ctx, loc)
		}()
	}
	wg.Wait()
}

func (c *collector) auditLocation(ctx context.Context, loc string) {
	base := filepath.Join(c.opts.OutDir, "locations", loc)
	d := func(rel, svc, op string) {
		c.dump(ctx, filepath.Join(base, rel), gcpapi.Dump{Location: loc, Service: svc, Operation: op, Input: map[string]string{"Location": loc}})
	}
	d("sql/instances.json", "sqladmin", "ListInstances")
	d("container/clusters.json", "container", "ListClusters")
	d("run/services.json", "run", "ListServices")
	d("functions/functions.json", "cloudfunctions", "ListFunctions")
	d("kms/key-rings.json", "cloudkms", "ListKeyRings")
	d("secretmanager/secrets.json", "secretmanager", "ListSecrets")
	d("artifactregistry/repositories.json", "artifactregistry", "ListRepositories")
	d("compute/subnetworks.json", "compute", "ListSubnetworks")
	d("compute/addresses.json", "compute", "ListAddresses")
	d("compute/forwarding-rules.json", "compute", "ListForwardingRules")
	d("compute/routers.json", "compute", "ListRouters")
}

func fieldList(raw json.RawMessage, array, field string) []string {
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	arr, _ := doc[array].([]any)
	var out []string
	for _, item := range arr {
		m, _ := item.(map[string]any)
		if s, ok := m[field].(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, string(rune(0)), "_")
	return s
}

func writeReports(dir string, rep *report.Report, formats []string) error {
	for _, f := range formats {
		switch strings.ToLower(strings.TrimSpace(f)) {
		case "md", "markdown":
			if err := os.WriteFile(filepath.Join(dir, "summary.md"), []byte(rep.Markdown()), 0o600); err != nil {
				return err
			}
		case "csv":
			s, err := rep.CSV()
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, "summary.csv"), []byte(s), 0o600); err != nil {
				return err
			}
		case "json":
			b, err := rep.JSONBytes()
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, "summary.json"), append(b, '\n'), 0o600); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown --format %q (use md, csv, json)", f)
		}
	}
	return nil
}
