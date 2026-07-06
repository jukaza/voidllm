package keys

import (
	"encoding/json"
	"strings"
)

// ParseModelLimits decodes a JSON array of model names.
func ParseModelLimits(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var models []string
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return nil
	}
	out := make([]string, 0, len(models))
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m != "" {
			out = append(out, m)
		}
	}
	return out
}

// FormatModelLimits encodes model names as JSON.
func FormatModelLimits(models []string) string {
	if len(models) == 0 {
		return "[]"
	}
	b, err := json.Marshal(models)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// ModelAllowed reports whether model is permitted for the key.
func ModelAllowed(model string, enabled bool, limits []string) bool {
	if !enabled || len(limits) == 0 {
		return true
	}
	model = strings.TrimSpace(model)
	for _, allowed := range limits {
		if strings.EqualFold(allowed, model) {
			return true
		}
	}
	return false
}