package proxy

import (
	"strings"

	"github.com/jukaza/tavo/internal/tokens"
)

func fillEstimatedUsage(enabled bool, ui UsageInfo, requestBody []byte, responseText, estimateModel string) UsageInfo {
	if !enabled || estimateModel == "" {
		return ui
	}
	needPrompt := ui.PromptTokens == 0
	needCompletion := ui.CompletionTokens == 0
	if !needPrompt && !needCompletion {
		return ui
	}

	estimated := false
	if needPrompt {
		inputText := tokens.InputTextFromRequest(requestBody)
		if n := tokens.EstimateByModel(estimateModel, inputText); n > 0 {
			ui.PromptTokens = n
			estimated = true
		}
	}
	if needCompletion && responseText != "" {
		if n := tokens.EstimateByModel(estimateModel, responseText); n > 0 {
			ui.CompletionTokens = n
			estimated = true
		}
	}
	if estimated {
		ui.TotalTokens = ui.PromptTokens + ui.CompletionTokens
		ui.Estimated = true
	}
	return ui
}

func estimateModelName(model Model, requestedModelName string) string {
	if model.UpstreamModel != "" {
		return model.UpstreamModel
	}
	if requestedModelName != "" {
		return requestedModelName
	}
	return model.Name
}

type streamOutputAccumulator struct {
	content strings.Builder
}

func (s *streamOutputAccumulator) observe(line []byte) {
	tokens.AppendStreamDeltaContent(line, &s.content)
}

func (s *streamOutputAccumulator) text() string {
	return s.content.String()
}