package proxy

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/jsonx"
	"github.com/jukaza/tavo/internal/tokens"
)

// MessagesHandler handles POST /v1/messages for Anthropic-compatible clients
// such as Claude Desktop Gateway mode.
func (p *ProxyHandler) MessagesHandler(c fiber.Ctx) error {
	setResponseSkin(c, SkinAnthropic)

	maxReqBody, _, _ := p.resolveEffectiveLimits()
	body := c.Body()
	if len(body) > maxReqBody {
		return sendAnthropicError(c, fiber.StatusRequestEntityTooLarge,
			"invalid_request_error", "request body exceeds size limit")
	}

	openaiBody, envelope, err := anthropicRequestToOpenAI(body)
	if err != nil {
		return sendAnthropicError(c, fiber.StatusBadRequest,
			"invalid_request_error", err.Error())
	}

	if envelope.Stream {
		return sendAnthropicError(c, fiber.StatusBadRequest,
			"invalid_request_error", "streaming is not supported yet")
	}

	// The proxy pipeline speaks OpenAI chat/completions to upstream providers.
	c.Locals(upstreamPathOverrideKey, "chat/completions")
	return p.handleCompat(c, openaiBody, envelope, time.Now())
}

// CountTokensHandler handles POST /v1/messages/count_tokens.
func (p *ProxyHandler) CountTokensHandler(c fiber.Ctx) error {
	setResponseSkin(c, SkinAnthropic)

	maxReqBody, _, _ := p.resolveEffectiveLimits()
	body := c.Body()
	if len(body) > maxReqBody {
		return sendAnthropicError(c, fiber.StatusRequestEntityTooLarge,
			"invalid_request_error", "request body exceeds size limit")
	}

	var req anthropicSkinCountRequest
	if err := jsonx.Unmarshal(body, &req); err != nil {
		return sendAnthropicError(c, fiber.StatusBadRequest,
			"invalid_request_error", "invalid request body")
	}
	if req.Model == "" {
		return sendAnthropicError(c, fiber.StatusBadRequest,
			"invalid_request_error", "model field is required")
	}

	count := tokens.EstimateByModel(req.Model, anthropicCountInputText(req))
	return c.JSON(anthropicSkinCountResponse{InputTokens: count})
}