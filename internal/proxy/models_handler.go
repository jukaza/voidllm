package proxy

import (
	"github.com/gofiber/fiber/v3"
)

// modelEntry is the OpenAI-compatible representation of a single model as
// returned by GET /v1/models.
type modelEntry struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	OwnedBy string   `json:"owned_by"`
	Aliases []string `json:"aliases,omitempty"`
}

// modelsResponse is the top-level envelope for the GET /v1/models response,
// matching the OpenAI models list format.
type modelsResponse struct {
	Object string       `json:"object"`
	Data   []modelEntry `json:"data"`
}

// ModelsHandler handles GET /v1/models and returns all active models in the
// registry in an OpenAI-compatible list format. Sensitive fields (APIKey,
// BaseURL) are never included in the response.
func (p *ProxyHandler) ModelsHandler(c fiber.Ctx) error {
	allModels := p.Registry.ListInfo()

	data := make([]modelEntry, len(allModels))
	for i, m := range allModels {
		data[i] = modelEntry{
			ID:      m.Name,
			Object:  "model",
			Created: 0,
			OwnedBy: "voidllm",
			Aliases: m.Aliases,
		}
	}

	return c.JSON(modelsResponse{Object: "list", Data: data})
}
