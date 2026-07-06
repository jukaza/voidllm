package site

// Settings keys stored in the settings table.
const (
	KeySystemName      = "site.system_name"
	KeyLogo            = "site.logo"
	KeyServerAddress   = "site.server_address"
	KeyFooter          = "site.footer"
	KeyAbout           = "site.about"
	KeyHomePageContent = "site.home_page_content"
	KeyUserAgreement   = "site.user_agreement"
	KeyPrivacyPolicy   = "site.privacy_policy"
	KeyNotice          = "site.notice" // legacy single notice; migrated to announcements
	KeyAnnouncements         = "site.announcements"
	KeyAnnouncementsDemoSeed = "site.announcements_demo_seeded"
	KeyNoticeEnabled         = "site.notice_enabled"
	KeyRegisterEnabled   = "site.register_enabled"
	KeySiteSubtitle      = "site.site_subtitle"
	KeySupportZalo       = "site.support_zalo"
	KeySupportTelegram   = "site.support_telegram"
	KeyDocURL            = "site.doc_url"
)

// DefaultSystemName is the product name used until an operator customizes it.
const DefaultSystemName = "Tavo"

// DefaultLogo is the bundled logo path served by the embedded UI.
const DefaultLogo = "/logo.svg"