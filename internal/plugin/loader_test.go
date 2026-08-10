package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveCommand_CommandIsSplit(t *testing.T) {
	got, err := resolveCommand(PluginSpec{Name: "x", Command: "/bin/echo hi there"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "/bin/echo" || got[1] != "hi" || got[2] != "there" {
		t.Errorf("unexpected argv: %v", got)
	}
}

func TestResolveCommand_BinaryFromPathMock(t *testing.T) {
	old := loadLookPath
	loadLookPath = func(name string) (string, error) {
		if name == "snagify-plugin-fake" {
			return "/tmp/snagify-plugin-fake", nil
		}
		return "", errors.New("not on PATH")
	}
	defer func() { loadLookPath = old }()

	got, err := resolveCommand(PluginSpec{Name: "x", Binary: "snagify-plugin-fake", Args: []string{"--flag"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/tmp/snagify-plugin-fake", "--flag"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("argv = %v, want %v", got, want)
	}
}

func TestResolveCommand_RelativeBinarySkipsLookPath(t *testing.T) {
	old := loadLookPath
	loadLookPath = func(string) (string, error) { return "", errors.New("unused") }
	defer func() { loadLookPath = old }()

	got, err := resolveCommand(PluginSpec{Name: "x", Binary: "./bin/my-plugin"})
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != "./bin/my-plugin" {
		t.Errorf("argv = %v, want [./bin/my-plugin]", got)
	}
}

func TestResolveCommand_Errors(t *testing.T) {
	cases := []PluginSpec{
		{Name: "x"},                              // neither binary nor command
		{Name: "x", Command: "   "},              // whitespace command
	}
	for _, c := range cases {
		if _, err := resolveCommand(c); err == nil {
			t.Errorf("expected error for %+v", c)
		}
	}
}

// fakePlugin writes a static JSON response to stdout and a chosen exit
// code. It writes the OK response only when exitCode == 0; non-zero exits
// are used to test the host's error path.
func fakePlugin(t *testing.T, dir, name string, body string, exitCode int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\n"+
		"cat <<'JSON'\n"+body+"\nJSON\n"+
		"exit "+itoa(exitCode)+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func TestRun_HappyPath(t *testing.T) {
	dir := t.TempDir()
	resp := Response{Protocol: Protocol, Results: []Result{
		{Name: "a", Status: "ok"},
		{Name: "b", Status: "warning", Message: "lag high"},
	}}
	body, _ := json.Marshal(resp)
	plugin := fakePlugin(t, dir, "plugin-happy", string(body), 0)

	rr := Run(
		PluginSpec{Name: "x", Command: plugin},
		[]Spec{{Name: "a", Run: []string{"true"}}, {Name: "b", Run: []string{"true"}}},
		"",
	)
	if rr.Error != "" {
		t.Fatalf("unexpected error: %s", rr.Error)
	}
	if len(rr.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(rr.Results))
	}
	if rr.Results[0].Status != "ok" || rr.Results[1].Status != "warning" {
		t.Errorf("unexpected results: %+v", rr.Results)
	}
}

func TestRun_NonZeroExit(t *testing.T) {
	dir := t.TempDir()
	plugin := fakePlugin(t, dir, "plugin-bad", `{"protocol":"snagify/plugin/v1","results":[]}`, 2)

	rr := Run(PluginSpec{Name: "x", Command: plugin}, nil, "")
	if rr.Error == "" {
		t.Fatalf("expected error for non-zero exit")
	}
	if !strings.Contains(rr.Error, "exit") {
		t.Errorf("error should mention exit; got %q", rr.Error)
	}
}

func TestRun_BadJSON(t *testing.T) {
	dir := t.TempDir()
	plugin := fakePlugin(t, dir, "plugin-bad-json", "this is not json", 0)
	rr := Run(PluginSpec{Name: "x", Command: plugin}, nil, "")
	if rr.Error == "" {
		t.Fatal("expected error for bad JSON")
	}
	if !strings.Contains(rr.Error, "bad JSON") {
		t.Errorf("error should mention bad JSON; got %q", rr.Error)
	}
}

func TestRun_ProtocolMismatch(t *testing.T) {
	dir := t.TempDir()
	body := `{"protocol":"snagify/plugin/v999","results":[]}`
	plugin := fakePlugin(t, dir, "plugin-mismatch", body, 0)
	rr := Run(PluginSpec{Name: "x", Command: plugin}, nil, "")
	if rr.Error == "" {
		t.Fatal("expected protocol mismatch error")
	}
	if !strings.Contains(rr.Error, "protocol") {
		t.Errorf("expected protocol error; got %q", rr.Error)
	}
}

func TestRun_PluginReportedError(t *testing.T) {
	dir := t.TempDir()
	body := `{"protocol":"snagify/plugin/v1","results":[],"error":"boom"}`
	plugin := fakePlugin(t, dir, "plugin-self-err", body, 0)
	rr := Run(PluginSpec{Name: "x", Command: plugin}, nil, "")
	if rr.Error == "" {
		t.Fatal("expected plugin error to surface")
	}
	if !strings.Contains(rr.Error, "boom") {
		t.Errorf("expected 'boom' in error; got %q", rr.Error)
	}
}

func TestRun_PluginTimesOut(t *testing.T) {
	dir := t.TempDir()
	// Sleeps forever; test passes if our timeout kills it.
	if err := os.WriteFile(filepath.Join(dir, "plugin-slow"),
		[]byte("#!/usr/bin/env bash\nsleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	rr := Run(PluginSpec{Name: "slow", Command: filepath.Join(dir, "plugin-slow"), Timeout: "100ms"}, nil, "")
	if rr.Error == "" {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(rr.Error, "timed out") {
		t.Errorf("expected 'timed out' in error; got %q", rr.Error)
	}
}

func TestDecodeRequest_HappyPath(t *testing.T) {
	body := `{"protocol":"snagify/plugin/v1","kind":"checks","checks":[{"name":"a","run":["true"]}]}`
	req, err := DecodeRequest(strings.NewReader(body))
	if err != nil {
		t.Fatalf("DecodeRequest: %v", err)
	}
	if len(req.Checks) != 1 || req.Checks[0].Name != "a" {
		t.Errorf("unexpected: %+v", req)
	}
}

func TestDecodeRequest_ProtocolMismatch(t *testing.T) {
	body := `{"protocol":"snagify/plugin/v0","kind":"checks","checks":[]}`
	if _, err := DecodeRequest(strings.NewReader(body)); err == nil {
		t.Fatal("expected protocol mismatch")
	}
}

func TestEncodeResponse_RoundTrip(t *testing.T) {
	r, _ := EncodeResponse([]Result{{Name: "x", Status: "ok"}})
	var got Response
	if err := json.Unmarshal(bytes.TrimSpace(r), &got); err != nil {
		t.Fatal(err)
	}
	if got.Protocol != Protocol {
		t.Errorf("protocol = %q, want %q", got.Protocol, Protocol)
	}
}

func TestEncodeResponse_EmptyResults(t *testing.T) {
	// Plugins that no-op (e.g. a check that's always trivially true) emit
	// an empty Results slice — the loader must still accept that.
	r, err := EncodeResponse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(r, []byte(`"results":`)) {
		t.Errorf("expected results field; got %s", r)
	}
}

// guard io import usage.
var _ = io.Discard
