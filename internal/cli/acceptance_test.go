package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIValidationDoesNotMutate(t *testing.T) {
	base := []string{"new", "demo", "--module", "example.com/demo", "--entrypoints", "http", "--http", "stdlib"}
	cases := [][]string{nil, {"new"}, {"new", "demo"}, {"new", "demo", "--module", "example.com/demo"}, {"new", "demo", "--entrypoints", "http", "--http", "stdlib"}, {"new", "demo", "--module", "example.com/demo", "--entrypoints", "http"}}
	for _, tail := range [][]string{{"extra"}, {"--unknown"}, {"--module=example.com/other"}, {"--module", "example.com/other"}, {"--http=stdlib"}, {"--http", "stdlib"}, {"--entrypoints=http"}, {"--entrypoints", "http"}} {
		cases = append(cases, append(append([]string{}, base...), tail...))
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			d := t.TempDir()
			var out, err bytes.Buffer
			if code := Run(args, &out, &err, d); code != 2 {
				t.Fatalf("code %d: %s", code, err.String())
			}
			entries, e := os.ReadDir(d)
			if e != nil || len(entries) != 0 {
				t.Fatalf("invalid args mutated parent: %v %v", entries, e)
			}
			if err.Len() == 0 {
				t.Fatal("missing diagnostic")
			}
		})
	}
}

func TestCLISuccessGuidance(t *testing.T) {
	d := t.TempDir()
	var out, err bytes.Buffer
	if code := Run([]string{"new", "demo", "--module=example.com/demo", "--entrypoints=http", "--http=stdlib"}, &out, &err, d); code != 0 {
		t.Fatal(code, err.String())
	}
	if !strings.Contains(out.String(), "demo") || !strings.Contains(out.String(), "README.md") {
		t.Fatal("missing project/README guidance:", out.String())
	}
	data, e := os.ReadFile(filepath.Join(d, "demo", "README.md"))
	if e != nil {
		t.Fatal(e)
	}
	for _, want := range []string{"go mod download", "go build", "go run ./cmd/http", ".env.example", "not automatically loaded"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("README missing %q", want)
		}
	}
}
