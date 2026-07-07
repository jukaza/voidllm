package proxy

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/jsonx"
)

const (
	responseSkinKey         = "tavo.response_skin"
	upstreamPathOverrideKey = "tavo.upstream_path_override"
)

// ResponseSkin selects the wire format written to the client.
type ResponseSkin int

const (
	SkinOpenAI ResponseSkin = iota
	SkinAnthropic
)

func setResponseSkin(c fiber.Ctx, skin ResponseSkin) {
	c.Locals(responseSkinKey, skin)
}

func responseSkinFromCtx(c fiber.Ctx) ResponseSkin {
	if v, ok := c.Locals(responseSkinKey).(ResponseSkin); ok {
		return v
	}
	return SkinOpenAI
}

type anthropicSkinRequest struct {
	Model       string                    `json:"model"`
	MaxTokens   int                       `json:"max_tokens"`
	System      jsonx.RawMessage          `json:"system,omitempty"`
	Messages    []anthropicSkinMessage    `json:"messages"`
	Tools       []anthropicToolDefinition `json:"tools,omitempty"`
	ToolChoice  jsonx.RawMessage          `json:"tool_choice,omitempty"`
	Stream      bool                      `json:"stream,omitempty"`
	Temperature *float64                  `json:"temperature,omitempty"`
	TopP        *float64                  `json:"top_p,omitempty"`
}

type anthropicSkinMessage struct {
	Role    string           `json:"role"`
	Content jsonx.RawMessage `json:"content"`
}

type anthropicSkinCountRequest struct {
	Model    string                 `json:"model"`
	Messages []anthropicSkinMessage `json:"messages"`
	System   jsonx.RawMessage       `json:"system,omitempty"`
}

