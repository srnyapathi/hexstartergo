package generate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func validateFiles(files []File) error {
	seen := map[string]bool{}
	ancestors := map[string]string{}
	for _, f := range files {
		p := f.Path
		if p == "" || path.IsAbs(p) || strings.ContainsAny(p, "\\:") || path.Clean(p) != p {
			return fmt.Errorf("unsafe output path %q", p)
		}
		for _, c := range p {
			if c < 32 {
				return fmt.Errorf("unsafe output path %q", p)
			}
		}
		for _, s := range strings.Split(p, "/") {
			if s == "." || s == ".." || s == "" || device(s) || strings.TrimRight(s, ". ") != s || strings.ContainsAny(s, "<>\"|?*\x00") {
				return fmt.Errorf("unsafe output path %q", p)
			}
		}
		key := strings.ToLower(p)
		if seen[key] {
			return fmt.Errorf("duplicate output path %q", p)
		}
		seen[key] = true
		for d := path.Dir(p); d != "."; d = path.Dir(d) {
			key := strings.ToLower(d)
			if previous, ok := ancestors[key]; ok && previous != d {
				return fmt.Errorf("case-conflicting output directories %q and %q", previous, d)
			}
			ancestors[key] = d
		}
	}
	for p := range seen {
		for d := path.Dir(p); d != "."; d = path.Dir(d) {
			if seen[d] {
				return fmt.Errorf("file-directory conflict %q", d)
			}
		}
	}
	return nil
}

type owned struct {
	path string
	info os.FileInfo
}

func Write(parent, name string, files []File) error {
	return write(parent, name, files, func(w io.Writer, b []byte) error {
		n, e := w.Write(b)
		if e == nil && n != len(b) {
			return io.ErrShortWrite
		}
		return e
	})
}
func write(parent, name string, files []File, writeData func(io.Writer, []byte) error) error {
	return writeWithOpenRoot(parent, name, files, writeData, func(root *os.Root, name string) (*os.Root, error) { return root.OpenRoot(name) })
}
func writeWithOpenRoot(parent, name string, files []File, writeData func(io.Writer, []byte) error, openRoot func(*os.Root, string) (*os.Root, error)) (err error) {
	if err = validName(name); err != nil {
		return err
	}
	if err = validateFiles(files); err != nil {
		return err
	}
	pr, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer pr.Close()
	if err = pr.Mkdir(name, 0755); err != nil {
		return fmt.Errorf("claim destination: %w", err)
	}
	di, err := pr.Lstat(name)
	if err != nil {
		return err
	}
	var r *os.Root
	var created []owned
	defer func() {
		if r != nil {
			err = errors.Join(err, r.Close())
		}
	}()
	defer func() {
		if err == nil {
			return
		}
		for i := len(created) - 1; i >= 0; i-- {
			item := created[i]
			current, e := r.Lstat(item.path)
			if os.IsNotExist(e) {
				continue
			}
			if e != nil {
				err = errors.Join(err, e)
				continue
			}
			if !os.SameFile(item.info, current) {
				err = errors.Join(err, fmt.Errorf("cleanup preserved replaced path %q", item.path))
				continue
			}
			if e = r.Remove(item.path); e != nil {
				err = errors.Join(err, fmt.Errorf("cleanup %s: %w", item.path, e))
			}
		}
		current, e := pr.Lstat(name)
		if e == nil && os.SameFile(di, current) {
			if e = pr.Remove(name); e != nil {
				err = errors.Join(err, fmt.Errorf("cleanup destination: %w", e))
			}
		} else if e == nil {
			err = errors.Join(err, fmt.Errorf("cleanup preserved replaced destination %q", name))
		} else if !os.IsNotExist(e) {
			err = errors.Join(err, e)
		}
	}()
	r, err = openRoot(pr, name)
	if err != nil {
		return err
	}

	dirs := map[string]bool{}
	for _, f := range files {
		parts := strings.Split(f.Path, "/")
		for i := 1; i < len(parts); i++ {
			d := strings.Join(parts[:i], "/")
			if dirs[d] {
				continue
			}
			if err = r.Mkdir(d, 0755); err != nil {
				return err
			}
			info, e := r.Lstat(d)
			if e != nil {
				return e
			}
			created = append(created, owned{d, info})
			dirs[d] = true
		}
		out, e := r.OpenFile(f.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if e != nil {
			return e
		}
		info, e := out.Stat()
		if e != nil {
			out.Close()
			return e
		}
		created = append(created, owned{f.Path, info})
		e = writeData(out, f.Data)
		e = errors.Join(e, out.Close())
		if e != nil {
			return e
		}
	}
	return nil
}
func Generate(parent string, o Options) (string, error) {
	files, err := Render(o)
	if err != nil {
		return "", err
	}
	if err = Write(parent, o.Name, files); err != nil {
		return "", err
	}
	return filepath.Join(parent, o.Name), nil
}
