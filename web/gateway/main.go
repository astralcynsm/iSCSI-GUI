package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func main() {
	listen := getenv("GATEWAY_LISTEN", "0.0.0.0:8080")
	agentSocket := getenv("AGENT_SOCKET", "/run/iscsi-agent/agent.sock")

	proxy := newUnixProxy(agentSocket)

	mux := http.NewServeMux()
	mux.Handle("/api/", withCORS(proxy))
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/", index)

	log.Printf("web-gateway listening on %s, proxying /api to unix://%s", listen, agentSocket)
	if err := http.ListenAndServe(listen, mux); err != nil {
		log.Fatalf("gateway failed: %v", err)
	}
}

func newUnixProxy(socketPath string) *httputil.ReverseProxy {
	target, _ := url.Parse("http://unix")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "unix", socketPath)
		},
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `{"error":"agent unavailable"}`)
	}
	return proxy
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!doctype html>
<html>
  <head><meta charset="utf-8"><title>iSCSIGUI</title></head>
  <body>
    <h1>iSCSIGUI Scaffold</h1>
    <p>Gateway is running. Frontend placeholder is active.</p>
    <p>Try <code>/api/v1/system/health</code> after agent starts.</p>
  </body>
</html>`)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok","service":"iscsi-web-gateway"}`)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Defensive: keep only /api paths proxied through this CORS handler.
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
