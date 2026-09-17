package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Finding is one portable audit row (shared shape for future GCP/AWS merge).
type Finding struct {
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Resource string `json:"resource,omitempty"`
}

// Report is the exported audit document.
type Report struct {
	Cloud     string    `json:"cloud"`
	Project   string    `json:"project"`
	Principal string    `json:"principal,omitempty"`
	Generated string    `json:"generated"`
	DumpDir   string    `json:"dump_dir,omitempty"`
	Findings  []Finding `json:"findings"`
	Notes     []string  `json:"notes,omitempty"`
	Errors    int       `json:"error_lines"`
}

// New empty report for a GCP project.
func New(project, principal, dumpDir string) *Report {
	return &Report{
		Cloud:     "gcp",
		Project:   project,
		Principal: principal,
		Generated: time.Now().UTC().Format(time.RFC3339),
		DumpDir:   dumpDir,
	}
}

func (r *Report) Add(severity, title, detail, resource string) {
	r.Findings = append(r.Findings, Finding{Severity: severity, Title: title, Detail: detail, Resource: resource})
}

func (r *Report) Note(msg string) {
	r.Notes = append(r.Notes, msg)
}

// Markdown is the auditor-facing narrative.
func (r *Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# GCP Configuration Audit — Project %s\n\n", r.Project)
	if r.Principal != "" {
		fmt.Fprintf(&b, "- **Principal:** %s\n", r.Principal)
	}
	fmt.Fprintf(&b, "- **Generated:** %s\n", r.Generated)
	if r.DumpDir != "" {
		fmt.Fprintf(&b, "- **Raw data:** `%s`\n", r.DumpDir)
	}
	b.WriteString("\n> This is an automated read-only audit. Verify each finding before acting.\n")
	b.WriteString("> Failed API calls are in `errors.log` and may produce false negatives.\n")

	if len(r.Notes) > 0 {
		b.WriteString("\n## Coverage\n\n")
		for _, n := range r.Notes {
			fmt.Fprintf(&b, "- %s\n", n)
		}
	}

	b.WriteString("\n## Findings\n\n")
	if len(r.Findings) == 0 {
		b.WriteString("No automated findings.\n")
	} else {
		for _, f := range r.Findings {
			if f.Resource != "" {
				fmt.Fprintf(&b, "- **%s: %s** — %s (`%s`)\n", f.Severity, f.Title, f.Detail, f.Resource)
			} else {
				fmt.Fprintf(&b, "- **%s: %s** — %s\n", f.Severity, f.Title, f.Detail)
			}
		}
	}
	if r.Errors > 0 {
		fmt.Fprintf(&b, "\n## Permission / API Errors\n\n- %d log lines in `errors.log` — review for 403/404 to identify blind spots.\n", r.Errors)
	}
	return b.String()
}

// CSV is a spreadsheet/GRC export.
func (r *Report) CSV() (string, error) {
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write([]string{"cloud", "project", "severity", "title", "detail", "resource"}); err != nil {
		return "", err
	}
	for _, f := range r.Findings {
		if err := w.Write([]string{r.Cloud, r.Project, f.Severity, f.Title, f.Detail, f.Resource}); err != nil {
			return "", err
		}
	}
	w.Flush()
	return b.String(), w.Error()
}

// JSONBytes is the machine-readable export.
func (r *Report) JSONBytes() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
