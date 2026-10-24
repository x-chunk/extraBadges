package httpapi

import (
	"net/http"
	"strconv"
)

// maxLookupIDs caps how many users a single lookup may ask about.
const maxLookupIDs = 100

type lookupRequest struct {
	IDs []int64 `json:"ids"`
}

type lookupBadge struct {
	EmojiID     int64  `json:"emoji_id"`
	Description string `json:"description"`
}

type lookupUser struct {
	Client bool         `json:"client,omitempty"`
	Badge  *lookupBadge `json:"badge,omitempty"`
}

type lookupResponse struct {
	Users map[string]lookupUser `json:"users"`
}

// lookup reports client marks and badges for a batch of users. Users with nothing to show
// are left out of the response. The request body is never logged: it
// reveals whose profiles the caller is viewing.
func (h *Handler) lookup(w http.ResponseWriter, r *http.Request) {
	var req lookupRequest
	if !h.decode(w, r, &req) {
		return
	}
	if len(req.IDs) > maxLookupIDs {
		h.writeMessage(w, http.StatusBadRequest, "too many ids, max "+strconv.Itoa(maxLookupIDs))
		return
	}
	ids := make([]int64, 0, len(req.IDs))
	seen := make(map[int64]bool, len(req.IDs))
	for _, id := range req.IDs {
		if id > 0 && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	badges, err := h.svc.GetMany(r.Context(), ids)
	if err != nil {
		h.writeError(w, err)
		return
	}

	clients, err := h.clients.GetMany(r.Context(), ids)
	if err != nil {
		h.writeError(w, err)
		return
	}

	now := h.now()
	resp := lookupResponse{Users: make(map[string]lookupUser, len(badges))}
	for _, id := range ids {
		var u lookupUser
		if c, ok := clients[id]; ok && c.Visible(now, h.clientTTL) {
			u.Client = true
		}
		if b, ok := badges[id]; ok {
			u.Badge = &lookupBadge{EmojiID: b.EmojiID, Description: b.Description}
		}
		if u.Client || u.Badge != nil {
			resp.Users[strconv.FormatInt(id, 10)] = u
		}
	}
	h.writeJSON(w, http.StatusOK, resp)
}
