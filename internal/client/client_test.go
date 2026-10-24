package client

import (
	"testing"
	"time"
)

func TestVisible(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	ttl := time.Hour
	tests := []struct {
		name string
		c    Client
		want bool
	}{
		{"fresh", Client{LastSeen: now.Add(-time.Minute)}, true},
		{"expired", Client{LastSeen: now.Add(-2 * time.Hour)}, false},
		{"hidden", Client{LastSeen: now, Hidden: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.Visible(now, ttl); got != tt.want {
				t.Fatalf("Visible = %v, want %v", got, tt.want)
			}
		})
	}
}
