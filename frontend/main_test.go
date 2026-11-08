package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestProxyPreservesAPIRequests(t *testing.T) {
	for _, tc := range []struct {
		method string
		path   string
		body   string
		status int
	}{
		{"GET", "/members?view=all", "", http.StatusOK},
		{"POST", "/members", `{"name":"Charlie","email":"charlie@example.com"}`, http.StatusCreated},
		{"PUT", "/members/3", `{"name":"Updated","email":"updated@example.com"}`, http.StatusOK},
		{"DELETE", "/members/3", "", http.StatusNoContent},
		{"GET", "/members/999", "", http.StatusNotFound},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			var wantHost string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tc.method || r.URL.RequestURI() != "/members-api-http"+tc.path {
					t.Errorf("unexpected upstream request: %s %s", r.Method, r.URL.RequestURI())
				}
				if r.Host != wantHost {
					t.Errorf("upstream Host = %q, want %q", r.Host, wantHost)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != tc.body {
					t.Errorf("upstream body = %q, err = %v", body, err)
				}
				if tc.body != "" && r.Header.Get("Content-Type") != "application/json" {
					t.Error("proxy lost Content-Type")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				if tc.status != http.StatusNoContent {
					_, _ = w.Write([]byte(`{"result":"forwarded"}`))
				}
			}))
			defer upstream.Close()
			target, _ := url.Parse(upstream.URL + "/members-api-http")
			wantHost = target.Host
			r := httptest.NewRequest(tc.method, "http://frontend.example/api"+tc.path, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			newHandler(target).ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, tc.status, w.Body.String())
			}
			if tc.status == http.StatusNoContent && w.Body.Len() != 0 {
				t.Fatal("DELETE must preserve the empty 204 response")
			}
			if tc.status != http.StatusNoContent && w.Body.String() != `{"result":"forwarded"}` {
				t.Fatalf("proxy changed the response body: %s", w.Body.String())
			}
		})
	}
}

func TestStaticFilesAndHealthDoNotRequireBackend(t *testing.T) {
	target, _ := url.Parse("http://127.0.0.1:1")
	handler := newHandler(target)
	for path, contentType := range map[string]string{
		"/":           "text/html",
		"/app.js":     "javascript",
		"/styles.css": "text/css",
		"/healthz":    "application/json",
	} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Header().Get("Content-Type"), contentType) {
			t.Errorf("GET %s: status=%d, content-type=%s", path, w.Code, w.Header().Get("Content-Type"))
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/does-not-exist", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing asset: status=%d", w.Code)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", "/", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("static files must reject POST: status=%d", w.Code)
	}
}

func TestUnavailableBackendReturnsUsefulError(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	target, _ := url.Parse(upstream.URL)
	upstream.Close()
	w := httptest.NewRecorder()
	newHandler(target).ServeHTTP(w, httptest.NewRequest("GET", "/api/members", nil))
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "Cannot reach the members API") {
		t.Fatalf("expected a useful 502 response, got %d: %s", w.Code, w.Body.String())
	}
}
