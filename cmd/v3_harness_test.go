package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// v3Request is one request the fake v3 server received.
type v3Request struct {
	Method string
	Path   string
	Query  string
	Auth   string
	Body   string
}

// v3Server is a fake v3 Data API that records every request and answers each
// with the same response.
type v3Server struct {
	*httptest.Server
	mu       sync.Mutex
	requests []v3Request
}

type v3Response struct {
	status int
	body   string
}

// newV3Server answers every request with resp, or with an empty object when
// none is given (for tests that expect no request at all).
func newV3Server(t *testing.T, resp ...v3Response) *v3Server {
	t.Helper()
	s := &v3Server{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.requests = append(s.requests, v3Request{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.RawQuery,
			Auth:   r.Header.Get("Authorization"),
			Body:   string(body),
		})
		s.mu.Unlock()

		answer := v3Response{status: http.StatusOK, body: `{"data":{}}`}
		if len(resp) > 0 {
			answer = resp[0]
		}
		if answer.status == http.StatusNoContent {
			w.WriteHeader(answer.status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(answer.status)
		_, _ = w.Write([]byte(answer.body))
	}))
	t.Cleanup(s.Close)
	return s
}

func ok(body string) v3Response { return v3Response{status: http.StatusOK, body: body} }

func status(code int, body string) v3Response { return v3Response{status: code, body: body} }

// Requests returns what the server has received so far.
func (s *v3Server) Requests() []v3Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]v3Request(nil), s.requests...)
}

// only returns the single request the server received, failing otherwise.
func (s *v3Server) only(t *testing.T) v3Request {
	t.Helper()
	reqs := s.Requests()
	if len(reqs) != 1 {
		t.Fatalf("server got %d requests, want 1: %+v", len(reqs), reqs)
	}
	return reqs[0]
}

// cliEnv is the environment a test command runs in. Every HONEYBADGER_ variable
// the CLI reads is cleared first, so the developer's own shell can't leak in.
type cliEnv map[string]string

var honeybadgerEnvVars = []string{
	"HONEYBADGER_API_TOKEN",
	"HONEYBADGER_AUTH_TOKEN",
	"HONEYBADGER_API_KEY",
	"HONEYBADGER_PROJECT_ID",
	"HONEYBADGER_ENDPOINT",
}

// runCLI runs hb with args through the root command, as a user would, and
// returns what it printed to stdout.
func runCLI(t *testing.T, env cliEnv, args ...string) (string, error) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	for _, k := range honeybadgerEnvVars {
		t.Setenv(k, "")
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
	viper.Reset()
	resetFlags(rootCmd)

	stdout := captureStdout(t)
	var stderr bytes.Buffer
	rootCmd.SetOut(&stderr)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	return stdout(), err
}

// resetFlags puts every flag in the tree back to its default, since cobra
// keeps flag values in package variables between Execute calls.
func resetFlags(cmd *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if sv, isSlice := f.Value.(pflag.SliceValue); isSlice {
			_ = sv.Replace(nil)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	}
	cmd.Flags().VisitAll(reset)
	cmd.PersistentFlags().VisitAll(reset)
	for _, c := range cmd.Commands() {
		resetFlags(c)
	}
}

// captureStdout redirects os.Stdout until the returned function is called,
// which restores it and returns what was written.
func captureStdout(t *testing.T) func() string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	restored := false
	restore := func() string {
		if restored {
			return ""
		}
		restored = true
		_ = w.Close()
		os.Stdout = orig
		return <-done
	}
	t.Cleanup(func() { restore() })
	return restore
}

// assertJSONBody fails unless body is exactly the JSON want, key for key.
func assertJSONBody(t *testing.T, want, body string) {
	t.Helper()
	var got, exp any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("request body %q is not JSON: %v", body, err)
	}
	if err := json.Unmarshal([]byte(want), &exp); err != nil {
		t.Fatalf("want %q is not JSON: %v", want, err)
	}
	gotJSON, _ := json.Marshal(got)
	expJSON, _ := json.Marshal(exp)
	if string(gotJSON) != string(expJSON) {
		t.Errorf("request body = %s, want %s", gotJSON, expJSON)
	}
}

// v3Env is the environment for a command that should reach the fake server
// with a valid API Token.
func v3Env(s *v3Server) cliEnv {
	return cliEnv{"HONEYBADGER_API_TOKEN": "hbu_test", "HONEYBADGER_ENDPOINT": s.URL}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
