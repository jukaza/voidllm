package site

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// SettingsStore reads and writes key-value site settings.
type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	SetSettingIfNotExists(ctx context.Context, key, value string) error
}

// Config is the full site branding and legal configuration.
type Config struct {
	SystemName            string `json:"system_name"`
	Logo                  string `json:"logo"`
	ServerAddress         string `json:"server_address"`
	Footer                string `json:"footer"`
	About                 string `json:"about"`
	HomePageContent       string `json:"home_page_content"`
	UserAgreement         string         `json:"user_agreement"`
	PrivacyPolicy         string         `json:"privacy_policy"`
	Announcements         []Announcement `json:"announcements"`
	NoticeEnabled         bool           `json:"notice_enabled"`
	RegisterEnabled       bool   `json:"register_enabled"`
	UserAgreementEnabled  bool   `json:"user_agreement_enabled"`
	PrivacyPolicyEnabled  bool   `json:"privacy_policy_enabled"`
}

// UpdateInput is the admin payload for PUT /admin/settings/site.
type UpdateInput struct {
	SystemName      *string `json:"system_name"`
	Logo            *string `json:"logo"`
	ServerAddress   *string `json:"server_address"`
	Footer          *string `json:"footer"`
	About           *string `json:"about"`
	HomePageContent *string `json:"home_page_content"`
	UserAgreement   *string          `json:"user_agreement"`
	PrivacyPolicy   *string          `json:"privacy_policy"`
	Announcements   *[]Announcement  `json:"announcements"`
	NoticeEnabled   *bool            `json:"notice_enabled"`
	RegisterEnabled *bool   `json:"register_enabled"`
}

// EnsureDefaults seeds first-run site settings without overwriting operator edits.
func EnsureDefaults(ctx context.Context, store SettingsStore) error {
	name := DefaultSystemName
	announcementsJSON, err := defaultAnnouncementsJSON(name)
	if err != nil {
		return fmt.Errorf("default announcements: %w", err)
	}
	defaults := map[string]string{
		KeySystemName:      name,
		KeyLogo:            DefaultLogo,
		KeyServerAddress:   "",
		KeyFooter:          defaultFooter(name),
		KeyAbout:           "",
		KeyHomePageContent: defaultHomePageContent(name),
		KeyUserAgreement:   defaultUserAgreement(name),
		KeyPrivacyPolicy:   defaultPrivacyPolicy(name),
		KeyNotice:          "",
		KeyAnnouncements:   announcementsJSON,
		KeyNoticeEnabled:   "true",
		KeyRegisterEnabled: "true",
	}
	for key, value := range defaults {
		if err := store.SetSettingIfNotExists(ctx, key, value); err != nil {
			return fmt.Errorf("seed %s: %w", key, err)
		}
	}
	return nil
}

// Load reads the current site configuration, seeding defaults when missing.
func Load(ctx context.Context, store SettingsStore) (Config, error) {
	if err := EnsureDefaults(ctx, store); err != nil {
		return Config{}, err
	}

	noticeEnabled, err := loadBool(ctx, store, KeyNoticeEnabled, false)
	if err != nil {
		return Config{}, err
	}
	registerEnabled, err := loadBool(ctx, store, KeyRegisterEnabled, true)
	if err != nil {
		return Config{}, err
	}

	systemName, err := store.GetSetting(ctx, KeySystemName)
	if err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(systemName) == "" {
		systemName = DefaultSystemName
	}

	logo, err := store.GetSetting(ctx, KeyLogo)
	if err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(logo) == "" {
		logo = DefaultLogo
	}

	serverAddress, err := store.GetSetting(ctx, KeyServerAddress)
	if err != nil {
		return Config{}, err
	}
	footer, err := store.GetSetting(ctx, KeyFooter)
	if err != nil {
		return Config{}, err
	}
	about, err := store.GetSetting(ctx, KeyAbout)
	if err != nil {
		return Config{}, err
	}
	homePage, err := store.GetSetting(ctx, KeyHomePageContent)
	if err != nil {
		return Config{}, err
	}
	userAgreement, err := store.GetSetting(ctx, KeyUserAgreement)
	if err != nil {
		return Config{}, err
	}
	privacyPolicy, err := store.GetSetting(ctx, KeyPrivacyPolicy)
	if err != nil {
		return Config{}, err
	}
	announcements, err := loadAnnouncements(ctx, store)
	if err != nil {
		return Config{}, err
	}

	return Config{
		SystemName:           systemName,
		Logo:                 logo,
		ServerAddress:        serverAddress,
		Footer:               footer,
		About:                about,
		HomePageContent:      homePage,
		UserAgreement:        userAgreement,
		PrivacyPolicy:        privacyPolicy,
		Announcements:        announcements,
		NoticeEnabled:        noticeEnabled,
		RegisterEnabled:      registerEnabled,
		UserAgreementEnabled: strings.TrimSpace(userAgreement) != "",
		PrivacyPolicyEnabled: strings.TrimSpace(privacyPolicy) != "",
	}, nil
}

