package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charlesgreen/gcp-audit/internal/collect"
	"github.com/charlesgreen/gcp-audit/internal/gcpapi"
	"github.com/charlesgreen/gcp-audit/internal/locations"
)

// Set by GoReleaser ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	opts := collect.Options{
		Project:  firstEnv("GOOGLE_CLOUD_PROJECT", "CLOUDSDK_CORE_PROJECT", "GCP_PROJECT"),
		Parallel: 4,
		Formats:  []string{"md", "csv", "json"},
	}
	listOnly := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		need := func() (string, bool) {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return "", false
			}
			i++
			return args[i], true
		}
		switch a {
		case "-h", "--help":
			fmt.Print(usage())
			return 0
		case "--version":
			fmt.Printf("gcp-audit %s (commit %s, date %s)\n", Version, Commit, Date)
			return 0
		case "--list-locations":
			listOnly = true
		case "--project":
			v, ok := need()
			if !ok {
				fmt.Fprintln(os.Stderr, "error: --project requires a project id")
				return 2
			}
			opts.Project = v
		case "--locations":
			v, ok := need()
			if !ok {
				fmt.Fprintln(os.Stderr, "error: --locations requires a comma-separated list (see --list-locations)")
				return 2
			}
			opts.RequestedCSV = v
		case "--out":
			v, ok := need()
			if !ok {
				fmt.Fprintln(os.Stderr, "error: --out requires a directory")
				return 2
			}
			opts.OutDir = v
		case "--parallel":
			v, ok := need()
			if !ok {
				fmt.Fprintln(os.Stderr, "error: --parallel requires a number")
				return 2
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				fmt.Fprintln(os.Stderr, "error: --parallel must be a positive integer")
				return 2
			}
			opts.Parallel = n
		case "--format":
			v, ok := need()
			if !ok {
				fmt.Fprintln(os.Stderr, "error: --format requires md,csv,json")
				return 2
			}
			opts.Formats = splitCSV(v)
		default:
			fmt.Fprintf(os.Stderr, "unknown arg: %s (try --help or --list-locations)\n", a)
			return 2
		}
	}
	if listOnly {
		fmt.Print(locations.FormatCatalog())
		return 0
	}
	if opts.RequestedCSV != "" {
		if _, err := locations.ParseRequested(opts.RequestedCSV); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n\n%s", err, locations.FormatCatalog())
			return 2
		}
	}
	live, err := gcpapi.NewLive(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load GCP credentials (ADC): %v\n", err)
		return 1
	}
	opts.Client = live
	if err := collect.Run(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if opts.RequestedCSV != "" {
			return 2
		}
		return 1
	}
	return 0
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func usage() string {
	return `Usage:
  gcp-audit --project ID [--locations l1,l2,...] [--out DIR] [--parallel N] [--format md,csv,json]
  gcp-audit --list-locations
  gcp-audit --version
  gcp-audit -h|--help

  --project ID         GCP project id (or GOOGLE_CLOUD_PROJECT)
  --locations l1,l2    Scan only these location codes (must be valid GCP locations)
  --out DIR            Output directory (default: ./gcp-audit-<project>-<timestamp>)
  --parallel N         Concurrent location audits (default: 4)
  --format LIST        Report formats: md, csv, json (default: all three)
  --list-locations     Print valid location codes and exit (no GCP calls)
  --version            Print version and exit
  -h, --help           Show this help and the valid location list

Default scan: every compute region available to the project (worldwide).

` + locations.FormatCatalog()
}
