// Package httpapi exposes badges over HTTP.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"extra-badges/internal/badge"
)

// BadgeService is what the HTTP layer needs from the domain.
type BadgeService interface {
	Get(ctx context.Context, userID int64) (badge.Badge, error)
	List(ctx context.Context) ([]badge.Badge, error)
	Create(ctx context.Context, b badge.Badge) (badge.Badge, error)
	Put(ctx context.Context, b badge.Badge) (badge.Badge, error)
	Update(ctx context.Context, userID int64, p badge.Patch) (badge.Badge, error)
	Delete(ctx context.Context, userID int64) error
}

type Handler struct {
	svc BadgeService
	log *slog.Logger
}

const maxBodyBytes = 1 << 16

// NewRouter builds the HTTP routes.
func NewRouter(svc BadgeService, log *slog.Logger) http.Handler {
	h := &Handler{svc: svc, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /badges", h.list)
	mux.HandleFunc("POST /badges", h.create)
	mux.HandleFunc("GET /badges/{userID}", h.get)
	mux.HandleFunc("PUT /badges/{userID}", h.put)
	mux.HandleFunc("PATCH /badges/{userID}", h.patch)
	mux.HandleFunc("DELETE /badges/{userID}", h.delete)
	return mux
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	badges, err := h.svc.List(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, badges)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := h.userID(w, r)
	if !ok {
		return
	}
	b, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, b)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var b badge.Badge
	if !h.decode(w, r, &b) {
		return
	}
	created, err := h.svc.Create(r.Context(), b)
	if err != nil {
		h.writeError(w, err)
		return
	}
	w.Header().Set("Location", "/badges/"+strconv.FormatInt(created.UserID, 10))
	h.writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	id, ok := h.userID(w, r)
	if !ok {
		return
	}
	var b badge.Badge
	if !h.decode(w, r, &b) {
		return
	}
	if b.UserID != 0 && b.UserID != id {
		h.writeMessage(w, http.StatusBadRequest, "user_id in body does not match path")
		return
	}
	b.UserID = id
	saved, err := h.svc.Put(r.Context(), b)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, saved)
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id, ok := h.userID(w, r)
	if !ok {
		return
	}
	var p badge.Patch
	if !h.decode(w, r, &p) {
		return
	}
	updated, err := h.svc.Update(r.Context(), id, p)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.userID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) userID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil || id <= 0 {
		h.writeMessage(w, http.StatusBadRequest, "user_id must be a positive integer")
		return 0, false
	}
	return id, true
}

func (h *Handler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		h.writeMessage(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, badge.ErrNotFound):
		h.writeMessage(w, http.StatusNotFound, err.Error())
	case errors.Is(err, badge.ErrAlreadyExists):
		h.writeMessage(w, http.StatusConflict, err.Error())
	case errors.Is(err, badge.ErrInvalid):
		h.writeMessage(w, http.StatusBadRequest, err.Error())
	default:
		h.log.Error("request failed", "err", err)
		h.writeMessage(w, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) writeMessage(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]string{"error": msg})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.log.Error("encode response", "err", err)
	}
}
