//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/svetlyopet/mindpalace/internal/config"
)

func runMP(t *testing.T, vaultDir string, extraEnv []string, stdin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	allArgs := append([]string{"--vault", vaultDir}, args...)
	return runMPArgs(t, allArgs, extraEnv, stdin)
}

func runMPArgs(t *testing.T, args, extraEnv []string, stdin string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(mpBin, args...)
	cmd.Env = testEnv(extraEnv)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return outBuf.String(), errBuf.String(), code
}

// runMPPty runs mp with a terminal so password and confirm prompts can be answered.
// Stdout and stderr are combined.
func runMPPty(t *testing.T, vaultDir string, extraEnv []string, keys string, args ...string) (output string, exitCode int) {
	t.Helper()
	allArgs := append([]string{"--vault", vaultDir}, args...)
	cmd := exec.Command(mpBin, allArgs...)
	cmd.Env = testEnv(extraEnv)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, ptmx)
		close(done)
	}()
	if keys != "" {
		if _, err := io.WriteString(ptmx, keys); err != nil {
			_ = ptmx.Close()
			_ = cmd.Wait()
			<-done
			t.Fatal(err)
		}
	}
	err = cmd.Wait()
	_ = ptmx.Close()
	<-done
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return buf.String(), code
}

func testEnv(extra []string) []string {
	base := os.Environ()
	out := make([]string, 0, len(base)+len(extra))
	for _, e := range base {
		if strings.HasPrefix(e, "MINDPALACE_PASSWORD=") || strings.HasPrefix(e, "MINDPALACE_VAULT=") {
			continue
		}
		out = append(out, e)
	}
	return append(out, extra...)
}

func initVault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	out, errOut, code := runMP(t, dir, nil, "", "vault", "init", dir)
	if code != 0 {
		t.Fatalf("init code=%d out=%q err=%q", code, out, errOut)
	}
	return dir
}

func mustRun(t *testing.T, vaultDir string, args ...string) string {
	t.Helper()
	out, errOut, code := runMP(t, vaultDir, nil, "", args...)
	if code != 0 {
		t.Fatalf("%v code=%d\nout=%q\nerr=%q", args, code, out, errOut)
	}
	return out
}

func createdID(t *testing.T, out string) string {
	t.Helper()
	fields := strings.Fields(out)
	if len(fields) < 2 || fields[0] != "Created" {
		t.Fatalf("created line = %q", out)
	}
	return fields[1]
}

func decodeJSON[T any](t *testing.T, raw string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &v); err != nil {
		t.Fatalf("json: %v raw=%q", err, raw)
	}
	return v
}

type runningServer struct {
	URL   string
	Token string
	cmd   *exec.Cmd
}

func startServe(t *testing.T, vaultDir string) *runningServer {
	t.Helper()
	port := freePort(t)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	cmd := exec.Command(mpBin, "--vault", vaultDir, "serve", "--addr", addr)
	cmd.Env = testEnv(nil)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	var log bytes.Buffer
	var mu sync.Mutex
	ready := make(chan struct{}, 1)
	scanPipe := func(r io.Reader) {
		sc := newLineReader(r)
		for sc.scan() {
			line := sc.text()
			mu.Lock()
			log.WriteString(line)
			log.WriteByte('\n')
			mu.Unlock()
			if strings.Contains(line, "Mindpalace listening") {
				select {
				case ready <- struct{}{}:
				default:
				}
			}
		}
	}
	go scanPipe(stderr)
	go scanPipe(stdout)

	t.Cleanup(func() {
		stopProcess(cmd)
	})

	select {
	case <-ready:
	case <-time.After(20 * time.Second):
		mu.Lock()
		logged := log.String()
		mu.Unlock()
		t.Fatalf("serve did not listen on %s\n%s", addr, logged)
	}
	// The process prints the listen line before Accept, so wait until the port answers.
	waitForHTTP(t, "http://"+addr+"/api/session")

	cfg, err := config.Load(vaultDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Serve.Token == "" {
		t.Fatal("serve token was not persisted")
	}
	if cfg.Serve.Addr != addr {
		t.Fatalf("serve.addr = %q, want %q", cfg.Serve.Addr, addr)
	}
	return &runningServer{URL: "http://" + addr, Token: cfg.Serve.Token, cmd: cmd}
}

func waitForHTTP(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		last = err
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server not accepting %s: %v", url, last)
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func stopProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(syscall.SIGINT)
	wait := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(wait)
	}()
	select {
	case <-wait:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		<-wait
	}
}

type lineReader struct {
	r   io.Reader
	buf bytes.Buffer
	err error
	s   string
}

func newLineReader(r io.Reader) *lineReader {
	return &lineReader{r: r}
}

func (l *lineReader) scan() bool {
	if l.err != nil {
		return false
	}
	tmp := make([]byte, 256)
	for {
		if i := bytes.IndexByte(l.buf.Bytes(), '\n'); i >= 0 {
			l.s = l.buf.String()[:i]
			rest := append([]byte(nil), l.buf.Bytes()[i+1:]...)
			l.buf.Reset()
			_, _ = l.buf.Write(rest)
			return true
		}
		n, err := l.r.Read(tmp)
		if n > 0 {
			_, _ = l.buf.Write(tmp[:n])
		}
		if err != nil {
			l.err = err
			if l.buf.Len() > 0 {
				l.s = l.buf.String()
				l.buf.Reset()
				return true
			}
			return false
		}
	}
}

func (l *lineReader) text() string { return l.s }

func apiDo(t *testing.T, srv *runningServer, token, method, path string, body []byte) (int, []byte) {
	t.Helper()
	status, respBody, _ := apiDoHeaders(t, srv, token, method, path, body, nil)
	return status, respBody
}

func apiDoHeaders(t *testing.T, srv *runningServer, token, method, path string, body []byte, client *http.Client) (int, []byte, http.Header) {
	t.Helper()
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, srv.URL+path, r)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil && method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, b, resp.Header
}

func writeExecutable(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func editorStub(t *testing.T, marker string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mp-editor")
	script := fmt.Sprintf(`#!/bin/sh
file=$1
tmp=$(mktemp)
awk 'BEGIN { n = 0 } /^---[[:space:]]*$/ { n++; print; if (n == 2) { print ""; print "%s"; exit } next } { print }' "$file" > "$tmp"
cat "$tmp" > "$file"
rm -f "$tmp"
`, marker)
	writeExecutable(t, path, script)
	return path
}

func openStubDir(t *testing.T) (pathEnv, logPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "opened.txt")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$1\" >> %q\n", logPath)
	for _, name := range []string{"open", "xdg-open"} {
		writeExecutable(t, filepath.Join(dir, name), script)
	}
	pathEnv = dir + string(os.PathListSeparator) + "/bin" + string(os.PathListSeparator) + "/usr/bin"
	return pathEnv, logPath
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

const articleHTML = `<!DOCTYPE html><html><head><title>Local Article</title></head><body><article><h1>Local Article Headline</h1><p>zephyrquartz is the unique marker in this captured article about databases.</p></article></body></html>`
