package generate

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptionDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		field, value, choice string
		mutate               func(*Options)
	}{
		{"entrypoints", "bogus", "http", func(o *Options) { o.Entrypoints = []string{"bogus"} }},
		{"http", "bogus", "stdlib", func(o *Options) { o.HTTP = "bogus" }},
		{"database", "sqlite", "postgres", func(o *Options) { o.Database = "sqlite" }},
		{"broker", "bogus", "rabbitmq", func(o *Options) { o.Entrypoints = []string{"consumer"}; o.HTTP = ""; o.Broker = "bogus" }},
		{"schedule", "bogus", "cron", func(o *Options) { o.Entrypoints = []string{"scheduler"}; o.HTTP = ""; o.Schedule = "bogus" }},
	} {
		t.Run(tc.field, func(t *testing.T) {
			o := good()
			tc.mutate(&o)
			e := o.Validate()
			if e == nil {
				t.Fatal("accepted invalid options")
			}
			for _, want := range []string{tc.field, tc.value, tc.choice} {
				if !strings.Contains(e.Error(), want) {
					t.Errorf("%q missing %q", e, want)
				}
			}
		})
	}
	for _, o := range []Options{
		{Name: "demo", Module: "example.com/demo", Entrypoints: []string{"http"}, HTTP: "chi"},
		{Name: "demo", Module: "example.com/demo", Entrypoints: []string{"grpc"}, Database: "postgres"},
		{Name: "demo", Module: "example.com/demo", Entrypoints: []string{"consumer"}, Broker: "rabbitmq"},
		{Name: "demo", Module: "example.com/demo", Entrypoints: []string{"consumer"}, Broker: "kafka"},
		{Name: "demo", Module: "example.com/demo", Entrypoints: []string{"scheduler"}, Schedule: "cron"},
	} {
		if e := o.Validate(); e != nil {
			t.Errorf("supported selection %+v: %v", o, e)
		}
	}
}

func TestWriterExistingObjectsRemain(t *testing.T) {
	for _, kind := range []string{"empty directory", "file", "sentinel", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			parent := t.TempDir()
			dest := filepath.Join(parent, "demo")
			switch kind {
			case "file":
				if e := os.WriteFile(dest, []byte("keep"), 0600); e != nil {
					t.Fatal(e)
				}
			case "symlink":
				target := t.TempDir()
				if e := os.Symlink(target, dest); e != nil {
					t.Skipf("symlink unavailable: %v", e)
				}
			default:
				if e := os.Mkdir(dest, 0755); e != nil {
					t.Fatal(e)
				}
				if kind == "sentinel" {
					if e := os.WriteFile(filepath.Join(dest, "sentinel"), []byte("keep"), 0600); e != nil {
						t.Fatal(e)
					}
				}
			}
			before, e := os.Lstat(dest)
			if e != nil {
				t.Fatal(e)
			}
			if e := Write(parent, "demo", []File{{Path: "new", Data: []byte("bad")}}); e == nil {
				t.Fatal("accepted existing destination")
			}
			after, e := os.Lstat(dest)
			if e != nil || !os.SameFile(before, after) {
				t.Fatal("destination changed", e)
			}
			if kind == "file" || kind == "sentinel" {
				p := dest
				if kind == "sentinel" {
					p = filepath.Join(dest, "sentinel")
				}
				b, e := os.ReadFile(p)
				if e != nil || string(b) != "keep" {
					t.Fatal("existing content changed", e)
				}
			}
		})
	}
}

func TestWriterForeignCleanupReportsFailure(t *testing.T) {
	parent := t.TempDir()
	boom := errors.New("injected failure")
	e := write(parent, "demo", []File{{Path: "nested/owned"}}, func(io.Writer, []byte) error {
		if e := os.WriteFile(filepath.Join(parent, "demo", "nested", "foreign"), []byte("keep"), 0600); e != nil {
			t.Fatal(e)
		}
		return boom
	})
	if !errors.Is(e, boom) || !strings.Contains(e.Error(), "cleanup") {
		t.Fatal("write and cleanup errors not joined", e)
	}
	if _, e := os.Stat(filepath.Join(parent, "demo", "nested", "owned")); !os.IsNotExist(e) {
		t.Fatal("owned file remains", e)
	}
	b, e := os.ReadFile(filepath.Join(parent, "demo", "nested", "foreign"))
	if e != nil || string(b) != "keep" {
		t.Fatal("foreign file removed", e)
	}
}

func TestWriterPreservesReplacedDestination(t *testing.T) {
	parent := t.TempDir()
	boom := errors.New("open root failed")
	e := writeWithOpenRoot(parent, "demo", nil, func(io.Writer, []byte) error { return nil }, func(root *os.Root, name string) (*os.Root, error) {
		if e := os.Rename(filepath.Join(parent, name), filepath.Join(parent, "original")); e != nil {
			t.Fatal(e)
		}
		if e := os.Mkdir(filepath.Join(parent, name), 0755); e != nil {
			t.Fatal(e)
		}
		return nil, boom
	})
	if !errors.Is(e, boom) || !strings.Contains(e.Error(), "cleanup preserved replaced destination") {
		t.Fatal("missing joined replacement diagnostic", e)
	}
	if _, e := os.Stat(filepath.Join(parent, "demo")); e != nil {
		t.Fatal("replacement removed", e)
	}
}

func TestWriterPrevalidatesAncestorCaseAliases(t *testing.T) {
	for _, paths := range [][]string{{"A/one", "a/two"}, {"top/A/one", "top/a/two"}, {"A/one", "a"}, {"a", "A/one"}} {
		t.Run(strings.Join(paths, "+"), func(t *testing.T) {
			parent := t.TempDir()
			called := false
			files := []File{{Path: paths[0]}, {Path: paths[1]}}
			e := write(parent, "demo", files, func(io.Writer, []byte) error { called = true; return nil })
			if e == nil || called {
				t.Fatalf("aliases not rejected before write: error=%v called=%v", e, called)
			}
			if _, e := os.Lstat(filepath.Join(parent, "demo")); !os.IsNotExist(e) {
				t.Fatal("destination claimed", e)
			}
		})
	}
}
func TestWriterAllowsExactCaseSharedAncestors(t *testing.T) {
	parent := t.TempDir()
	files := []File{{Path: "A/shared/one", Data: []byte("one")}, {Path: "A/shared/two", Data: []byte("two")}}
	if e := Write(parent, "demo", files); e != nil {
		t.Fatal(e)
	}
	for _, f := range files {
		data, e := os.ReadFile(filepath.Join(parent, "demo", filepath.FromSlash(f.Path)))
		if e != nil || string(data) != string(f.Data) {
			t.Fatalf("%s: %q %v", f.Path, data, e)
		}
	}
}
