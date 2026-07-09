package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"savetherecipe/internal/models"
)

type Server struct {
	handler InputHandler
}

type InputHandler interface {
	Handle(ctx context.Context, text string) (*models.Result, error)
}

func New(handler InputHandler) *Server {
	return &Server{handler: handler}
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /recipe", s.recipe)

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("HTTP server listening on :%s", port)
	return http.ListenAndServe(":"+port, mux)
}

func (s *Server) recipe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := s.handler.Handle(r.Context(), req.URL)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if result == nil {
		jsonError(w, "unsupported URL", http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"caption":    result.Caption,
		"source_url": result.SourceURL,
	})
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
