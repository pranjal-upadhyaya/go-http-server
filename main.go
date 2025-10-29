package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

func main() {
	mux := http.NewServeMux()

	fileServerHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))

	apiCfg := apiConfig{
		fileServerHits: atomic.Int32{},
	}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fileServerHandler))

	mux.Handle("GET /healthz", HealthCheck{statusCode: 200, body: "OK"})

	mux.Handle("GET /metrics", apiCfg.loadServerHits())

	mux.Handle("POST /reset", apiCfg.resetServerHits())

	server := http.Server{}

	server.Handler = mux
	server.Addr = ":8080"

	server.ListenAndServe()
}

type HealthCheck struct {
	statusCode int
	body       string
}

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = cfg.fileServerHits.Add(1)

		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func (cfg *apiConfig) loadServerHits() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		hits := cfg.fileServerHits.Load()

		response := fmt.Sprintf("Hits: %d", hits)

		body := []byte(response)

		w.Write(body)

		cfg.ServeHTTP(w, r)

	})
}

func (cfg *apiConfig) resetServerHits() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)

		_ = cfg.fileServerHits.Swap(0)

		// body := []byte{}

		// w.Write(body)

		cfg.ServeHTTP(w, r)

	})
}

func (hc HealthCheck) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")

	rw.WriteHeader(hc.statusCode)

	body := []byte(hc.body)

	rw.Write(body)
}
