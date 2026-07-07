package admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/pkg/keygen"
)

// HandleCliSetup generates a Bash or PowerShell script that writes local CLI/IDE config.
// GET /api/v1/public/llm-setup
//
// Query params:
//
//	tool              — claude, codex, cline, opencode, openclaw, hermes, kilo, opencode, jcode, ...
//	key               — API key (required)
//	serverUrl         — base URL without /v1 (optional; inferred from request)
//	os                — "windows" for PowerShell, default Bash
//	model             — default model ID
//	models            — comma-separated model IDs (Kilo multi-model)
//	provider          — provider slug in generated config (default: tavo)
//	subagentModel     — Codex subagent model
//	haiku, sonnet, opus — Claude Code model mappings
//	telegramBotToken  — Telegram bot token (Hermes / OpenClaw)
//	telegramUserId    — numeric Telegram user ID (Hermes / OpenClaw)
func (h *Handler) HandleCliSetup(c fiber.Ctx) error {
	key := c.Query("key")
	tool := c.Query("tool")
	osType := c.Query("os")

	if osType == "" {
		ua := strings.ToLower(c.Get("User-Agent"))
		if strings.Contains(ua, "windows") || strings.Contains(ua, "powershell") ||
			strings.Contains(ua, "win64") || strings.Contains(ua, "win32") {
			osType = "windows"
		}
	}

	if key == "" {
		return setupErrorScript(c, osType, "API key is required (?key=sk-...)")
	}
	if _, err := keygen.ValidatePrefix(key); err != nil {
		return setupErrorScript(c, osType, "invalid API key format")
	}

	hash := keygen.Hash(key, h.HMACSecret)
	keyInfo, ok := h.KeyCache.Get(hash)
	if !ok && h.DB != nil {
		record, err := h.DB.LookupActiveKeyByHash(c.Context(), hash)
		if err == nil {
			keyInfo = auth.KeyInfo{
				ID:     record.ID,
				Status: record.Status,
			}
			h.KeyCache.Set(hash, keyInfo)
			ok = true
		}
	}
	if !ok {
		return setupErrorScript(c, osType, "invalid or expired API key")
	}
	switch keyInfo.Status {
	case "disabled", "expired", "quota_exhausted":
		return setupErrorScript(c, osType, "API key is disabled")
	}

	serverURL := strings.TrimSuffix(c.Query("serverUrl"), "/")
	if serverURL == "" {
		scheme := "http"
		if c.Protocol() == "https" {
			scheme = "https"
		}
		host := c.Get("X-Forwarded-Host")
		if host == "" {
			host = c.Hostname()
		}
		serverURL = fmt.Sprintf("%s://%s", scheme, host)
	}

	baseWithV1 := serverURL + "/v1"
	baseNoV1 := serverURL

	provider := c.Query("provider")
	if provider == "" {
		provider = "tavo"
	}

	mainModel := c.Query("model")
	if mainModel == "" {
		mainModel = "gpt-4o"
	}
	subagentModel := c.Query("subagentModel")
	if subagentModel == "" {
		subagentModel = mainModel
	}
	haiku := defaultIfEmpty(c.Query("haiku"), "claude-haiku-4-5")
	sonnet := defaultIfEmpty(c.Query("sonnet"), "claude-sonnet-4-5")
	opus := defaultIfEmpty(c.Query("opus"), "claude-opus-4-5")

	models := parseModelsList(c.Query("models"), mainModel)
	tgToken := c.Query("telegramBotToken")
	tgUserID := strings.TrimSpace(c.Query("telegramUserId"))

	isWin := strings.EqualFold(osType, "windows")
	var script string
	if isWin {
		script = generateCliSetupPowerShell(tool, key, provider, baseWithV1, baseNoV1, mainModel, models, subagentModel, haiku, sonnet, opus, tgToken, tgUserID)
	} else {
		script = generateCliSetupBash(tool, key, provider, baseWithV1, baseNoV1, mainModel, models, subagentModel, haiku, sonnet, opus, tgToken, tgUserID)
	}

	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.Status(http.StatusOK).SendString(script)
}

