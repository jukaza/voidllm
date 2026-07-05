package security

const (
	KeyTurnstileEnabled   = "security.turnstile.enabled"
	KeyTurnstileSiteKey   = "security.turnstile.site_key"
	KeyTurnstileSecretKey = "security.turnstile.secret_key"

	prefixOAuth = "security.oauth."

	KeyOAuthGoogleEnabled      = prefixOAuth + "google.enabled"
	KeyOAuthGoogleAllowLogin   = prefixOAuth + "google.allow_login"
	KeyOAuthGoogleAllowSignup  = prefixOAuth + "google.allow_signup"
	KeyOAuthGoogleClientID     = prefixOAuth + "google.client_id"
	KeyOAuthGoogleClientSecret = prefixOAuth + "google.client_secret"

	KeyOAuthGitHubEnabled      = prefixOAuth + "github.enabled"
	KeyOAuthGitHubAllowLogin   = prefixOAuth + "github.allow_login"
	KeyOAuthGitHubAllowSignup  = prefixOAuth + "github.allow_signup"
	KeyOAuthGitHubClientID     = prefixOAuth + "github.client_id"
	KeyOAuthGitHubClientSecret = prefixOAuth + "github.client_secret"

	prefixTwoFA    = "security.two_fa."
	prefixSession  = "security.session."
	prefixPassword = "security.password."

	KeyTwoFAAllowUserEnable    = prefixTwoFA + "allow_user_enable"
	KeyTwoFARequireSystemAdmin = prefixTwoFA + "require_system_admin"

	KeySessionTTLHours      = prefixSession + "ttl_hours"
	KeySessionAllowMultiple = prefixSession + "allow_multiple"
	KeySessionMaxConcurrent = prefixSession + "max_concurrent"

	KeyPasswordMinLength              = prefixPassword + "min_length"
	KeyPasswordAllowOAuthSetPassword  = prefixPassword + "allow_oauth_set_password"
)