package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	authRateWindow              = 15 * time.Minute
	genericAuthRateLimitMessage = "Anfrage vorübergehend begrenzt. Bitte später erneut versuchen."
)

var (
	magicLinkSourcePolicy  = authRatePolicy{scope: "magic-source", limit: 20, window: authRateWindow}
	magicLinkAccountPolicy = authRatePolicy{
		scope:  "magic-account",
		limit:  5,
		window: authRateWindow,
	}
	oidcStartSourcePolicy       = authRatePolicy{scope: "oidc-start-source", limit: 20, window: authRateWindow}
	loginCompletionSourcePolicy = authRatePolicy{
		scope:  "login-completion-source",
		limit:  60,
		window: authRateWindow,
	}
	loginCompletionAccountPolicy = authRatePolicy{
		scope:  "login-completion-account",
		limit:  10,
		window: authRateWindow,
	}
)

type authRatePolicy struct {
	scope  string
	limit  int
	window time.Duration
}

type authRateCheck struct {
	policy authRatePolicy
	value  string
}

type authRateBucket struct {
	count   int
	resetAt time.Time
}

type trustedProxyAllowlist struct {
	prefixes []netip.Prefix
}

// parseTrustedProxyAllowlist deliberately has different local and production
// defaults. A local app may trust its own loopback reverse proxy without extra
// configuration. A publicly addressed app must name every trusted proxy
// explicitly; silently trusting a complete Docker/private network would let
// any other workload on that network spoof the client identity.
func parseTrustedProxyAllowlist(raw string, publicURL bool) (trustedProxyAllowlist, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if publicURL {
			return trustedProxyAllowlist{}, fmt.Errorf("TRUSTED_PROXY_CIDRS is required when BASE_URL is public")
		}
		raw = "127.0.0.1/32,::1/128"
	}

	out := trustedProxyAllowlist{}
	seen := map[netip.Prefix]struct{}{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(item)
		if err != nil {
			addr, addrErr := netip.ParseAddr(item)
			if addrErr != nil {
				return trustedProxyAllowlist{}, fmt.Errorf("invalid TRUSTED_PROXY_CIDRS entry")
			}
			prefix = netip.PrefixFrom(addr, addr.BitLen())
		}
		prefix = prefix.Masked()
		if prefix.Bits() == 0 {
			return trustedProxyAllowlist{}, fmt.Errorf("TRUSTED_PROXY_CIDRS must not trust the entire address space")
		}
		if _, ok := seen[prefix]; ok {
			continue
		}
		seen[prefix] = struct{}{}
		out.prefixes = append(out.prefixes, prefix)
	}
	if len(out.prefixes) == 0 {
		return trustedProxyAllowlist{}, fmt.Errorf("TRUSTED_PROXY_CIDRS must contain at least one address or CIDR")
	}
	return out, nil
}

func (a trustedProxyAllowlist) contains(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	for _, prefix := range a.prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// authRateLimiter intentionally retains only SHA-256 digests of source/account
// keys. It is process-local because HAUSV currently runs exactly one app
// replica; the deployment contract documents that a shared limiter is required
// before adding replicas.
type authRateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	buckets map[string]authRateBucket
}

func newAuthRateLimiter(now func() time.Time) *authRateLimiter {
	if now == nil {
		now = time.Now
	}
	return &authRateLimiter{
		now:     now,
		buckets: map[string]authRateBucket{},
	}
}

// Allow evaluates and increments all checks atomically. If any bucket is
// exhausted, no bucket is incremented and the longest remaining retry delay is
// returned.
func (l *authRateLimiter) Allow(checks ...authRateCheck) (bool, time.Duration) {
	if l == nil {
		return true, 0
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	for key, bucket := range l.buckets {
		if !now.Before(bucket.resetAt) {
			delete(l.buckets, key)
		}
	}

	retryAfter := time.Duration(0)
	for _, check := range checks {
		if check.policy.limit <= 0 || check.policy.window <= 0 {
			continue
		}
		key := authRateKey(check.policy.scope, check.value)
		bucket, ok := l.buckets[key]
		if ok && bucket.count >= check.policy.limit {
			if remaining := bucket.resetAt.Sub(now); remaining > retryAfter {
				retryAfter = remaining
			}
		}
	}
	if retryAfter > 0 {
		return false, retryAfter
	}

	for _, check := range checks {
		if check.policy.limit <= 0 || check.policy.window <= 0 {
			continue
		}
		key := authRateKey(check.policy.scope, check.value)
		bucket, ok := l.buckets[key]
		if !ok {
			bucket.resetAt = now.Add(check.policy.window)
		}
		bucket.count++
		l.buckets[key] = bucket
	}
	return true, 0
}

func authRateKey(scope string, value string) string {
	digest := sha256.Sum256([]byte(scope + "\x00" + strings.TrimSpace(value)))
	return scope + ":" + hex.EncodeToString(digest[:])
}

func (a *app) getAuthRateLimiter() *authRateLimiter {
	a.authLimiterMu.Lock()
	defer a.authLimiterMu.Unlock()
	if a.authLimiter == nil {
		a.authLimiter = newAuthRateLimiter(time.Now)
	}
	return a.authLimiter
}

func (a *app) allowAuthRequest(
	w http.ResponseWriter,
	r *http.Request,
	sourcePolicy authRatePolicy,
	accountPolicy authRatePolicy,
	account string,
) bool {
	checks := []authRateCheck{{
		policy: sourcePolicy,
		value:  a.authRequestSource(r),
	}}
	if strings.TrimSpace(account) != "" && accountPolicy.limit > 0 {
		checks = append(checks, authRateCheck{
			policy: accountPolicy,
			value:  strings.ToLower(strings.TrimSpace(account)),
		})
	}
	allowed, retryAfter := a.getAuthRateLimiter().Allow(checks...)
	if allowed {
		return true
	}
	writeAuthRateLimited(w, retryAfter)
	return false
}

func writeAuthRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := int64((retryAfter + time.Second - 1) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
	http.Error(w, genericAuthRateLimitMessage, http.StatusTooManyRequests)
}

// authRequestSource accepts X-Real-IP only from a peer named in the explicit
// proxy allowlist. X-Forwarded-For is deliberately ignored because clients can
// append/spoof it.
func (a *app) authRequestSource(r *http.Request) string {
	peer := requestPeerIP(r)
	if a != nil && a.trustedProxies.contains(peer) {
		if forwarded := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); forwarded != nil {
			return forwarded.String()
		}
	}
	if peer != nil {
		return peer.String()
	}
	return "unknown"
}

func requestPeerIP(r *http.Request) net.IP {
	if r == nil {
		return nil
	}
	remote := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	remote = strings.Trim(remote, "[]")
	if zone := strings.LastIndex(remote, "%"); zone >= 0 {
		remote = remote[:zone]
	}
	return net.ParseIP(remote)
}
