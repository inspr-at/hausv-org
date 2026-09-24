package ai

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Destination-policy sentinels. Error text never includes the URL: an
// override can carry userinfo, and that must not land in logs.
var (
	ErrDestinationUserinfo = errors.New("ai: destination contains userinfo")
	ErrDestinationFragment = errors.New("ai: destination contains a fragment")
	ErrDestinationScheme   = errors.New("ai: destination scheme is not http or https")
	ErrDestinationHost     = errors.New("ai: destination host is missing or invalid")
	ErrDestinationHTTP     = errors.New("ai: plain http destination is not local")
)

// ValidateDestination decides whether an organisation may name raw as its AI
// endpoint. An empty string means there is no override.
//
// https is accepted for any host. Names are not resolved, so an https host can
// still be an attacker, or can later resolve to a private, link-local or
// metadata address (DNS rebinding). That residual SSRF risk is accepted for
// https. Callers check again at use time, and the HTTP client refuses
// cross-origin redirects so a 307 cannot move the request to another origin.
//
// http is accepted only for loopback, RFC 1918 (10/8, 172.16/12, 192.168/16)
// and IPv6 ULA (fc00::/7) literals, the name localhost, and hostnames ending
// in .local or .lan. Userinfo, fragments and other schemes are rejected.
func ValidateDestination(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.Contains(raw, "#") {
		return ErrDestinationFragment
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ErrDestinationHost
	}
	if parsed.User != nil {
		return ErrDestinationUserinfo
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ErrDestinationScheme
	}
	if parsed.Hostname() == "" {
		return ErrDestinationHost
	}
	if _, explicit := explicitPort(parsed.Host); explicit {
		port, err := strconv.Atoi(parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return ErrDestinationHost
		}
	}
	if scheme == "http" && !httpDestinationAllowed(parsed.Hostname()) {
		return ErrDestinationHTTP
	}
	return nil
}

// Origin returns scheme://host:port with a lowercase host and the default
// port (443 or 80) filled in. The path, query and userinfo are not part of
// the origin. ok is false when raw is not an http(s) URL with a host.
func Origin(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" {
		return "", false
	}
	port := ""
	if explicit, ok := explicitPort(parsed.Host); ok {
		number, err := strconv.Atoi(explicit)
		if err != nil || number < 1 || number > 65535 {
			return "", false
		}
		port = strconv.Itoa(number)
	} else if scheme == "https" {
		port = "443"
	} else {
		port = "80"
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return scheme + "://" + host + ":" + port, true
}

// SameOrigin reports whether left and right name the same scheme, host and port.
func SameOrigin(left, right string) bool {
	a, oka := Origin(left)
	b, okb := Origin(right)
	return oka && okb && a == b
}

func httpDestinationAllowed(hostname string) bool {
	host := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(hostname), "."))
	if host == "" {
		return false
	}
	if host == "localhost" || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".lan") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	// IPv4-mapped IPv6 is judged as the embedded IPv4 address, so a mapped
	// public address is not treated as private.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

// explicitPort reports a port written in the authority. ok is false when the
// authority has no port. IPv6 brackets are not a port.
func explicitPort(authority string) (string, bool) {
	if strings.HasPrefix(authority, "[") {
		end := strings.LastIndex(authority, "]")
		if end < 0 || end+1 >= len(authority) {
			return "", false
		}
		rest := authority[end+1:]
		if !strings.HasPrefix(rest, ":") || len(rest) < 2 {
			return "", false
		}
		return rest[1:], true
	}
	colon := strings.LastIndex(authority, ":")
	if colon < 0 || colon+1 >= len(authority) {
		return "", false
	}
	return authority[colon+1:], true
}

func aiHTTPClient() *http.Client {
	client := &http.Client{CheckRedirect: refuseCrossOriginRedirect}
	if http.DefaultClient != nil {
		client.Timeout = http.DefaultClient.Timeout
		if http.DefaultClient.Transport != nil {
			client.Transport = http.DefaultClient.Transport
		}
	}
	return client
}

// refuseCrossOriginRedirect keeps a completion on the origin that was already
// accepted. A 307/308 would otherwise forward the triage body, and possibly
// the process credential, to a host the organisation did not name.
func refuseCrossOriginRedirect(req *http.Request, via []*http.Request) error {
	if req == nil || req.URL == nil || len(via) == 0 || via[0] == nil || via[0].URL == nil {
		return errors.New("ai: redirect refused")
	}
	if len(via) >= 10 {
		return errors.New("ai: too many redirects")
	}
	if req.URL.User != nil || req.URL.Fragment != "" {
		return errors.New("ai: redirect destination refused")
	}
	if !SameOrigin(via[0].URL.String(), req.URL.String()) {
		return errors.New("ai: cross-origin redirect refused")
	}
	return nil
}
