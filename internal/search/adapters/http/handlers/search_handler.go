package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/search/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/search/domain/model"
)

type SearchHandler struct {
	service *service.SearchService
	logger  logger.Logger
}

func NewSearchHandler(service *service.SearchService, logger logger.Logger) *SearchHandler {
	return &SearchHandler{
		service: service,
		logger:  logger,
	}
}

func (h *SearchHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/search", h.Search).Methods("POST")
	router.HandleFunc("/search/index", h.Index).Methods("POST")
	router.HandleFunc("/search/suggest", h.Suggest).Methods("GET")
	router.HandleFunc("/search/stats", h.GetStats).Methods("GET")
}

func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var query model.SearchQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.Search(ctx, query)
	if err != nil {
		h.logger.Error("Search failed", "error", err)
		h.respondError(w, http.StatusInternalServerError, "search failed")
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

func (h *SearchHandler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var doc model.SearchDocument
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.Index(ctx, &doc); err != nil {
		h.logger.Error("Failed to index document", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to index document")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "indexed"})
}

func (h *SearchHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	prefix := r.URL.Query().Get("q")
	index := r.URL.Query().Get("index")
	
	if prefix == "" {
		h.respondError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	suggestions, err := h.service.Suggest(ctx, prefix, model.SearchIndex(index))
	if err != nil {
		h.logger.Error("Failed to get suggestions", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to get suggestions")
		return
	}

	h.respondJSON(w, http.StatusOK, suggestions)
}

func (h *SearchHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.service.GetStats(ctx)
	if err != nil {
		h.logger.Error("Failed to get stats", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	h.respondJSON(w, http.StatusOK, stats)
}

func (h *SearchHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *SearchHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
