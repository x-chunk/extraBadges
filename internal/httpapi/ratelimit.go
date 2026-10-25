package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimit is a per-IP token bucket. A zero Burst disables limiting.
type RateLimit struct {
	Rate  rate.Limit
	Burst int
}

var (
	// DefaultClientLimit guards register and me: a client calls them about once a day.
	DefaultClientLimit = RateLimit{Rate: rate.Every(10 * time.Second), Burst: 5}
	// DefaultLookupLimit guards lookup and config, which a client calls while browsing.
	DefaultLookupLimit = RateLimit{Rate: 5, Burst: 30}
)

// limiterIdle is how long an IP's bucket is kept after its last request.
const limiterIdle = 10 * time.Minute

type ipLimiter struct {
	limit RateLimit
	now   func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

func newIPLimiter(limit RateLimit, now func() time.Time) *ipLimiter {
	return &ipLimiter{limit: limit, now: now, buckets: make(map[string]*bucket)}
}

func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	// Sweep idle buckets so arbitrary source IPs cannot grow the map forever.
	if now.Sub(l.lastSweep) > limiterIdle {
		for k, b := range l.buckets {
			if now.Sub(b.lastSeen) > limiterIdle {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}
	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{lim: rate.NewLimiter(l.limit.Rate, l.limit.Burst)}
		l.buckets[ip] = b
	}
	b.lastSeen = now
	return b.lim.AllowN(now, 1)
}

// clientIP returns the address a request came from. When the connection comes
// from a trusted proxy, the proxy-supplied X-Real-IP header wins.
func clientIP(r *http.Request, trusted []netip.Prefix) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	addr = addr.Unmap()
	for _, p := range trusted {
		if !p.Contains(addr) {
			continue
		}
		if real, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get("X-Real-IP"))); err == nil {
			return real.Unmap().String()
		}
		break
	}
	return addr.String()
}

// rateLimit returns a middleware enforcing a per-IP limit keyed on the
// client address. Every handler it wraps shares the same buckets.
func (h *Handler) rateLimit(limit RateLimit) func(http.HandlerFunc) http.HandlerFunc {
	if limit.Burst <= 0 {
		return func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	l := newIPLimiter(limit, h.now)
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !l.allow(clientIP(r, h.trustedProxies)) {
				w.Header().Set("Retry-After", "10")
				h.writeMessage(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next(w, r)
		}
	}
}
