package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestLookup(t *testing.T) {
	rec := doAuth(t, newRouter(), "POST", "/v1/lookup", `{"ids":[1256738876,42,1256738876,-1]}`, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body)
	}
	var got struct {
		Users map[string]json.RawMessage `json:"users"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Users) != 1 {
		t.Fatalf("users = %s, want only the seeded id", rec.Body)
	}
	want := `{"badge":{"emoji_id":5276360730361229322,"description":"seed"}}`
	if string(got.Users["1256738876"]) != want {
		t.Fatalf("user = %s, want %s", got.Users["1256738876"], want)
	}
}

func TestLookupEmpty(t *testing.T) {
	rec := doAuth(t, newRouter(), "POST", "/v1/lookup", `{"ids":[]}`, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"users":{}`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}

func TestLookupTooManyIDs(t *testing.T) {
	ids := make([]string, 101)
	for i := range ids {
		ids[i] = fmt.Sprint(i + 1)
	}
	body := `{"ids":[` + strings.Join(ids, ",") + `]}`
	if rec := doAuth(t, newRouter(), "POST", "/v1/lookup", body, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestLookupBadJSON(t *testing.T) {
	if rec := doAuth(t, newRouter(), "POST", "/v1/lookup", `{"ids":"x"}`, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
