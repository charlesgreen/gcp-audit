package gcpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2/google"
	artifactregistry "google.golang.org/api/artifactregistry/v1"
	cloudfunctions "google.golang.org/api/cloudfunctions/v2"
	cloudkms "google.golang.org/api/cloudkms/v1"
	crm "google.golang.org/api/cloudresourcemanager/v3"
	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"
	dns "google.golang.org/api/dns/v1"
	iam "google.golang.org/api/iam/v1"
	logging "google.golang.org/api/logging/v2"
	"google.golang.org/api/option"
	pubsub "google.golang.org/api/pubsub/v1"
	run "google.golang.org/api/run/v2"
	secretmanager "google.golang.org/api/secretmanager/v1"
	serviceusage "google.golang.org/api/serviceusage/v1"
	sqladmin "google.golang.org/api/sqladmin/v1"
	storage "google.golang.org/api/storage/v1"
)

// Live is the production Client using Google APIs and ADC.
type Live struct {
	opts []option.ClientOption
}

// NewLive uses Application Default Credentials.
func NewLive(ctx context.Context) (*Live, error) {
	creds, err := google.FindDefaultCredentials(ctx,
		"https://www.googleapis.com/auth/cloud-platform.read-only",
		"https://www.googleapis.com/auth/cloud-platform",
	)
	if err != nil {
		return nil, err
	}
	return &Live{opts: []option.ClientOption{option.WithTokenSource(creds.TokenSource)}}, nil
}

func (l *Live) Identity(ctx context.Context, project string) (Identity, error) {
	svc, err := crm.NewService(ctx, l.opts...)
	if err != nil {
		return Identity{}, err
	}
	p, err := svc.Projects.Get("projects/" + project).Context(ctx).Do()
	if err != nil {
		return Identity{}, err
	}
	principal := os.Getenv("GOOGLE_ACCOUNT")
	if creds, err := google.FindDefaultCredentials(ctx); err == nil && creds.JSON != nil {
		var raw struct {
			ClientEmail string `json:"client_email"`
		}
		if json.Unmarshal(creds.JSON, &raw) == nil {
			principal = raw.ClientEmail
		}
	}
	num := strings.TrimPrefix(p.Name, "projects/")
	return Identity{
		ProjectID:     p.ProjectId,
		ProjectNumber: num,
		Principal:     principal,
	}, nil
}

func (l *Live) ListRegions(ctx context.Context, project string) ([]string, error) {
	svc, err := compute.NewService(ctx, l.opts...)
	if err != nil {
		return nil, err
	}
	out, err := svc.Regions.List(project).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, r := range out.Items {
		if r.Status == "UP" || r.Status == "" {
			names = append(names, r.Name)
		}
	}
	return names, nil
}

func (l *Live) Dump(ctx context.Context, req Dump) (json.RawMessage, error) {
	project := req.Input["Project"]
	loc := req.Input["Location"]
	if loc == "" {
		loc = req.Location
	}
	key := req.Service + "." + req.Operation
	var (
		v   any
		err error
	)
	switch key {
	case "cloudresourcemanager.GetProject":
		svc, e := crm.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Projects.Get("projects/" + project).Context(ctx).Do()
	case "cloudresourcemanager.GetIamPolicy":
		svc, e := crm.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Projects.GetIamPolicy("projects/"+project, &crm.GetIamPolicyRequest{}).Context(ctx).Do()
	case "iam.ListServiceAccounts":
		svc, e := iam.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Projects.ServiceAccounts.List("projects/" + project).Context(ctx).Do()
	case "iam.ListServiceAccountKeys":
		svc, e := iam.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		name := "projects/" + project + "/serviceAccounts/" + req.Input["ServiceAccount"]
		v, err = svc.Projects.ServiceAccounts.Keys.List(name).Context(ctx).Do()
	case "serviceusage.ListServices":
		svc, e := serviceusage.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Services.List("projects/" + project).Filter("state:ENABLED").Context(ctx).Do()
	case "storage.ListBuckets":
		svc, e := storage.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Buckets.List(project).Context(ctx).Do()
	case "compute.ListNetworks":
		svc, e := compute.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Networks.List(project).Context(ctx).Do()
	case "compute.ListFirewalls":
		svc, e := compute.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Firewalls.List(project).Context(ctx).Do()
	case "compute.AggregatedListInstances":
		svc, e := compute.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Instances.AggregatedList(project).Context(ctx).Do()
	case "compute.AggregatedListDisks":
		svc, e := compute.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Disks.AggregatedList(project).Context(ctx).Do()
	case "compute.AggregatedListVpnTunnels":
		svc, e := compute.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.VpnTunnels.AggregatedList(project).Context(ctx).Do()
	case "compute.ListSubnetworks":
		svc, e := compute.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Subnetworks.List(project, loc).Context(ctx).Do()
	case "compute.ListAddresses":
		svc, e := compute.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Addresses.List(project, loc).Context(ctx).Do()
	case "compute.ListForwardingRules":
		svc, e := compute.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.ForwardingRules.List(project, loc).Context(ctx).Do()
	case "compute.ListRouters":
		svc, e := compute.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Routers.List(project, loc).Context(ctx).Do()
	case "dns.ListManagedZones":
		svc, e := dns.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.ManagedZones.List(project).Context(ctx).Do()
	case "logging.ListSinks":
		svc, e := logging.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Sinks.List("projects/" + project).Context(ctx).Do()
	case "pubsub.ListTopics":
		svc, e := pubsub.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Projects.Topics.List("projects/" + project).Context(ctx).Do()
	case "sqladmin.ListInstances":
		svc, e := sqladmin.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Instances.List(project).Context(ctx).Do()
	case "container.ListClusters":
		svc, e := container.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		parent := "projects/" + project + "/locations/" + loc
		v, err = svc.Projects.Locations.Clusters.List(parent).Context(ctx).Do()
	case "run.ListServices":
		svc, e := run.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		parent := "projects/" + project + "/locations/" + loc
		v, err = svc.Projects.Locations.Services.List(parent).Context(ctx).Do()
	case "cloudfunctions.ListFunctions":
		svc, e := cloudfunctions.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		parent := "projects/" + project + "/locations/" + loc
		v, err = svc.Projects.Locations.Functions.List(parent).Context(ctx).Do()
	case "cloudkms.ListKeyRings":
		svc, e := cloudkms.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		parent := "projects/" + project + "/locations/" + loc
		v, err = svc.Projects.Locations.KeyRings.List(parent).Context(ctx).Do()
	case "secretmanager.ListSecrets":
		svc, e := secretmanager.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		v, err = svc.Projects.Secrets.List("projects/" + project).Context(ctx).Do()
	case "artifactregistry.ListRepositories":
		svc, e := artifactregistry.NewService(ctx, l.opts...)
		if e != nil {
			return nil, e
		}
		parent := "projects/" + project + "/locations/" + loc
		v, err = svc.Projects.Locations.Repositories.List(parent).Context(ctx).Do()
	default:
		return nil, fmt.Errorf("unmapped GCP operation %s", key)
	}
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(v)
	return b, err
}
