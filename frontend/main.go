package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

//go:embed static/*
var assets embed.FS

func newHandler(target *url.URL) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.SetXForwarded()
		},
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 10 * time.Second
	proxy.Transport = transport
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("backend request failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Cannot reach the members API. Check the backend and API_BASE_URL.",
		})
	}
	static, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	// Keep browser requests on the frontend origin; strip only our /api prefix.
	// The target may include a gateway prefix when using an external API URL.
	mux.Handle("/api/", http.StripPrefix("/api", proxy))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"status\":\"ok\",\"service\":\"members-demo-frontend\"}\n"))
	})
	files := http.FileServer(http.FS(static))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		files.ServeHTTP(w, r)
	})
	return mux
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	apiBase := strings.TrimRight(os.Getenv("API_BASE_URL"), "/")
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	target, err := url.Parse(apiBase)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") || target.RawQuery != "" || target.Fragment != "" {
		log.Fatal("API_BASE_URL must be an http(s) URL without a query or fragment")
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           newHandler(target),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	fmt.Printf("Members frontend listening on http://localhost:%s\n", port)
	log.Fatal(server.ListenAndServe())
}
