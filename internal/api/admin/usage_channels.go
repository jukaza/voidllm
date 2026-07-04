package admin

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
)

type channelTopologyNode struct {
	ChannelID   string `json:"channel_id"`
	Label       string `json:"label"`
	Provider    string `json:"provider,omitempty"`
	ChannelType string `json:"channel_type,omitempty"`
}

type channelTopologyProduct struct {
	Name     string                `json:"name"`
	Provider string                `json:"provider,omitempty"`
	Channels []channelTopologyNode `json:"channels"`
}

// UsageChannels handles GET /api/v1/usage/channels — admin channel breakdown with nested models.
func (h *Handler) UsageChannels(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "system admin access required")
	}

	from, to, ok := parseUsageRange(c)
	if !ok {
		return nil
	}

	channels, totals, err := h.DB.GetChannelUsageStats(c.Context(), from, to)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "usage channels", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to retrieve channel usage")
	}

	topology := h.buildChannelTopology()

	return c.JSON(fiber.Map{
		"from":     from.UTC().Format(time.RFC3339),
		"to":       to.UTC().Format(time.RFC3339),
		"totals":   totals,
		"channels": channels,
		"topology": topology,
	})
}

func (h *Handler) buildChannelTopology() []channelTopologyProduct {
	if h.Registry == nil {
		return nil
	}

	type nodeKey struct {
		id, label, provider, channelType string
	}
	productChannels := make(map[string]map[nodeKey]struct{})

	for _, m := range h.Registry.AllModels() {
		key := m.Name
		if productChannels[key] == nil {
			productChannels[key] = make(map[nodeKey]struct{})
		}
		for _, dep := range m.Deployments {
			if dep.ID == "" && dep.Name == "" {
				continue
			}
			id := dep.ID
			if id == "" {
				id = dep.Name
			}
			label := m.Name + "/" + dep.Name
			nk := nodeKey{id: id, label: label, provider: dep.Provider, channelType: "deployment"}
			productChannels[key][nk] = struct{}{}
		}
		for _, step := range m.RouteSteps {
			nk := nodeKey{
				id:          step.ProviderID,
				label:       step.Provider + "/" + step.UpstreamModel,
				provider:    step.Provider,
				channelType: "connection",
			}
			productChannels[key][nk] = struct{}{}
		}
	}

	result := make([]channelTopologyProduct, 0, len(productChannels))
	for name, nodes := range productChannels {
		if len(nodes) == 0 {
			continue
		}
		prod := channelTopologyProduct{Name: name}
		for nk := range nodes {
			prod.Channels = append(prod.Channels, channelTopologyNode{
				ChannelID:   nk.id,
				Label:       nk.label,
				Provider:    nk.provider,
				ChannelType: nk.channelType,
			})
		}
		result = append(result, prod)
	}

	// Sort products by name for stable UI.
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Name < result[i].Name {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}