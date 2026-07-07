package admin_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestCliSetupRequiresKey(t *testing.T) {
	app, _, _ := setupTestApp(t, ":memory:")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/llm-setup?tool=claude", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "API key is required") {
		t.Fatalf("body = %q", body)
	}
}

func TestCliSetupRejectsInvalidKey(t *testing.T) {
	app, _, _ := setupTestApp(t, ":memory:")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/llm-setup?tool=claude&key=not-a-real-key", nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestCliSetupClaudeBash(t *testing.T) {
	app, _, keyCache := setupTestApp(t, ":memory:")
	key := addTestKey(t, keyCache, "user")

	q := url.Values{
		"tool":      {"claude"},
		"key":       {key},
		"serverUrl": {"https://api.example.com"},
		"haiku":     {"claude-haiku-test"},
		"sonnet":    {"claude-sonnet-test"},
		"opus":      {"claude-opus-test"},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/llm-setup?"+q.Encode(), nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	script := string(body)
	for _, want := range []string{
		"#!/bin/bash",
		"$HOME/.claude/settings.json",
		"https://api.example.com",
		key,
		"claude-haiku-test",
		"claude-sonnet-test",
		"claude-opus-test",
		"[tavo] Claude Code configured.",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q", want)
		}
	}
}

func TestCliSetupKiloMultiModel(t *testing.T) {
	app, _, keyCache := setupTestApp(t, ":memory:")
	key := addTestKey(t, keyCache, "user")

	q := url.Values{
		"tool":      {"kilo"},
		"key":       {key},
		"serverUrl": {"https://api.example.com"},
		"model":     {"gpt-4o"},
		"models":    {"gpt-4o,claude-sonnet,gpt-4o-mini"},
		"provider":  {"tavo"},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/llm-setup?"+q.Encode(), nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	script := string(body)
	for _, want := range []string{
		"kilo.jsonc",
		`"gpt-4o": {"name": "gpt-4o"}`,
		`"claude-sonnet": {"name": "claude-sonnet"}`,
		`"gpt-4o-mini": {"name": "gpt-4o-mini"}`,
		"tavo/gpt-4o",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q", want)
		}
	}
}

func TestCliSetupHermesTelegram(t *testing.T) {
	app, _, keyCache := setupTestApp(t, ":memory:")
	key := addTestKey(t, keyCache, "user")

	q := url.Values{
		"tool":              {"hermes"},
		"key":               {key},
		"serverUrl":         {"https://api.example.com"},
		"model":             {"gpt-4o"},
		"telegramBotToken":  {"123:ABC"},
		"telegramUserId":    {"987654321"},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/llm-setup?"+q.Encode(), nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	script := string(body)
	for _, want := range []string{
		"custom_providers:",
		"provider: \"custom:tavo\"",
		"api_key:",
		"TELEGRAM_BOT_TOKEN=123:ABC",
		"TELEGRAM_ALLOWED_USERS=987654321",
		"hermes gateway",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q", want)
		}
	}
}

func TestCliSetupOpenClawTelegram(t *testing.T) {
	app, _, keyCache := setupTestApp(t, ":memory:")
	key := addTestKey(t, keyCache, "user")

	q := url.Values{
		"tool":             {"openclaw"},
		"key":              {key},
		"serverUrl":        {"https://api.example.com"},
		"telegramBotToken": {"123:ABC"},
		"telegramUserId":   {"987654321"},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/llm-setup?"+q.Encode(), nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	script := string(body)
	for _, want := range []string{
		`"botToken": "123:ABC"`,
		`"allowFrom": ["987654321"]`,
		"openclaw gateway",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q", want)
		}
	}
}

func TestCliSetupWindowsPowerShell(t *testing.T) {
	app, _, keyCache := setupTestApp(t, ":memory:")
	key := addTestKey(t, keyCache, "user")

	q := url.Values{
		"tool":      {"claude"},
		"key":       {key},
		"serverUrl": {"https://api.example.com"},
		"os":        {"windows"},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/llm-setup?"+q.Encode(), nil)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: testTimeout})
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	script := string(body)
	if !strings.Contains(script, "$ErrorActionPreference") {
		t.Errorf("expected PowerShell script, got: %s", script[:min(120, len(script))])
	}
	if !strings.Contains(script, "Write-Host") {
		t.Error("expected Write-Host in PowerShell script")
	}
}