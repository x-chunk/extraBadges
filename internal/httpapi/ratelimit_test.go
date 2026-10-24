package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"extra-badges/internal/httpapi"
)

func TestRateLimit(t *testing.T) {
	d := deps(seedService())
	d.ClientLimit = httpapi.RateLimit{Rate: 0, Burst: 2}
	d.LookupLimit = httpapi.RateLimit{Rate: 0, Burst: 1}
	h := httpapi.NewRouter(d)

	send := func(method, path, body, ip string) int {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := send("POST", "/v1/lookup", `{"ids":[]}`, "10.0.0.1"); code != http.StatusOK {
		t.Fatalf("first lookup = %d", code)
	}
	if code := send("POST", "/v1/lookup", `{"ids":[]}`, "10.0.0.1"); code != http.StatusTooManyRequests {
		t.Fatalf("second lookup = %d, want 429", code)
	}
	if code := send("POST", "/v1/lookup", `{"ids":[]}`, "10.0.0.2"); code != http.StatusOK {
		t.Fatalf("other IP lookup = %d, want 200", code)
	}

	// register and me share a bucket.
	if code := send("POST", "/v1/clients/register", `{"init_data":"user:1"}`, "10.0.0.1"); code != http.StatusNoContent {
		t.Fatalf("register = %d", code)
	}
	if code := send("DELETE", "/v1/clients/me", `{"init_data":"user:1"}`, "10.0.0.1"); code != http.StatusNoContent {
		t.Fatalf("me = %d", code)
	}
	if code := send("POST", "/v1/clients/register", `{"init_data":"user:1"}`, "10.0.0.1"); code != http.StatusTooManyRequests {
		t.Fatalf("third client call = %d, want 429", code)
	}

	// Admin and health endpoints are not limited.
	if code := send("GET", "/healthz", "", "10.0.0.1"); code != http.StatusOK {
		t.Fatalf("healthz = %d", code)
	}
}