type anthropicSkinErrorBody struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type anthropicSkinMessageResponse struct {
	ID           string                   `json:"id"`
	Type         string                   `json:"type"`
	Role         string                   `json:"role"`
	Model        string                   `json:"model"`
	Content      []anthropicResponseBlock `json:"content"`
	StopReason   string                   `json:"stop_reason"`
	StopSequence *string                  `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicSkinCountResponse struct {
	InputTokens int `json:"input_tokens"`
}

type anthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

func sendAnthropicError(c fiber.Ctx, status int, errType, message string) error {
	var body anthropicSkinErrorBody
	body.Type = "error"
	body.Error.Type = errType
	body.Error.Message = message
	return c.Status(status).JSON(body)
}

func anthropicRequestToOpenAI(body []byte) ([]byte, requestEnvelope, error) {
	var req anthropicSkinRequest
	if err := jsonx.Unmarshal(body, &req); err != nil {
		return nil, requestEnvelope{}, fmt.Errorf("unmarshal anthropic request: %w", err)
	}
	if req.Model == "" {
		return nil, requestEnvelope{}, fmt.Errorf("model field is required")
	}

	out := map[string]any{"model": req.Model}
	if req.MaxTokens > 0 {
		out["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		out["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		out["top_p"] = *req.TopP
	}
	if req.Stream {
		out["stream"] = true
	}

	messages := make([]map[string]any, 0, len(req.Messages)+1)
	if len(req.System) > 0 {
		if system := anthropicSystemToOpenAI(req.System); system != "" {
			messages = append(messages, map[string]any{"role": "system", "content": system})
		}
	}

	for _, msg := range req.Messages {
		converted, err := anthropicMessageToOpenAI(msg)
		if err != nil {
			return nil, requestEnvelope{}, err
		}
		if converted == nil {
			continue
		}
		if slice, ok := converted.([]map[string]any); ok {
			messages = append(messages, slice...)
			continue
		}
		if one, ok := converted.(map[string]any); ok {
			messages = append(messages, one)
		}
	}

	if len(messages) == 0 {
		return nil, requestEnvelope{}, fmt.Errorf("messages field is required")
	}
	out["messages"] = messages

	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, tool := range req.Tools {
			schema := tool.InputSchema
			if schema == nil {
				schema = jsonx.RawMessage(`{"type":"object","properties":{}}`)
			}
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  schema,
				},
			})
		}
		out["tools"] = tools
	}

	if len(req.ToolChoice) > 0 {
		choice, err := anthropicToolChoiceToOpenAI(req.ToolChoice)
		if err != nil {
			return nil, requestEnvelope{}, err
		}
		if choice != nil {
			out["tool_choice"] = choice
		}
	}

	openaiBody, err := jsonx.Marshal(out)
	if err != nil {
		return nil, requestEnvelope{}, fmt.Errorf("marshal openai request: %w", err)
	}

	return openaiBody, requestEnvelope{Model: req.Model, Stream: req.Stream}, nil
}

func anthropicSystemToOpenAI(raw jsonx.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	if trimmed[0] == '"' {
		var s string
		if err := jsonx.Unmarshal(raw, &s); err == nil {
			return s
		}
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := jsonx.Unmarshal(raw, &blocks); err == nil {
		parts := make([]string, 0, len(blocks))
		for _, b := range blocks {
			if b.Type == "" || b.Type == "text" {
				if b.Text != "" {
					parts = append(parts, b.Text)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func anthropicMessageToOpenAI(msg anthropicSkinMessage) (any, error) {
	role := msg.Role
	switch role {
	case "user", "tool":
		role = "user"
	case "assistant":
		role = "assistant"
	case "system":
		if text := anthropicMessageContentToText(msg.Content); text != "" {
			return map[string]any{
				"role":    "user",
				"content": "<system-reminder>\n" + text + "\n</system-reminder>",
			}, nil
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported message role %q", msg.Role)
	}

	trimmed := strings.TrimSpace(string(msg.Content))
	if trimmed == "" {
		return map[string]any{"role": role, "content": ""}, nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := jsonx.Unmarshal(msg.Content, &s); err == nil {
			return map[string]any{"role": role, "content": s}, nil
		}
	}

	var blocks []struct {
		Type      string                `json:"type"`
		Text      string                `json:"text,omitempty"`
		ID        string                `json:"id,omitempty"`
		Name      string                `json:"name,omitempty"`
		Input     jsonx.RawMessage      `json:"input,omitempty"`
		ToolUseID string                `json:"tool_use_id,omitempty"`
		Content   jsonx.RawMessage      `json:"content,omitempty"`
		Source    *anthropicImageSource `json:"source,omitempty"`
	}
	if err := jsonx.Unmarshal(msg.Content, &blocks); err != nil {
		return nil, fmt.Errorf("unmarshal message content: %w", err)
	}

	parts := make([]map[string]any, 0, len(blocks))
	toolCalls := make([]map[string]any, 0)
	toolResults := make([]map[string]any, 0)

	for _, block := range blocks {
		switch block.Type {
		case "text", "":
			parts = append(parts, map[string]any{"type": "text", "text": block.Text})
		case "image":
			if block.Source != nil && block.Source.Type == "base64" {
				parts = append(parts, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": fmt.Sprintf("data:%s;base64,%s", block.Source.MediaType, block.Source.Data),
					},
				})
			}
		case "tool_use":
			args := "{}"
			if len(block.Input) > 0 {
				args = string(block.Input)
			}
			toolCalls = append(toolCalls, map[string]any{
				"id": block.ID, "type": "function",
				"function": map[string]any{"name": block.Name, "arguments": args},
			})
		case "tool_result":
			content := ""
			if len(block.Content) > 0 {
				trim := strings.TrimSpace(string(block.Content))
				if trim != "" && trim[0] == '"' {
					var s string
					if err := jsonx.Unmarshal(block.Content, &s); err == nil {
						content = s
					}
				} else {
					var resultBlocks []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					}
					if err := jsonx.Unmarshal(block.Content, &resultBlocks); err == nil {
						texts := make([]string, 0, len(resultBlocks))
						for _, rb := range resultBlocks {
							if rb.Text != "" {
								texts = append(texts, rb.Text)
							}
						}
						content = strings.Join(texts, "\n")
					} else {
						content = string(block.Content)
					}
				}
			}
			toolResults = append(toolResults, map[string]any{
				"role": "tool", "tool_call_id": block.ToolUseID, "content": content,
			})
		}
	}

	if len(toolResults) > 0 {
		out := make([]map[string]any, 0, len(toolResults)+1)
		out = append(out, toolResults...)
		if len(parts) > 0 {
			out = append(out, map[string]any{"role": "user", "content": collapseAnthropicTextParts(parts)})
		}
		return out, nil
	}

	if len(toolCalls) > 0 {
		result := map[string]any{"role": "assistant", "tool_calls": toolCalls}
		if len(parts) > 0 {
			result["content"] = collapseAnthropicTextParts(parts)
		}
		return result, nil
	}

	if len(parts) == 0 {
		return map[string]any{"role": role, "content": ""}, nil
	}
	if len(parts) == 1 && parts[0]["type"] == "text" {
		return map[string]any{"role": role, "content": parts[0]["text"]}, nil
	}
	return map[string]any{"role": role, "content": parts}, nil
}

func collapseAnthropicTextParts(parts []map[string]any) any {
	if len(parts) == 1 {
		if parts[0]["type"] == "text" {
			return parts[0]["text"]
		}
	}
	return parts
}

func anthropicMessageContentToText(raw jsonx.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return ""
	}
	if trimmed[0] == '"' {
		var s string
		if err := jsonx.Unmarshal(raw, &s); err == nil {
			return s
		}
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := jsonx.Unmarshal(raw, &blocks); err == nil {
		texts := make([]string, 0, len(blocks))
		for _, b := range blocks {
			if b.Text != "" {
				texts = append(texts, b.Text)
			}
		}
		return strings.Join(texts, "\n")
	}
	return ""
}

func anthropicToolChoiceToOpenAI(raw jsonx.RawMessage) (any, error) {
	var s string
	if err := jsonx.Unmarshal(raw, &s); err == nil {
		switch s {
		case "auto":
			return "auto", nil
		case "any":
			return "required", nil
		case "none":
			return "none", nil
		default:
			return "auto", nil
		}
	}
	var obj struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := jsonx.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("unmarshal tool_choice: %w", err)
	}
	switch obj.Type {
	case "auto":
		return "auto", nil
	case "any":
		return "required", nil
	case "tool":
		return map[string]any{
			"type": "function",
			"function": map[string]any{"name": obj.Name},
		}, nil
	default:
		return "auto", nil
	}
}

func openAIResponseToAnthropic(body []byte, model string) ([]byte, error) {
	var oai openAIResponse
	if err := jsonx.Unmarshal(body, &oai); err != nil {
		return nil, fmt.Errorf("unmarshal openai response: %w", err)
	}
	if len(oai.Choices) == 0 {
		return nil, fmt.Errorf("openai response has no choices")
	}

	choice := oai.Choices[0]
	content := make([]anthropicResponseBlock, 0)

	if choice.Message.Content != nil && *choice.Message.Content != "" {
		content = append(content, anthropicResponseBlock{Type: "text", Text: *choice.Message.Content})
	}
	for _, tc := range choice.Message.ToolCalls {
		input := jsonx.RawMessage("{}")
		if tc.Function.Arguments != "" {
			input = jsonx.RawMessage(tc.Function.Arguments)
		}
		content = append(content, anthropicResponseBlock{
			Type: "tool_use", ID: tc.ID, Name: tc.Function.Name, Input: input,
		})
	}
	if len(content) == 0 {
		content = append(content, anthropicResponseBlock{Type: "text", Text: ""})
	}

	respModel := oai.Model
	if respModel == "" {
		respModel = model
	}

	inputTokens := oai.Usage.PromptTokens
	if oai.Usage.PromptTokensDetails != nil {
		inputTokens -= oai.Usage.PromptTokensDetails.CachedTokens
	}
	inputTokens -= oai.Usage.CacheCreationInputTokens
	if inputTokens < 0 {
		inputTokens = 0
	}

	resp := anthropicSkinMessageResponse{
		ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Type: "message",
		Role: "assistant", Model: respModel, Content: content,
		StopReason: mapFinishReasonToAnthropic(choice.FinishReason),
		Usage: struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		}{InputTokens: inputTokens, OutputTokens: oai.Usage.CompletionTokens},
	}

	return jsonx.Marshal(resp)
}

func mapFinishReasonToAnthropic(reason string) string {
	switch reason {
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return "end_turn"
	}
}

func anthropicCountInputText(req anthropicSkinCountRequest) string {
	var b strings.Builder
	if system := anthropicSystemToOpenAI(req.System); system != "" {
		b.WriteString(system)
		b.WriteByte('\n')
	}
	for _, msg := range req.Messages {
		b.WriteString(anthropicMessageContentToText(msg.Content))
		b.WriteByte('\n')
	}
	return b.String()
}