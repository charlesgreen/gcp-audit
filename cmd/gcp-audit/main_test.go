package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestListLocationsNoGCP(t *testing.T) {
	exe := build(t)
	out, err := exec.Command(exe, "--list-locations").CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(string(out), "europe-west1") {
		t.Fatalf("%s", out)
	}
}

func TestUnknownLocationExit2(t *testing.T) {
	exe := build(t)
	cmd := exec.Command(exe, "--locations", "us-east-1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected failure")
	}
	if cmd.ProcessState.ExitCode() != 2 {
		t.Fatalf("exit %d\n%s", cmd.ProcessState.ExitCode(), out)
	}
	if !strings.Contains(string(out), "us-east-1") {
		t.Fatalf("%s", out)
	}
}

func build(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "gcp-audit")
	cmd := exec.Command("go", "build", "-o", exe, ".")
	cmd.Dir = filepath.Join(findMod(t), "cmd", "gcp-audit")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return exe
}

func findMod(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
