package generate_test

import (
	"bytes"
	"context"
	"hexgen/internal/cli"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestGeneratedHTTPProject(t *testing.T) {
	parent := t.TempDir()
	var out, stderr bytes.Buffer
	if code := cli.Run([]string{"new", "demo", "--module", "example.com/demo", "--entrypoints", "http", "--http", "stdlib"}, &out, &stderr, parent); code != 0 {
		t.Fatalf("CLI code %d: %s", code, stderr.String())
	}
	d := filepath.Join(parent, "demo")
	for _, args := range [][]string{{"test", "./..."}, {"build", "-o", "server.exe", "./cmd/http"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir = d
		var output cappedOutput
		cmd.Stdout = &output
		cmd.Stderr = &output
		e := cmd.Run()
		cancel()
		if e != nil {
			t.Fatalf("go %v: %v\n%s", args, e, output.String())
		}
	}
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	addr := ln.Addr().String()
	ln.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	exe := filepath.Join(d, "server.exe")
	if runtime.GOOS != "windows" {
		os.Chmod(exe, 0755)
	}
	cmd := exec.CommandContext(ctx, exe)
	cmd.Dir = d
	cmd.Env = append(os.Environ(), "HTTP_ADDR="+addr)
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { cancel(); cmd.Wait() }()
	client := http.Client{Timeout: time.Second}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		res, e := client.Get("http://" + addr + "/readyz")
		if e == nil {
			b, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
			res.Body.Close()
			if res.StatusCode != 200 || string(b) != "ready\n" {
				t.Fatalf("readiness %d %q", res.StatusCode, b)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("server never became ready")
}

// cappedOutput drains subprocess output without retaining unbounded diagnostics.
type cappedOutput struct{ bytes.Buffer }

func (w *cappedOutput) Write(p []byte) (int, error) {
	n := len(p)
	if remaining := 64*1024 - w.Len(); remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = w.Buffer.Write(p)
	}
	return n, nil
}