// Update applies partial admin changes and returns the saved configuration.
func Update(ctx context.Context, store SettingsStore, input UpdateInput) (Config, error) {
	if err := EnsureDefaults(ctx, store); err != nil {
		return Config{}, err
	}

	if input.SystemName != nil {
		name := strings.TrimSpace(*input.SystemName)
		if name == "" {
			return Config{}, fmt.Errorf("system_name is required")
		}
		if err := store.SetSetting(ctx, KeySystemName, name); err != nil {
			return Config{}, err
		}
	}
	if input.Logo != nil {
		if err := store.SetSetting(ctx, KeyLogo, strings.TrimSpace(*input.Logo)); err != nil {
			return Config{}, err
		}
	}
	if input.ServerAddress != nil {
		if err := store.SetSetting(ctx, KeyServerAddress, strings.TrimSpace(*input.ServerAddress)); err != nil {
			return Config{}, err
		}
	}
	if input.Footer != nil {
		if err := store.SetSetting(ctx, KeyFooter, strings.TrimSpace(*input.Footer)); err != nil {
			return Config{}, err
		}
	}
	if input.About != nil {
		if err := store.SetSetting(ctx, KeyAbout, strings.TrimSpace(*input.About)); err != nil {
			return Config{}, err
		}
	}
	if input.HomePageContent != nil {
		if err := store.SetSetting(ctx, KeyHomePageContent, strings.TrimSpace(*input.HomePageContent)); err != nil {
			return Config{}, err
		}
	}
	if input.UserAgreement != nil {
		if err := store.SetSetting(ctx, KeyUserAgreement, strings.TrimSpace(*input.UserAgreement)); err != nil {
			return Config{}, err
		}
	}
	if input.PrivacyPolicy != nil {
		if err := store.SetSetting(ctx, KeyPrivacyPolicy, strings.TrimSpace(*input.PrivacyPolicy)); err != nil {
			return Config{}, err
		}
	}
	if input.Announcements != nil {
		encoded, err := marshalAnnouncements(*input.Announcements)
		if err != nil {
			return Config{}, err
		}
		if err := store.SetSetting(ctx, KeyAnnouncements, encoded); err != nil {
			return Config{}, err
		}
		_ = store.SetSetting(ctx, KeyNotice, "")
	}
	if input.NoticeEnabled != nil {
		if err := store.SetSetting(ctx, KeyNoticeEnabled, strconv.FormatBool(*input.NoticeEnabled)); err != nil {
			return Config{}, err
		}
	}
	if input.RegisterEnabled != nil {
		if err := store.SetSetting(ctx, KeyRegisterEnabled, strconv.FormatBool(*input.RegisterEnabled)); err != nil {
			return Config{}, err
		}
	}

	return Load(ctx, store)
}

func loadBool(ctx context.Context, store SettingsStore, key string, fallback bool) (bool, error) {
	raw, err := store.GetSetting(ctx, key)
	if err != nil {
		return false, err
	}
	if raw == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback, nil
	}
	return parsed, nil
}