package keys

// API key status values.
const (
	StatusActive         = "active"
	StatusDisabled       = "disabled"
	StatusExpired        = "expired"
	StatusQuotaExhausted = "quota_exhausted"
)

// ValidStatus reports whether s is a known key status.
func ValidStatus(s string) bool {
	switch s {
	case StatusActive, StatusDisabled, StatusExpired, StatusQuotaExhausted:
		return true
	default:
		return false
	}
}

// IsUsable reports whether a key with the given status may authenticate requests.
func IsUsable(status string) bool {
	return status == StatusActive
}