package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"extra-badges/internal/badge"
	"extra-badges/internal/httpapi"
	"extra-badges/internal/storage/memory"
)

func newRouter() http.Handler {
	return httpapi.NewRouter(deps(seedService()))
}

func seedService() httpapi.BadgeService {
	return badge.NewService(memory.New(badge.Badge{UserID: 1256738876, EmojiID: 5276360730361229322, Description: "seed"}))
}

const adminToken = "test-token"

var testNow = time.Unix(1_700_000_000, 0)

func deps(svc httpapi.BadgeService) httpapi.Deps {
	return httpapi.Deps{
		Badges:     svc,
		Clients:    memory.NewClients(),
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		AdminToken: adminToken,
		// Accepts "user:<id>" so tests need not sign real initData.
		VerifyInitData: func(raw string) (int64, error) {
			id, err := strconv.ParseInt(strings.TrimPrefix(raw, "user:"), 10, 64)
			if err != nil || !strings.HasPrefix(raw, "user:") {
				return 0, errors.New("bad init data")
			}
			return id, nil
		},
		ClientTTL: time.Hour,
		Now:       func() time.Time { return testNow },
	}
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doAuth(t, h, method, path, body, "Bearer "+adminToken)
}

func doAuth(t *testing.T, h http.Handler, method, path, body, auth string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, r)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	h.ServeHTTP(rec, req)
	return rec
}

func TestRoutes(t *testing.T) {
	tests := []struct {
		name, method, path, body string
		status                   int
		contains                 string
	}{
		{"health", "GET", "/healthz", "", 200, `"ok"`},
		{"get seed keeps int64 precision", "GET", "/v1/admin/badges/1256738876", "", 200, `"emoji_id":5276360730361229322`},
		{"get missing", "GET", "/v1/admin/badges/42", "", 404, "not found"},
		{"get bad id", "GET", "/v1/admin/badges/abc", "", 400, "positive integer"},
		{"get negative id", "GET", "/v1/admin/badges/-1", "", 400, "positive integer"},
		{"list", "GET", "/v1/admin/badges", "", 200, `"user_id":1256738876`},
		{"create", "POST", "/v1/admin/badges", `{"user_id":5,"emoji_id":7}`, 201, `"description":""`},
		{"create duplicate", "POST", "/v1/admin/badges", `{"user_id":1256738876,"emoji_id":7}`, 409, "already exists"},
		{"create invalid", "POST", "/v1/admin/badges", `{"user_id":5}`, 400, "emoji_id"},
		{"create bad json", "POST", "/v1/admin/badges", `{`, 400, "invalid JSON"},
		{"create unknown field", "POST", "/v1/admin/badges", `{"user_id":5,"emoji_id":7,"x":1}`, 400, "invalid JSON"},
		{"put", "PUT", "/v1/admin/badges/9", `{"emoji_id":3,"description":"t"}`, 200, `"user_id":9`},
		{"put mismatch", "PUT", "/v1/admin/badges/9", `{"user_id":8,"emoji_id":3}`, 400, "does not match"},
		{"put invalid", "PUT", "/v1/admin/badges/9", `{"emoji_id":0}`, 400, "emoji_id"},
		{"put bad id", "PUT", "/v1/admin/badges/x", `{"emoji_id":1}`, 400, "positive integer"},
		{"put bad json", "PUT", "/v1/admin/badges/9", `nope`, 400, "invalid JSON"},
		{"patch description", "PATCH", "/v1/admin/badges/1256738876", `{"description":"new"}`, 200, `"emoji_id":5276360730361229322,"description":"new"`},
		{"patch emoji", "PATCH", "/v1/admin/badges/1256738876", `{"emoji_id":3}`, 200, `"emoji_id":3,"description":"seed"`},
		{"patch missing", "PATCH", "/v1/admin/badges/42", `{"description":"x"}`, 404, "not found"},
		{"patch invalid", "PATCH", "/v1/admin/badges/1256738876", `{"emoji_id":0}`, 400, "emoji_id"},
		{"patch unknown field", "PATCH", "/v1/admin/badges/1256738876", `{"user_id":1}`, 400, "invalid JSON"},
		{"patch bad id", "PATCH", "/v1/admin/badges/x", `{}`, 400, "positive integer"},
		{"delete", "DELETE", "/v1/admin/badges/1256738876", "", 204, ""},
		{"delete missing", "DELETE", "/v1/admin/badges/42", "", 404, "not found"},
		{"delete bad id", "DELETE", "/v1/admin/badges/0", "", 400, "positive integer"},
		{"method not allowed", "POST", "/v1/admin/badges/1", "", 405, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, newRouter(), tt.method, tt.path, tt.body)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.status, rec.Body)
			}
			if !strings.Contains(rec.Body.String(), tt.contains) {
				t.Errorf("body %q does not contain %q", rec.Body, tt.contains)
			}
		})
	}
}

func TestCreateThenGet(t *testing.T) {
	h := newRouter()
	rec := do(t, h, "POST", "/v1/admin/badges", `{"user_id":77,"emoji_id":88,"description":"hello"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if loc := rec.Header().Get("Location"); loc != "/v1/admin/badges/77" {
		t.Errorf("Location = %q", loc)
	}
	rec = do(t, h, "GET", "/v1/admin/badges/77", "")
	var got badge.Badge
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got != (badge.Badge{UserID: 77, EmojiID: 88, Description: "hello"}) {
		t.Fatalf("got %+v", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

type failingService struct{ httpapi.BadgeService }

func (failingService) List(context.Context) ([]badge.Badge, error) {
	return nil, errors.New("db down")
}

func TestInternalError(t *testing.T) {
	h := httpapi.NewRouter(deps(failingService{}))
	rec := do(t, h, "GET", "/v1/admin/badges", "")
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func TestAdminAuth(t *testing.T) {
	h := newRouter()
	for name, auth := range map[string]string{
		"missing":   "",
		"wrong":     "Bearer nope",
		"no scheme": adminToken,
	} {
		t.Run(name, func(t *testing.T) {
			if rec := doAuth(t, h, "GET", "/v1/admin/badges", "", auth); rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
		})
	}
}

func TestOldPathsRemoved(t *testing.T) {
	if rec := do(t, newRouter(), "GET", "/badges/1256738876", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
