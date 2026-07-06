package admin

import "testing"

func TestDeviceLabelFromUserAgent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		ua   string
		want string
	}{
		{"", "Unknown device"},
		{"curl/8.5.0", "curl CLI"},
		{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36", "Chrome on Linux"},
	}
	for _, tc := range tests {
		if got := deviceLabelFromUserAgent(tc.ua); got != tc.want {
			t.Errorf("deviceLabelFromUserAgent(%q) = %q, want %q", tc.ua, got, tc.want)
		}
	}
}