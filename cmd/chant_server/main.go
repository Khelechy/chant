package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/khelechy/chant"
	"github.com/khelechy/chant/internal/wavio"
)

const maxUploadBytes = 32 << 20

type decodeResponse struct {
	EncryptedMessageBase64 string `json:"encryptedMessageBase64"`
	Filename               string `json:"filename,omitempty"`
	SampleRate             int    `json:"sampleRate"`
	SampleCount            int    `json:"sampleCount"`
	EncryptedBytes         int    `json:"encryptedBytes"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/v1/decode", handleDecode)

	server := &http.Server{
		Addr:              *addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("chant_server listening on %s", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleDecode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("parse multipart form: %v", err))
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing multipart field 'audio'")
		return
	}
	defer file.Close()

	wavBytes, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("read uploaded audio: %v", err))
		return
	}

	samples, sampleRate, err := wavio.ReadWAVBytes(wavBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	blob, err := chant.ExtractEncryptedMessageWithSampleRate(samples, sampleRate)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, decodeResponse{
		EncryptedMessageBase64: base64.StdEncoding.EncodeToString(blob),
		Filename:               header.Filename,
		SampleRate:             sampleRate,
		SampleCount:            len(samples),
		EncryptedBytes:         len(blob),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
