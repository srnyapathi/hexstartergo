package generate

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func good() Options {
	return Options{Name: "demo", Module: "example.com/demo", Entrypoints: []string{"http"}, HTTP: "stdlib"}
}
func TestValidation(t *testing.T) {
	if err := good().Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Options){func(o *Options) { o.Name = "CON" }, func(o *Options) { o.Name = "../bad" }, func(o *Options) { o.Module = "bad module" }, func(o *Options) { o.Entrypoints = nil }, func(o *Options) { o.Entrypoints = []string{"http", "http"} }, func(o *Options) { o.HTTP = "" }, func(o *Options) { o.Broker = "rabbitmq" }, func(o *Options) { o.Database = "sqlite" }} {
		o := good()
		mutate(&o)
		if o.Validate() == nil {
			t.Fatalf("accepted invalid %+v", o)
		}
	}
}
func TestRender(t *testing.T) {
	a, e := Render(good())
	if e != nil {
		t.Fatal(e)
	}
	b, e := Render(good())
	if e != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("nondeterministic")
	}
	want := map[string]bool{"cmd/http/main.go": false, "internal/application/readiness.go": false, "internal/adapters/inbound/http/http.go": false, "internal/config/http.go": false, "go.mod": false, ".gitignore": false, ".env.example": false, "README.md": false}
	for _, f := range a {
		if _, ok := want[f.Path]; ok {
			want[f.Path] = true
		}
	}
	for p, ok := range want {
		if !ok {
			t.Error("missing", p)
		}
	}
	o := good()
	o.HTTP = "chi"
	if _, e = Render(o); e == nil {
		t.Fatal("unsupported render accepted")
	}
}
func TestWriterPrevalidatesAndRefusesExisting(t *testing.T) {
	for _, p := range []string{"../escape", "/root", "a\\b", "CON", "a/../b", "a.", "C:/foo"} {
		d := t.TempDir()
		if Write(d, "demo", []File{{Path: p}}) == nil {
			t.Error("accepted", p)
		}
		if _, e := os.Stat(filepath.Join(d, "demo")); !os.IsNotExist(e) {
			t.Error("mutated destination", p)
		}
	}
	d := t.TempDir()
	os.Mkdir(filepath.Join(d, "demo"), 0755)
	if Write(d, "demo", nil) == nil {
		t.Fatal("accepted existing directory")
	}
	for _, files := range [][]File{{{Path: "a"}, {Path: "a"}}, {{Path: "a"}, {Path: "a/b"}}} {
		if Write(t.TempDir(), "demo", files) == nil {
			t.Fatal("accepted conflict")
		}
	}
}
func TestWriterConcurrent(t *testing.T) {
	d := t.TempDir()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); results <- Write(d, "demo", []File{{Path: "a/b", Data: []byte("ok")}}) }()
	}
	wg.Wait()
	close(results)
	wins := 0
	for e := range results {
		if e == nil {
			wins++
		}
	}
	if wins != 1 {
		t.Fatal(wins)
	}
}
func TestWriterFailureCleanup(t *testing.T) {
	d := t.TempDir()
	boom := errors.New("write failed")
	err := write(d, "demo", []File{{Path: "a/b", Data: []byte("hello")}}, func(w io.Writer, b []byte) error { w.Write(b[:1]); return boom })
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if _, e := os.Stat(filepath.Join(d, "demo")); !os.IsNotExist(e) {
		t.Fatal("owned destination remains", e)
	}
}
func TestWriterPreservesForeignContent(t *testing.T) {
	d := t.TempDir()
	err := write(d, "demo", []File{{Path: "a/b"}}, func(w io.Writer, b []byte) error {
		os.WriteFile(filepath.Join(d, "demo", "foreign"), []byte("keep"), 0600)
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("missing error")
	}
	b, e := os.ReadFile(filepath.Join(d, "demo", "foreign"))
	if e != nil || string(b) != "keep" {
		t.Fatal("foreign content removed", e)
	}
}
func TestWriterRejectsPortableInvalidPaths(t *testing.T) {
	for _, p := range []string{"COM¹/file", "a\x01b", "aux.txt", "a /b"} {
		if e := Write(t.TempDir(), "demo", []File{{Path: p}}); e == nil {
			t.Errorf("accepted %q", p)
		}
	}
}
func TestWriterPreservesReplacement(t *testing.T) {
	d := t.TempDir()
	err := write(d, "demo", []File{{Path: "one"}}, func(w io.Writer, b []byte) error {
		if err := os.Rename(filepath.Join(d, "demo", "one"), filepath.Join(d, "demo", "moved")); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(d, "demo", "one"), []byte("replacement"), 0600); err != nil {
			return err
		}
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	b, e := os.ReadFile(filepath.Join(d, "demo", "one"))
	if e != nil || string(b) != "replacement" {
		t.Fatal("replacement removed", e)
	}
}
func TestWriterPrevalidatesBeforeAnyWrite(t *testing.T) {
	for _, p := range []string{"COM¹/file", "a\x01b"} {
		called := false
		err := write(t.TempDir(), "demo", []File{{Path: "first"}, {Path: p}}, func(io.Writer, []byte) error { called = true; return nil })
		if err == nil || called {
			t.Errorf("did not prevalidate %q", p)
		}
	}
}
func TestApplicationOwnsPortsLocation(t *testing.T) {
	files, e := Render(good())
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, f := range files {
		if f.Path == "internal/ports/README.md" {
			t.Fatal("ports outside application")
		}
		if f.Path == "internal/application/ports/README.md" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing application-owned ports")
	}
}
func TestRootOpenFailureCleanup(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		d := t.TempDir()
		boom := errors.New("cannot open root")
		e := writeWithOpenRoot(d, "demo", nil, func(io.Writer, []byte) error { return nil }, func(parent *os.Root, name string) (*os.Root, error) {
			if foreign {
				if err := os.WriteFile(filepath.Join(d, name, "foreign"), []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			return nil, boom
		})
		if !errors.Is(e, boom) {
			t.Fatal(e)
		}
		if foreign {
			b, err := os.ReadFile(filepath.Join(d, "demo", "foreign"))
			if err != nil || string(b) != "keep" {
				t.Fatal("foreign content lost", err)
			}
			if !strings.Contains(e.Error(), "cleanup destination") {
				t.Fatal("cleanup failure unreported", e)
			}
		} else if _, err := os.Stat(filepath.Join(d, "demo")); !os.IsNotExist(err) {
			t.Fatal("owned empty destination remains", err)
		}
	}
}
