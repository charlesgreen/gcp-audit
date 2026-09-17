package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charlesgreen/gcp-audit/internal/summarize"
)

// Set by GoReleaser ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("gcp-audit-summarize %s (commit %s, date %s)\n", Version, Commit, Date)
		os.Exit(0)
	}
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintf(os.Stderr, "Usage: %s <audit-output-dir> [--format md|csv|json]\n", os.Args[0])
		os.Exit(2)
	}
	format := "md"
	dir := os.Args[1]
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--format" && i+1 < len(os.Args) {
			format = os.Args[i+1]
			i++
		}
	}
	rep, err := summarize.Build(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	switch strings.ToLower(format) {
	case "md", "markdown":
		fmt.Print(rep.Markdown())
	case "csv":
		s, err := rep.CSV()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Print(s)
	case "json":
		b, err := rep.JSONBytes()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(b))
	default:
		fmt.Fprintf(os.Stderr, "unknown --format %s\n", format)
		os.Exit(2)
	}
}