func defaultIfEmpty(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}

func parseModelsList(raw, fallback string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{fallback}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{fallback}
	}
	return out
}

func setupErrorScript(c fiber.Ctx, osType, msg string) error {
	c.Set("Content-Type", "text/plain; charset=utf-8")
	if strings.EqualFold(osType, "windows") {
		return c.Status(http.StatusBadRequest).SendString(fmt.Sprintf("Write-Error \"[tavo] %s\"\nExit 1", msg))
	}
	return c.Status(http.StatusBadRequest).SendString(fmt.Sprintf("echo \"[tavo] Error: %s\"\nexit 1", msg))
}

func kiloModelsJSON(models []string) string {
	var b strings.Builder
	b.WriteString("{\n")
	for i, m := range models {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString(fmt.Sprintf("        \"%s\": {\"name\": \"%s\"}", m, m))
	}
	b.WriteString("\n      }")
	return b.String()
}

func generateCliSetupBash(tool, key, provider, baseWithV1, baseNoV1, mainModel string, models []string, subagentModel, haiku, sonnet, opus, tgToken, tgUserID string) string {
	var sb strings.Builder
	sb.WriteString("#!/bin/bash\nset -e\n\n")

	switch tool {
	case "claude":
		sb.WriteString("mkdir -p \"$HOME/.claude\"\n")
		sb.WriteString("cat << 'EOF' > \"$HOME/.claude/settings.json\"\n")
		sb.WriteString(fmt.Sprintf(`{
  "hasCompletedOnboarding": true,
  "env": {
    "ANTHROPIC_BASE_URL": "%s",
    "ANTHROPIC_AUTH_TOKEN": "%s",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "%s",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "%s",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "%s"
  }
}
`, baseNoV1, key, haiku, sonnet, opus))
		sb.WriteString("EOF\n")
		sb.WriteString("echo \"[tavo] Claude Code configured.\"\n")

	case "codex":
		sb.WriteString("mkdir -p \"$HOME/.codex\"\n")
		sb.WriteString(fmt.Sprintf(`cat << 'EOF' > "$HOME/.codex/config.toml"
model = "%s"
model_provider = "%s"

[model_providers.%s]
name = "%s"
base_url = "%s"
wire_api = "responses"

[agents.subagent]
model = "%s"
EOF

cat << 'EOF' > "$HOME/.codex/auth.json"
{
  "OPENAI_API_KEY": "%s",
  "auth_mode": "apikey"
}
EOF
`, mainModel, provider, provider, provider, baseWithV1, subagentModel, key))
		sb.WriteString("echo \"[tavo] Codex CLI configured.\"\n")

	case "cline":
		sb.WriteString("mkdir -p \"$HOME/.cline/data\"\n")
		sb.WriteString(fmt.Sprintf(`cat << 'EOF' > "$HOME/.cline/data/globalState.json"
{
  "actModeApiProvider": "openai",
  "planModeApiProvider": "openai",
  "openAiBaseUrl": "%s",
  "openAiModelId": "%s",
  "planModeOpenAiModelId": "%s"
}
EOF

cat << 'EOF' > "$HOME/.cline/data/secrets.json"
{
  "openAiApiKey": "%s"
}
EOF
`, baseNoV1, mainModel, mainModel, key))
		sb.WriteString("echo \"[tavo] Cline configured.\"\n")

	case "kilo":
		sb.WriteString("mkdir -p \"$HOME/.config/kilo\"\n")
		sb.WriteString(fmt.Sprintf(`cat << 'EOF' > "$HOME/.config/kilo/kilo.jsonc"
{
  "$schema": "https://app.kilo.ai/config.json",
  "enabled_providers": ["%s"],
  "provider": {
    "%s": {
      "api": "openai",
      "options": {
        "apiKey": "%s",
        "baseURL": "%s"
      },
      "models": %s
    }
  },
  "model": "%s/%s"
}
EOF
`, provider, provider, key, baseWithV1, kiloModelsJSON(models), provider, mainModel))
		sb.WriteString("echo \"[tavo] Kilo Code configured.\"\n")

	case "hermes":
		sb.WriteString("mkdir -p \"$HOME/.hermes\"\n")
		sb.WriteString(fmt.Sprintf(`cat << 'EOF' > "$HOME/.hermes/config.yaml"
custom_providers:
  - name: tavo
    base_url: "%s"
    api_key: "%s"
    model: "%s"

model:
  default: "%s"
  provider: "custom:tavo"
  base_url: "%s"
EOF
`, baseWithV1, key, mainModel, mainModel, baseWithV1))
		sb.WriteString("cat << 'EOF' > \"$HOME/.hermes/.env\"\n")
		sb.WriteString(fmt.Sprintf("OPENAI_API_KEY=%s\n", key))
		if tgToken != "" {
			sb.WriteString(fmt.Sprintf("TELEGRAM_BOT_TOKEN=%s\n", tgToken))
		}
		if tgUserID != "" {
			sb.WriteString(fmt.Sprintf("TELEGRAM_ALLOWED_USERS=%s\n", tgUserID))
		}
		sb.WriteString("EOF\n")
		sb.WriteString("echo \"[tavo] Hermes Agent configured.\"\n")
		if tgToken != "" && tgUserID != "" {
			sb.WriteString("echo \"[tavo] Run: hermes gateway\"\n")
		}

	case "openclaw":
		sb.WriteString("mkdir -p \"$HOME/.openclaw\"\n")
		tgBlock := ""
		if tgToken != "" {
			allowFrom := "[]"
			if tgUserID != "" {
				allowFrom = fmt.Sprintf("[\"%s\"]", tgUserID)
			}
			tgBlock = fmt.Sprintf(`,
  "channels": {
    "telegram": {
      "enabled": true,
      "botToken": "%s",
      "dmPolicy": "allowlist",
      "allowFrom": %s
    }
  }`, tgToken, allowFrom)
		}
		sb.WriteString(fmt.Sprintf(`cat << 'EOF' > "$HOME/.openclaw/openclaw.json"
{
  "api_base": "%s",
  "api_key": "%s",
  "model": "%s"%s
}
EOF
`, baseWithV1, key, mainModel, tgBlock))
		sb.WriteString("echo \"[tavo] Open Claw configured.\"\n")
		if tgToken != "" {
			sb.WriteString("echo \"[tavo] Run: openclaw gateway\"\n")
		}

	case "opencode":
		sb.WriteString("mkdir -p \"$HOME/.config/opencode\"\n")
		sb.WriteString(fmt.Sprintf(`cat << 'EOF' > "$HOME/.config/opencode/opencode.json"
{
  "selected_provider": "%s",
  "provider": {
    "%s": {
      "npm": "@ai-sdk/openai-compatible",
      "options": {
        "baseURL": "%s",
        "apiKey": "%s"
      },
      "models": {
        "%s": {}
      }
    }
  },
  "default_model": "%s/%s"
}
EOF
`, provider, provider, baseWithV1, key, mainModel, provider, mainModel))
		sb.WriteString("echo \"[tavo] OpenCode configured.\"\n")

	case "jcode":
		sb.WriteString("mkdir -p \"$HOME/.jcode\" \"$HOME/.config/jcode\"\n")
		sb.WriteString(fmt.Sprintf(`cat << 'EOF' > "$HOME/.jcode/config.toml"
[model_providers.%s]
api_key = "%s"
base_url = "%s"
EOF

cat << 'EOF' > "$HOME/.config/jcode/provider-%s.env"
OPENAI_API_KEY=%s
OPENAI_API_BASE=%s
EOF
`, provider, key, baseWithV1, provider, key, baseWithV1))
		sb.WriteString("echo \"[tavo] jcode configured.\"\n")

	default:
		sb.WriteString(fmt.Sprintf("echo \"[tavo] Unsupported tool: %s\"\nexit 1\n", tool))
	}

	return sb.String()
}

