package generate

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/template"
)

//go:embed all:templates
var templates embed.FS
var mappings = map[string]string{"main.go.tmpl": "cmd/http/main.go", "readiness.go.tmpl": "internal/application/readiness.go", "http.go.tmpl": "internal/adapters/inbound/http/http.go", "http_test.go.tmpl": "internal/adapters/inbound/http/http_test.go", "config.go.tmpl": "internal/config/http.go", "go.mod.tmpl": "go.mod", "gitignore.tmpl": ".gitignore", "env.tmpl": ".env.example", "README.md.tmpl": "README.md", "domain.md.tmpl": "internal/domain/README.md", "ports.md.tmpl": "internal/application/ports/README.md"}

func Render(o Options) ([]File, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	if len(o.Entrypoints) != 1 || o.Entrypoints[0] != "http" || o.HTTP != "stdlib" || o.Database != "" {
		return nil, fmt.Errorf("this milestone supports only http with stdlib and no database")
	}
	files := make([]File, 0, len(mappings))
	for src, dst := range mappings {
		raw, err := templates.ReadFile("templates/" + src)
		if err != nil {
			return nil, err
		}
		t, err := template.New(src).Option("missingkey=error").Parse(string(raw))
		if err != nil {
			return nil, err
		}
		var b bytes.Buffer
		if err = t.Execute(&b, o); err != nil {
			return nil, err
		}
		data := b.Bytes()
		if strings.HasSuffix(dst, ".go") {
			data, err = format.Source(data)
			if err != nil {
				return nil, fmt.Errorf("format %s: %w", dst, err)
			}
		}
		files = append(files, File{dst, data})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
