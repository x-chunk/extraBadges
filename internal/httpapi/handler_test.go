package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"extra-badges/internal/badge"
	"extra-badges/internal/httpapi"
	"extra-badges/internal/storage/memory"
)

func newRouter() http.Handler {
	repo := memory.New(badge.Badge{UserID: 1256738876, EmojiID: 5276360730361229322, Title: "seed"})
	return httpapi.NewRouter(badge.NewService(repo), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, r))
	return rec
}

func TestRoutes(t *testing.T) {
	tests := []struct {
		name, method, path, body string
		status                   int
		contains                 string
	}{
		{"health", "GET", "/healthz", "", 200, `"ok"`},
		{"get seed keeps int64 precision", "GET", "/badges/1256738876", "", 200, `"emoji_id":5276360730361229322`},
		{"get missing", "GET", "/badges/42", "", 404, "not found"},
		{"get bad id", "GET", "/badges/abc", "", 400, "positive integer"},
		{"get negative id", "GET", "/badges/-1", "", 400, "positive integer"},
		{"list", "GET", "/badges", "", 200, `"user_id":1256738876`},
		{"create", "POST", "/badges", `{"user_id":5,"emoji_id":7}`, 201, badge.DefaultTitle},
		{"create duplicate", "POST", "/badges", `{"user_id":1256738876,"emoji_id":7}`, 409, "already exists"},
		{"create invalid", "POST", "/badges", `{"user_id":5}`, 400, "emoji_id"},
		{"create bad json", "POST", "/badges", `{`, 400, "invalid JSON"},
		{"create unknown field", "POST", "/badges", `{"user_id":5,"emoji_id":7,"x":1}`, 400, "invalid JSON"},
		{"put", "PUT", "/badges/9", `{"emoji_id":3,"title":"t"}`, 200, `"user_id":9`},
		{"put mismatch", "PUT", "/badges/9", `{"user_id":8,"emoji_id":3}`, 400, "does not match"},
		{"put invalid", "PUT", "/badges/9", `{"emoji_id":0}`, 400, "emoji_id"},
		{"put bad id", "PUT", "/badges/x", `{"emoji_id":1}`, 400, "positive integer"},
		{"put bad json", "PUT", "/badges/9", `nope`, 400, "invalid JSON"},
		{"patch title", "PATCH", "/badges/1256738876", `{"title":"new"}`, 200, `"emoji_id":5276360730361229322,"title":"new"`},
		{"patch emoji", "PATCH", "/badges/1256738876", `{"emoji_id":3}`, 200, `"emoji_id":3,"title":"seed"`},
		{"patch missing", "PATCH", "/badges/42", `{"title":"x"}`, 404, "not found"},
		{"patch invalid", "PATCH", "/badges/1256738876", `{"emoji_id":0}`, 400, "emoji_id"},
		{"patch unknown field", "PATCH", "/badges/1256738876", `{"user_id":1}`, 400, "invalid JSON"},
		{"patch bad id", "PATCH", "/badges/x", `{}`, 400, "positive integer"},
		{"delete", "DELETE", "/badges/1256738876", "", 204, ""},
		{"delete missing", "DELETE", "/badges/42", "", 404, "not found"},
		{"delete bad id", "DELETE", "/badges/0", "", 400, "positive integer"},
		{"method not allowed", "POST", "/badges/1", "", 405, ""},
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
	rec := do(t, h, "POST", "/badges", `{"user_id":77,"emoji_id":88,"title":"hello"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if loc := rec.Header().Get("Location"); loc != "/badges/77" {
		t.Errorf("Location = %q", loc)
	}
	rec = do(t, h, "GET", "/badges/77", "")
	var got badge.Badge
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got != (badge.Badge{UserID: 77, EmojiID: 88, Title: "hello"}) {
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
	h := httpapi.NewRouter(failingService{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	rec := do(t, h, "GET", "/badges", "")
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}
