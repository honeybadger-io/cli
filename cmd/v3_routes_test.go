package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newV3Routes is a fake v3 Data API for commands that make more than one
// request: it answers each "METHOD /path?query" (or, failing that,
// "METHOD /path") with its own response and anything else with a 404,
// recording every request like newV3Server.
func newV3Routes(t *testing.T, routes map[string]v3Response) *v3Server {
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

		answer, found := routes[r.Method+" "+r.URL.RequestURI()]
		if !found {
			answer, found = routes[r.Method+" "+r.URL.Path]
		}
		if !found {
			answer = status(http.StatusNotFound,
				`{"error":{"code":"not_found","message":"Resource not found"}}`)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(answer.status)
		_, _ = w.Write([]byte(answer.body))
	}))
	t.Cleanup(s.Close)
	return s
}