func generateCliSetupPowerShell(tool, key, provider, baseWithV1, baseNoV1, mainModel string, models []string, subagentModel, haiku, sonnet, opus, tgToken, tgUserID string) string {
	var sb strings.Builder
	sb.WriteString("$ErrorActionPreference = \"Stop\"\n\n")

	switch tool {
	case "claude":
		sb.WriteString("$dir = Join-Path $HOME \".claude\"\n")
		sb.WriteString("if (!(Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }\n")
		sb.WriteString(fmt.Sprintf(`$json = @"
{
  "hasCompletedOnboarding": true,
  "env": {
    "ANTHROPIC_BASE_URL": "%s",
    "ANTHROPIC_AUTH_TOKEN": "%s",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "%s",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "%s",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "%s"
  }
}
"@
Set-Content -Path (Join-Path $dir "settings.json") -Value $json -Force
`, baseNoV1, key, haiku, sonnet, opus))
		sb.WriteString("Write-Host \"[tavo] Claude Code configured.\"\n")

	case "kilo":
		sb.WriteString("$dir = Join-Path $HOME \".config\\kilo\"\n")
		sb.WriteString("if (!(Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }\n")
		sb.WriteString(fmt.Sprintf(`$json = @"
{
  "$schema": "https://app.kilo.ai/config.json",
  "enabled_providers": ["%s"],
  "provider": {
    "%s": {
      "api": "openai",
      "options": {
        "apiKey": "%s",
        "baseURL": "%s"
      },
      "models": %s
    }
  },
  "model": "%s/%s"
}
"@
Set-Content -Path (Join-Path $dir "kilo.jsonc") -Value $json -Force
`, provider, provider, key, baseWithV1, kiloModelsJSON(models), provider, mainModel))
		sb.WriteString("Write-Host \"[tavo] Kilo Code configured.\"\n")

	case "hermes":
		sb.WriteString("$dir = Join-Path $HOME \".hermes\"\n")
		sb.WriteString("if (!(Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }\n")
		sb.WriteString(fmt.Sprintf(`$yaml = @"
model:
  default: "%s"
  provider: "custom"
  base_url: "%s"
"@
Set-Content -Path (Join-Path $dir "config.yaml") -Value $yaml -Force
`, mainModel, baseWithV1))
		envLines := fmt.Sprintf("OPENAI_API_KEY=%s`n", key)
		if tgToken != "" {
			envLines += fmt.Sprintf("TELEGRAM_BOT_TOKEN=%s`n", tgToken)
		}
		if tgUserID != "" {
			envLines += fmt.Sprintf("TELEGRAM_ALLOWED_USERS=%s`n", tgUserID)
		}
		sb.WriteString(fmt.Sprintf(`$envContent = @"
%s"@
Set-Content -Path (Join-Path $dir ".env") -Value $envContent -Force
`, envLines))
		sb.WriteString("Write-Host \"[tavo] Hermes Agent configured.\"\n")

	case "openclaw":
		sb.WriteString("$dir = Join-Path $HOME \".openclaw\"\n")
		sb.WriteString("if (!(Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }\n")
		tgBlock := ""
		if tgToken != "" {
			allowFrom := "[]"
			if tgUserID != "" {
				allowFrom = fmt.Sprintf("[\"%s\"]", tgUserID)
			}
			tgBlock = fmt.Sprintf(`,
  "channels": {
    "telegram": {
      "enabled": true,
      "botToken": "%s",
      "dmPolicy": "allowlist",
      "allowFrom": %s
    }
  }`, tgToken, allowFrom)
		}
		sb.WriteString(fmt.Sprintf(`$json = @"
{
  "api_base": "%s",
  "api_key": "%s",
  "model": "%s"%s
}
"@
Set-Content -Path (Join-Path $dir "openclaw.json") -Value $json -Force
`, baseWithV1, key, mainModel, tgBlock))
		sb.WriteString("Write-Host \"[tavo] Open Claw configured.\"\n")

	default:
		// Fall back to bash-oriented tools on Windows with a hint.
		sb.WriteString(fmt.Sprintf("Write-Error \"[tavo] Tool '%s' — use WSL or copy manual config from the dashboard.\"\nExit 1\n", tool))
	}

	return sb.String()
}