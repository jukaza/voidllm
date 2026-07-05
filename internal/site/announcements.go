package site

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Announcement is one rich-text system notice (markdown content).
type Announcement struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	PublishDate string `json:"publish_date"`
	Type        string `json:"type"`
	Extra       string `json:"extra,omitempty"`
}

var validAnnouncementTypes = map[string]bool{
	"default": true,
	"ongoing": true,
	"success": true,
	"warning": true,
	"error":   true,
}

const maxAnnouncements = 100
const maxAnnouncementContent = 8000
const maxAnnouncementExtra = 1000

func parseAnnouncements(raw string) ([]Announcement, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []Announcement{}, nil
	}
	var items []Announcement
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, fmt.Errorf("invalid announcements JSON: %w", err)
	}
	return normalizeAnnouncements(items)
}

func normalizeAnnouncements(items []Announcement) ([]Announcement, error) {
	if len(items) > maxAnnouncements {
		return nil, fmt.Errorf("announcements cannot exceed %d items", maxAnnouncements)
	}
	out := make([]Announcement, 0, len(items))
	for i, item := range items {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			return nil, fmt.Errorf("announcement %d: content is required", i+1)
		}
		if len(content) > maxAnnouncementContent {
			return nil, fmt.Errorf("announcement %d: content exceeds %d characters", i+1, maxAnnouncementContent)
		}
		extra := strings.TrimSpace(item.Extra)
		if len(extra) > maxAnnouncementExtra {
			return nil, fmt.Errorf("announcement %d: extra exceeds %d characters", i+1, maxAnnouncementExtra)
		}
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = uuid.NewString()
		}
		typ := strings.TrimSpace(item.Type)
		if typ == "" {
			typ = "default"
		}
		if !validAnnouncementTypes[typ] {
			return nil, fmt.Errorf("announcement %d: invalid type %q", i+1, typ)
		}
		publishDate := strings.TrimSpace(item.PublishDate)
		if publishDate == "" {
			publishDate = time.Now().UTC().Format(time.RFC3339)
		}
		if _, err := time.Parse(time.RFC3339, publishDate); err != nil {
			return nil, fmt.Errorf("announcement %d: invalid publish_date", i+1)
		}
		out = append(out, Announcement{
			ID:          id,
			Content:     content,
			PublishDate: publishDate,
			Type:        typ,
			Extra:       extra,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return announcementTime(out[i]).After(announcementTime(out[j]))
	})
	return out, nil
}

func announcementTime(a Announcement) time.Time {
	t, err := time.Parse(time.RFC3339, a.PublishDate)
	if err != nil {
		return time.Time{}
	}
	return t
}

func marshalAnnouncements(items []Announcement) (string, error) {
	normalized, err := normalizeAnnouncements(items)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("marshal announcements: %w", err)
	}
	return string(b), nil
}

func loadAnnouncements(ctx context.Context, store SettingsStore) ([]Announcement, error) {
	raw, err := store.GetSetting(ctx, KeyAnnouncements)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) != "" {
		items, err := parseAnnouncements(raw)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return maybeSeedDemoAnnouncements(ctx, store)
		}
		return items, nil
	}

	// Migrate legacy single-notice field.
	legacy, err := store.GetSetting(ctx, KeyNotice)
	if err != nil {
		return nil, err
	}
	legacy = strings.TrimSpace(legacy)
	if legacy == "" {
		return maybeSeedDemoAnnouncements(ctx, store)
	}

	items := []Announcement{{
		ID:          uuid.NewString(),
		Content:     legacy,
		PublishDate: time.Now().UTC().Format(time.RFC3339),
		Type:        "default",
	}}
	encoded, err := marshalAnnouncements(items)
	if err != nil {
		return nil, err
	}
	if err := store.SetSetting(ctx, KeyAnnouncements, encoded); err != nil {
		return nil, err
	}
	_ = store.SetSetting(ctx, KeyNotice, "")
	return items, nil
}

// maybeSeedDemoAnnouncements inserts the opening/Zalo sample once when the list is empty.
func maybeSeedDemoAnnouncements(ctx context.Context, store SettingsStore) ([]Announcement, error) {
	seeded, err := store.GetSetting(ctx, KeyAnnouncementsDemoSeed)
	if err != nil {
		return nil, err
	}
	if seeded == "true" {
		return []Announcement{}, nil
	}

	name, err := store.GetSetting(ctx, KeySystemName)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		name = DefaultSystemName
	}

	items := defaultAnnouncements(name)
	encoded, err := marshalAnnouncements(items)
	if err != nil {
		return nil, err
	}
	if err := store.SetSetting(ctx, KeyAnnouncements, encoded); err != nil {
		return nil, err
	}
	if err := store.SetSetting(ctx, KeyNoticeEnabled, "true"); err != nil {
		return nil, err
	}
	if err := store.SetSetting(ctx, KeyAnnouncementsDemoSeed, "true"); err != nil {
		return nil, err
	}
	return items, nil
}