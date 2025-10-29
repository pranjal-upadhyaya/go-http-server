package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
)

func main() {
	mux := http.NewServeMux()

	fileServerHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))

	apiCfg := apiConfig{
		fileServerHits: atomic.Int32{},
	}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fileServerHandler))

	mux.Handle("GET /api/healthz", HealthCheck{statusCode: 200, body: "OK"})

	mux.Handle("GET /admin/metrics", apiCfg.loadServerHits())

	mux.Handle("POST /admin/reset", apiCfg.resetServerHits())

	mux.Handle("POST /api/validate_chirp", validateChirp())

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

		w.Header().Set("Content-Type", "text/html")

		w.WriteHeader(200)

		hits := cfg.fileServerHits.Load()

		response := fmt.Sprintf(`<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>
			</html>`, hits)

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

type requestFormat struct {
	Body string `json:"body"`
}

type successfulResponseFormat struct {
	CleanedBody string `json:"cleaned_body"`
}

type errorResponseFormat struct {
	Error string `json:"error"`
}

func validateChirp() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parsedRequest := requestFormat{}

		requestBody := r.Body
		decoder := json.NewDecoder(requestBody)
		err := decoder.Decode(&parsedRequest)

		if err != nil {

			errMsg := fmt.Sprintf("Error decoding request: %s", err)

			respondWithError(w, 500, errMsg)

			return
		}

		if len(parsedRequest.Body) > 140 {

			errMsg := "Chirp is too long"

			respondWithError(w, 400, errMsg)

			return

		}

		parsedResponse := successfulResponseFormat{
			CleanedBody: cleanMsg(parsedRequest.Body),
		}

		respondWithJson(w, 200, parsedResponse)
	})
}

func respondWithError(w http.ResponseWriter, statusCode int, msg string) {
	parsedError := errorResponseFormat{
		Error: msg,
	}

	res, _ := json.Marshal(parsedError)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(res)
}

func respondWithJson(w http.ResponseWriter, statusCode int, payload interface{}) {
	res, err := json.Marshal(&payload)
	if err != nil {

		errMsg := fmt.Sprintf("Error marshaling response: %s", err)

		respondWithError(w, 500, errMsg)

		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(res)
}

func cleanMsg(msg string) string {

	replacementWord := "****"

	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}

	msgSlice := strings.Split(msg, " ")

	for index, word := range msgSlice {
		if lowerString := strings.ToLower(word); slices.Contains(profaneWords, lowerString) {
			msgSlice[index] = replacementWord
		}
	}

	cleanedMsg := strings.Join(msgSlice, " ")

	return cleanedMsg

}
