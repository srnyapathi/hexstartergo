package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want int
	}{{[]string{"--help"}, 0}, {[]string{"new", "--help"}, 0}, {[]string{"new", "demo", "--module", "example.com/demo", "--entrypoints", "http", "--http", "stdlib"}, 0}, {[]string{"new", "demo", "--module", "example.com/demo", "--module", "example.com/other", "--entrypoints", "http", "--http", "stdlib"}, 2}, {[]string{"new", "demo", "--unknown"}, 2}, {nil, 2}} {
		var out, err bytes.Buffer
		d := t.TempDir()
		if got := Run(tc.args, &out, &err, d); got != tc.want {
			t.Errorf("%v: got %d want %d (%s)", tc.args, got, tc.want, err.String())
		}
	}
}
func TestRunExistingDestination(t *testing.T) {
	d := t.TempDir()
	os.Mkdir(filepath.Join(d, "demo"), 0755)
	var out, err bytes.Buffer
	if got := Run([]string{"new", "demo", "--module", "example.com/demo", "--entrypoints", "http", "--http", "stdlib"}, &out, &err, d); got != 1 {
		t.Fatal(got, err.String())
	}
}
func TestApprovedDatabaseFlag(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"new", "demo", "--module", "example.com/demo", "--entrypoints", "http", "--http", "stdlib", "--database", "postgres"}, &out, &err, t.TempDir())
	if code != 2 || strings.Contains(err.String(), "flag provided but not defined") {
		t.Fatalf("database flag was not parsed: %d %s", code, err.String())
	}
	err.Reset()
	out.Reset()
	if code := Run([]string{"new", "demo", "--module", "example.com/demo", "--entrypoints", "http", "--http", "stdlib", "--db="}, &out, &err, t.TempDir()); code != 2 {
		t.Fatal("unapproved db alias accepted")
	}
}
func TestRejectEmptyInapplicableFlags(t *testing.T) {
	for _, flag := range []string{"--broker=", "--schedule="} {
		var out, err bytes.Buffer
		code := Run([]string{"new", "demo", "--module", "example.com/demo", "--entrypoints", "http", "--http", "stdlib", flag}, &out, &err, t.TempDir())
		if code != 2 {
			t.Errorf("accepted %s: %d", flag, code)
		}
	}
	var out, err bytes.Buffer
	code := Run([]string{"new", "demo", "--module", "example.com/demo", "--entrypoints", "grpc", "--http="}, &out, &err, t.TempDir())
	if code != 2 || !strings.Contains(err.String(), "--http requires http") {
		t.Errorf("lost http flag presence: %d %s", code, err.String())
	}
}
