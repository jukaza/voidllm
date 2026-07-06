package keys

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// ParseIPRules splits newline/comma-separated IP or CIDR rules.
func ParseIPRules(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	}) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// FormatIPRules joins rules for storage.
func FormatIPRules(rules []string) string {
	var cleaned []string
	for _, r := range rules {
		r = strings.TrimSpace(r)
		if r != "" {
			cleaned = append(cleaned, r)
		}
	}
	return strings.Join(cleaned, "\n")
}

// CompileIPRules parses rules into matchers. Invalid rules are skipped.
func CompileIPRules(rules []string) []func(net.IP) bool {
	var matchers []func(net.IP) bool
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		if strings.Contains(rule, "/") {
			_, cidr, err := net.ParseCIDR(rule)
			if err != nil {
				continue
			}
			netw := cidr
			matchers = append(matchers, func(ip net.IP) bool {
				return netw.Contains(ip)
			})
			continue
		}
		ip := net.ParseIP(rule)
		if ip == nil {
			continue
		}
		want := ip
		matchers = append(matchers, func(ip net.IP) bool {
			return ip.Equal(want)
		})
	}
	return matchers
}

// ClientIP returns the client address used for API key IP ACL checks.
// When trustForwarded is true, proxy headers may be honored (per Fiber config).
// When false, the direct TCP peer address is preferred.
func ClientIP(c fiber.Ctx, trustForwarded bool) string {
	if trustForwarded {
		return c.IP()
	}
	if ip := c.RequestCtx().RemoteIP(); ip != nil {
		return ip.String()
	}
	return c.IP()
}

// IPAllowed checks whitelist/blacklist against clientIP.
// Empty whitelist means allow all (except blacklist hits).
func IPAllowed(clientIP string, whitelist, blacklist []string) bool {
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return false
	}

	for _, m := range CompileIPRules(blacklist) {
		if m(ip) {
			return false
		}
	}

	if len(whitelist) == 0 {
		return true
	}
	for _, m := range CompileIPRules(whitelist) {
		if m(ip) {
			return true
		}
	}
	return false
}