package email

// Settings keys stored in the settings table.
const (
	KeySMTPEnabled    = "email.smtp.enabled"
	KeySMTPHost       = "email.smtp.host"
	KeySMTPPort       = "email.smtp.port"
	KeySMTPUsername   = "email.smtp.username"
	KeySMTPPassword   = "email.smtp.password"
	KeySMTPFrom       = "email.smtp.from"
	KeySMTPSSLEnabled = "email.smtp.ssl_enabled"
)

// Gmail SMTP defaults per Google documentation:
// https://support.google.com/a/answer/176600 — smtp.gmail.com, port 587 (STARTTLS) or 465 (SSL).
const (
	DefaultSMTPHost       = "smtp.gmail.com"
	DefaultSMTPPort       = 587
	DefaultSMTPSSLEnabled = false
)