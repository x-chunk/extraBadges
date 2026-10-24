// Package telegram verifies data signed by Telegram.
package telegram

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalid = errors.New("invalid init data")
	ErrExpired = errors.New("init data expired")
)

// maxClockSkew tolerates an auth_date slightly ahead of the server clock.
const maxClockSkew = time.Minute

// ValidateInitData checks a Mini App initData query string signed with
// botToken and returns the user ID it vouches for. The data must be
// younger than maxAge. See
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app.
func ValidateInitData(raw, botToken string, maxAge time.Duration) (int64, error) {
	return validate(raw, botToken, maxAge, time.Now())
}

func validate(raw, botToken string, maxAge time.Duration, now time.Time) (int64, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return 0, errors.Join(ErrInvalid, err)
	}
	gotHash, err := hex.DecodeString(values.Get("hash"))
	if err != nil || len(gotHash) == 0 {
		return 0, errors.Join(ErrInvalid, errors.New("missing or malformed hash"))
	}
	if !hmac.Equal(gotHash, sign(values, botToken)) {
		return 0, errors.Join(ErrInvalid, errors.New("hash mismatch"))
	}

	authUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return 0, errors.Join(ErrInvalid, errors.New("malformed auth_date"))
	}
	age := now.Sub(time.Unix(authUnix, 0))
	if age > maxAge {
		return 0, ErrExpired
	}
	if age < -maxClockSkew {
		return 0, errors.Join(ErrInvalid, errors.New("auth_date is in the future"))
	}

	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID <= 0 {
		return 0, errors.Join(ErrInvalid, errors.New("missing or malformed user"))
	}
	return user.ID, nil
}

// sign computes the expected hash over every field except hash itself.
func sign(values url.Values, botToken string) []byte {
	keys := make([]string, 0, len(values))
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	lines := make([]string, len(keys))
	for i, k := range keys {
		lines[i] = k + "=" + values.Get(k)
	}

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(lines, "\n")))
	return mac.Sum(nil)
}
