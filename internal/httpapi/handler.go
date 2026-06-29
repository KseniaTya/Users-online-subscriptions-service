package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"task-ef-mobile/internal/subscriptions"

	"github.com/google/uuid"
)

type Handler struct {
	repo   *subscriptions.Repository
	logger *slog.Logger
}

func NewHandler(repo *subscriptions.Repository, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, logger: logger}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /docs/openapi.yaml", h.openapi)
	mux.HandleFunc("POST /subscriptions", h.createSubscription)
	mux.HandleFunc("GET /subscriptions", h.listSubscriptions)
	mux.HandleFunc("GET /subscriptions/total", h.totalSubscriptions)
	mux.HandleFunc("GET /subscriptions/{id}", h.getSubscription)
	mux.HandleFunc("PUT /subscriptions/{id}", h.updateSubscription)
	mux.HandleFunc("DELETE /subscriptions/{id}", h.deleteSubscription)
	return h.logMiddleware(mux)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) openapi(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "docs/openapi.yaml")
}

func (h *Handler) createSubscription(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeSubscriptionRequest(w, r)
	if !ok {
		return
	}

	sub, err := h.repo.Create(r.Context(), input)
	if err != nil {
		h.serverError(w, "create subscription", err)
		return
	}

	h.logger.Info("subscription created", slog.String("id", sub.ID.String()), slog.String("user_id", sub.UserID.String()))
	writeJSON(w, http.StatusCreated, toResponse(sub))
}

func (h *Handler) getSubscription(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathUUID(w, r)
	if !ok {
		return
	}

	sub, err := h.repo.Get(r.Context(), id)
	if err != nil {
		h.handleRepoError(w, "get subscription", err)
		return
	}

	writeJSON(w, http.StatusOK, toResponse(sub))
}

func (h *Handler) updateSubscription(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathUUID(w, r)
	if !ok {
		return
	}

	input, ok := decodeSubscriptionRequest(w, r)
	if !ok {
		return
	}

	sub, err := h.repo.Update(r.Context(), id, subscriptions.UpdateInput(input))
	if err != nil {
		h.handleRepoError(w, "update subscription", err)
		return
	}

	h.logger.Info("subscription updated", slog.String("id", sub.ID.String()), slog.String("user_id", sub.UserID.String()))
	writeJSON(w, http.StatusOK, toResponse(sub))
}

func (h *Handler) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathUUID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.handleRepoError(w, "delete subscription", err)
		return
	}

	h.logger.Info("subscription deleted", slog.String("id", id.String()))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := subscriptions.ListFilter{
		ServiceName: strings.TrimSpace(query.Get("service_name")),
		Limit:       parseInt(query.Get("limit"), 50),
		Offset:      parseInt(query.Get("offset"), 0),
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		writeError(w, http.StatusBadRequest, "limit must be between 1 and 200")
		return
	}
	if filter.Offset < 0 {
		writeError(w, http.StatusBadRequest, "offset must be greater than or equal to 0")
		return
	}

	if userIDRaw := strings.TrimSpace(query.Get("user_id")); userIDRaw != "" {
		userID, err := uuid.Parse(userIDRaw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "user_id must be a valid UUID")
			return
		}
		filter.UserID = &userID
	}

	items, err := h.repo.List(r.Context(), filter)
	if err != nil {
		h.serverError(w, "list subscriptions", err)
		return
	}

	response := make([]subscriptionResponse, 0, len(items))
	for _, item := range items {
		response = append(response, toResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) totalSubscriptions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	from, err := parseMonth(query.Get("from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "from must use MM-YYYY format")
		return
	}
	to, err := parseMonth(query.Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "to must use MM-YYYY format")
		return
	}
	if to.Before(from) {
		writeError(w, http.StatusBadRequest, "to must be greater than or equal to from")
		return
	}

	filter := subscriptions.TotalFilter{
		From:        from,
		To:          to,
		ServiceName: strings.TrimSpace(query.Get("service_name")),
	}
	if userIDRaw := strings.TrimSpace(query.Get("user_id")); userIDRaw != "" {
		userID, err := uuid.Parse(userIDRaw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "user_id must be a valid UUID")
			return
		}
		filter.UserID = &userID
	}

	total, err := h.repo.Total(r.Context(), filter)
	if err != nil {
		h.serverError(w, "calculate subscriptions total", err)
		return
	}

	h.logger.Info("subscriptions total calculated",
		slog.String("from", formatMonth(from)),
		slog.String("to", formatMonth(to)),
		slog.Int64("total", total),
	)
	writeJSON(w, http.StatusOK, totalResponse{Total: total})
}

func decodeSubscriptionRequest(w http.ResponseWriter, r *http.Request) (subscriptions.CreateInput, bool) {
	defer r.Body.Close()
	var request subscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return subscriptions.CreateInput{}, false
	}

	input, err := request.toInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return subscriptions.CreateInput{}, false
	}
	return input, true
}

func parsePathUUID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) handleRepoError(w http.ResponseWriter, operation string, err error) {
	if errors.Is(err, subscriptions.ErrNotFound) {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	h.serverError(w, operation, err)
}

func (h *Handler) serverError(w http.ResponseWriter, operation string, err error) {
	h.logger.Error(operation, slog.String("error", err.Error()))
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func (h *Handler) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		h.logger.Info("http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func parseInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
