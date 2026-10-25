package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
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

func TestRateLimitTrustedProxy(t *testing.T) {
	d := deps(seedService())
	d.LookupLimit = httpapi.RateLimit{Rate: 0, Burst: 1}
	d.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("172.16.0.0/12")}
	h := httpapi.NewRouter(d)

	send := func(remote, realIP string) int {
		req := httptest.NewRequest("POST", "/v1/lookup", strings.NewReader(`{"ids":[]}`))
		req.RemoteAddr = remote + ":1234"
		if realIP != "" {
			req.Header.Set("X-Real-IP", realIP)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	// Clients behind the proxy get separate buckets.
	if code := send("172.17.0.1", "1.1.1.1"); code != http.StatusOK {
		t.Fatalf("first client = %d", code)
	}
	if code := send("172.17.0.1", "2.2.2.2"); code != http.StatusOK {
		t.Fatalf("second client = %d, want 200", code)
	}
	if code := send("172.17.0.1", "1.1.1.1"); code != http.StatusTooManyRequests {
		t.Fatalf("repeat client = %d, want 429", code)
	}
	// An untrusted peer cannot spoof the header.
	if code := send("10.0.0.1", "3.3.3.3"); code != http.StatusOK {
		t.Fatalf("untrusted first = %d", code)
	}
	if code := send("10.0.0.1", "4.4.4.4"); code != http.StatusTooManyRequests {
		t.Fatalf("untrusted spoof = %d, want 429", code)
	}
}
