package telegram

import (
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"testing"
	"time"
)

const botToken = "123456:test-token"

var now = time.Unix(1_700_000_000, 0)

// signed builds initData the way Telegram does.
func signed(token string, authDate time.Time, user string) url.Values {
	v := url.Values{
		"auth_date": {strconv.FormatInt(authDate.Unix(), 10)},
		"query_id":  {"AAH"},
		"user":      {user},
	}
	v.Set("hash", hex.EncodeToString(sign(v, token)))
	return v
}

func TestValidate(t *testing.T) {
	const user = `{"id":1256738876,"first_name":"A"}`
	tampered := signed(botToken, now, user)
	tampered.Set("user", `{"id":42,"first_name":"A"}`)
	noUser := signed(botToken, now, "")

	tests := []struct {
		name    string
		raw     string
		wantID  int64
		wantErr error
	}{
		{"valid", signed(botToken, now, user).Encode(), 1256738876, nil},
		{"slightly old", signed(botToken, now.Add(-23*time.Hour), user).Encode(), 1256738876, nil},
		{"tampered field", tampered.Encode(), 0, ErrInvalid},
		{"foreign token", signed("999:other", now, user).Encode(), 0, ErrInvalid},
		{"expired", signed(botToken, now.Add(-25*time.Hour), user).Encode(), 0, ErrExpired},
		{"from the future", signed(botToken, now.Add(time.Hour), user).Encode(), 0, ErrInvalid},
		{"no hash", "auth_date=1&user=%7B%7D", 0, ErrInvalid},
		{"bad hash", "auth_date=1&hash=zz", 0, ErrInvalid},
		{"no user", noUser.Encode(), 0, ErrInvalid},
		{"bad query", "%zz", 0, ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := validate(tt.raw, botToken, 24*time.Hour, now)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil || id != tt.wantID {
				t.Fatalf("got %d, %v; want %d", id, err, tt.wantID)
			}
		})
	}
}

// TestKnownVector pins the algorithm against a hash computed independently
// (Python hmac), not just a round trip through sign.
func TestKnownVector(t *testing.T) {
	v := url.Values{"auth_date": {"1"}, "user": {`{"id":7}`}}
	const want = "94dd42ad1a89baf32e45a6ed7f42f46fbe0179fb045aea5c079274756c9ae644"
	if got := hex.EncodeToString(sign(v, "token")); got != want {
		t.Fatalf("hash = %s, want %s", got, want)
	}
}
