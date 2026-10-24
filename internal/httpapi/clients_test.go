package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"extra-badges/internal/client"
	"extra-badges/internal/httpapi"
	"extra-badges/internal/storage/memory"
)

func TestRegisterAndHide(t *testing.T) {
	h := newRouter()
	lookup := func() string {
		t.Helper()
		return doAuth(t, h, "POST", "/v1/lookup", `{"ids":[5]}`, "").Body.String()
	}

	if rec := doAuth(t, h, "POST", "/v1/clients/register", `{"init_data":"user:5"}`, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("register: %d %s", rec.Code, rec.Body)
	}
	if got, want := lookup(), `{"users":{"5":{"client":true}}}`+"\n"; got != want {
		t.Fatalf("after register: %s, want %s", got, want)
	}
	if rec := doAuth(t, h, "DELETE", "/v1/clients/me", `{"init_data":"user:5"}`, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("hide: %d %s", rec.Code, rec.Body)
	}
	if got, want := lookup(), `{"users":{}}`+"\n"; got != want {
		t.Fatalf("after hide: %s, want %s", got, want)
	}
	_ = doAuth(t, h, "POST", "/v1/clients/register", `{"init_data":"user:5"}`, "")
	if got := lookup(); got == `{"users":{}}`+"\n" {
		t.Fatal("register after hide did not restore the mark")
	}
}

func TestRegisterRejectsBadInitData(t *testing.T) {
	h := newRouter()
	for _, path := range []string{"/v1/clients/register", "/v1/clients/me"} {
		method := "POST"
		if path == "/v1/clients/me" {
			method = "DELETE"
		}
		if rec := doAuth(t, h, method, path, `{"init_data":"forged"}`, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want 401", method, path, rec.Code)
		}
		if rec := doAuth(t, h, method, path, `nope`, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s %s bad json: status = %d, want 400", method, path, rec.Code)
		}
	}
}

func TestLookupClientMarkExpires(t *testing.T) {
	d := deps(seedService())
	d.Clients = memory.NewClients(
		client.Client{UserID: 1256738876, LastSeen: testNow.Add(-2 * time.Hour)},
		client.Client{UserID: 7, LastSeen: testNow.Add(-2 * time.Hour)},
		client.Client{UserID: 8, LastSeen: testNow.Add(-time.Minute)},
	)
	rec := doAuth(t, httpapi.NewRouter(d), "POST", "/v1/lookup", `{"ids":[1256738876,7,8]}`, "")
	want := `{"users":{"1256738876":{"badge":{"emoji_id":5276360730361229322,"description":"seed"}},"8":{"client":true}}}` + "\n"
	if rec.Body.String() != want {
		t.Fatalf("lookup = %s, want %s", rec.Body, want)
	}
}
