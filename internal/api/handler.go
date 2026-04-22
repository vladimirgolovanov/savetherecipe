package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"savetherecipe/internal/instagram"
)

type Handler struct {
	ig *instagram.Client
}

func NewHandler(ig *instagram.Client) *Handler {
	return &Handler{ig: ig}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /recipe", h.recipe)
}

func (h *Handler) recipe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	canonical, ok := instagram.NormalizeURL(req.URL)
	if !ok {
		jsonError(w, "not a valid Instagram post or reel URL", http.StatusBadRequest)
		return
	}

	post, err := h.ig.Fetch(canonical)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	text := instagram.CleanCaption(post.Caption)
	source := fmt.Sprintf("---\nИсточник: %s", canonical)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"text": text + "\n" + source})
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
