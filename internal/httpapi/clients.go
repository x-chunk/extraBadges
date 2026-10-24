package httpapi

import "net/http"

type initDataRequest struct {
	InitData string `json:"init_data"`
}

// registerClient marks the verified user as running the client.
func (h *Handler) registerClient(w http.ResponseWriter, r *http.Request) {
	id, ok := h.verifiedUser(w, r)
	if !ok {
		return
	}
	if err := h.clients.Touch(r.Context(), id, h.now()); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// hideClient hides the verified user's client mark until they register again.
func (h *Handler) hideClient(w http.ResponseWriter, r *http.Request) {
	id, ok := h.verifiedUser(w, r)
	if !ok {
		return
	}
	if err := h.clients.Hide(r.Context(), id); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// verifiedUser takes the user ID only from Telegram-signed initData.
func (h *Handler) verifiedUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	var req initDataRequest
	if !h.decode(w, r, &req) {
		return 0, false
	}
	id, err := h.verifyInitData(req.InitData)
	if err != nil {
		h.writeMessage(w, http.StatusUnauthorized, "invalid or expired init_data")
		return 0, false
	}
	return id, true
}
