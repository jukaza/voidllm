package admin

import "strings"

// deviceLabelFromUserAgent returns a short human-readable device label.
func deviceLabelFromUserAgent(ua string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return "Unknown device"
	}
	lower := strings.ToLower(ua)

	browser := "Browser"
	switch {
	case strings.HasPrefix(lower, "curl/"):
		return "curl CLI"
	case strings.HasPrefix(lower, "wget/"):
		return "wget CLI"
	case strings.Contains(lower, "edg/"):
		browser = "Edge"
	case strings.Contains(lower, "chrome/") && !strings.Contains(lower, "chromium"):
		browser = "Chrome"
	case strings.Contains(lower, "firefox/"):
		browser = "Firefox"
	case strings.Contains(lower, "safari/") && !strings.Contains(lower, "chrome/"):
		browser = "Safari"
	}

	os := "Unknown OS"
	switch {
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		os = "iOS"
	case strings.Contains(lower, "android"):
		os = "Android"
	case strings.Contains(lower, "mac os x") || strings.Contains(lower, "macintosh"):
		os = "macOS"
	case strings.Contains(lower, "windows"):
		os = "Windows"
	case strings.Contains(lower, "linux"):
		os = "Linux"
	}

	return browser + " on " + os
}