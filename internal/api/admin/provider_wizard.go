package admin

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/internal/jsonx"
	"github.com/voidmind-io/voidllm/internal/provider"
	"github.com/voidmind-io/voidllm/pkg/crypto"
)

// ListProviderPresets handles GET /api/v1/providers/presets (system_admin).
// Returns the setup-wizard catalog (self-hosted Ollama excluded — use Custom).
func (h *Handler) ListProviderPresets(c fiber.Ctx) error {
	out := make([]provider.Preset, 0, len(provider.Presets))
	for _, p := range provider.Presets {
		if p.ID == "ollama" {
			continue
		}
		out = append(out, p)
	}
	return c.JSON(fiber.Map{"data": out})
}

// discoverModelsRequest is the JSON body for POST /providers/discover-models.
// Either preset_id or (base_url + protocol) must be provided; explicit fields
// override preset defaults.
type discoverModelsRequest struct {
	PresetID string `json:"preset_id"`
	BaseURL  string `json:"base_url"`
	Protocol string `json:"protocol"`
	APIKey   string `json:"api_key"`
	// ProviderID, when set, reuses the stored key/base URL of an existing
	// provider instead of requiring the key to be pasted again.
	ProviderID string `json:"provider_id"`
}

// discoveredModel is one row in the discover response checklist.
type discoveredModel struct {
	ID string `json:"id"`
	// KnownCost is the preset reference cost, when the model appears in the
	// preset's DefaultCost table.
	KnownCost *provider.CostRef `json:"known_cost,omitempty"`
	// Exists is true when a product model with this name already exists —
	// import will attach a route to it instead of creating a new model.
	Exists bool `json:"exists"`
}

// discoverClient does not follow redirects (SSRF hardening, mirrors testClient).
var discoverClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// resolveDiscoverTarget merges preset defaults, stored provider fields, and
// explicit request fields into a concrete (baseURL, protocol, apiKey, preset).
func (h *Handler) resolveDiscoverTarget(ctx context.Context, req *discoverModelsRequest) (string, string, string, *provider.Preset, error) {
	var preset *provider.Preset
	if req.PresetID != "" {
		preset = provider.PresetByID(req.PresetID)
		if preset == nil {
			return "", "", "", nil, errors.New("unknown preset_id")
		}
	}

	baseURL, protocol, apiKey := req.BaseURL, req.Protocol, req.APIKey

	if req.ProviderID != "" {
		p, err := h.DB.GetProvider(ctx, req.ProviderID)
		if err != nil {
			return "", "", "", nil, errors.New("provider not found")
		}
		if baseURL == "" {
			baseURL = p.BaseURL
		}
		if protocol == "" {
			protocol = p.Protocol
		}
		if apiKey == "" && p.APIKeyEncrypted != nil {
			apiKey, err = crypto.DecryptString(*p.APIKeyEncrypted, h.EncryptionKey, providerAAD(p.ID))
			if err != nil {
				return "", "", "", nil, errors.New("stored api key cannot be decrypted")
			}
		}
	}

	if preset != nil {
		if baseURL == "" {
			baseURL = preset.BaseURL
		}
		if protocol == "" {
			protocol = preset.Protocol
		}
	}

	if baseURL == "" {
		return "", "", "", nil, errors.New("base_url is required (no preset default)")
	}
	if protocol == "" {
		protocol = "openai"
	}
	if !provider.ValidProviders[protocol] {
		return "", "", "", nil, errors.New("invalid protocol")
	}
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", "", "", nil, errors.New("base_url must begin with http:// or https://")
	}
	return baseURL, protocol, apiKey, preset, nil
}

// modelsEndpointFor returns the absolute model-listing URL for the target.
func modelsEndpointFor(baseURL, protocol string, preset *provider.Preset) string {
	base := strings.TrimRight(baseURL, "/")
	if preset != nil && preset.ModelsPath != "" {
		return base + preset.ModelsPath
	}
	switch protocol {
	case "anthropic":
		return base + "/v1/models"
	case "gemini":
		return base + "/v1beta/models"
	default:
		return base + "/models"
	}
}

// DiscoverProviderModels handles POST /api/v1/providers/discover-models
// (system_admin). It calls the upstream model-listing endpoint with the given
// credentials and returns the available model IDs as a checklist. A successful
// listing doubles as a key/connectivity test.
func (h *Handler) DiscoverProviderModels(c fiber.Ctx) error {
	ctx := c.Context()

	var req discoverModelsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	baseURL, protocol, apiKey, preset, err := h.resolveDiscoverTarget(ctx, &req)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}

	endpoint := modelsEndpointFor(baseURL, protocol, preset)

	reqCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return apierror.BadRequest(c, "invalid base_url")
	}
	if apiKey != "" {
		switch protocol {
		case "anthropic":
			httpReq.Header.Set("x-api-key", apiKey)
			httpReq.Header.Set("anthropic-version", "2023-06-01")
		case "gemini":
			httpReq.Header.Set("x-goog-api-key", apiKey)
		default:
			httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}

	resp, err := discoverClient.Do(httpReq)
	if err != nil {
		h.Log.WarnContext(ctx, "discover-models: request failed",
			slog.String("url", endpoint), slog.String("error", err.Error()))
		return c.JSON(fiber.Map{"success": false, "message": "unable to reach the provider", "data": []discoveredModel{}})
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return c.JSON(fiber.Map{"success": false, "message": "authentication failed — check the API key", "data": []discoveredModel{}})
	}
	if resp.StatusCode >= 400 {
		return c.JSON(fiber.Map{"success": false, "message": "provider returned HTTP " + resp.Status, "data": []discoveredModel{}})
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	ids := parseModelList(body, protocol)
	if len(ids) == 0 {
		return c.JSON(fiber.Map{"success": true, "message": "connected, but no models reported", "data": []discoveredModel{}})
	}

	// Mark models already in provider inventory (or global catalog when no provider).
	existing := make(map[string]bool)
	if req.ProviderID != "" {
		if inv, listErr := h.DB.ListProviderUpstreamModels(ctx, req.ProviderID, false); listErr == nil {
			for _, m := range inv {
				existing[m.UpstreamID] = true
			}
		}
	} else if names, listErr := h.DB.ListModelNames(ctx); listErr == nil {
		for _, n := range names {
			existing[n] = true
		}
	}

	out := make([]discoveredModel, 0, len(ids))
	for _, id := range ids {
		dm := discoveredModel{ID: id, Exists: existing[id]}
		if preset != nil {
			if cost, ok := preset.DefaultCost[id]; ok {
				cc := cost
				dm.KnownCost = &cc
			}
		}
		out = append(out, dm)
	}
	return c.JSON(fiber.Map{"success": true, "message": "", "data": out})
}

// parseModelList extracts model IDs from a provider model-listing response.
// OpenAI-compatible and Anthropic responses use {"data":[{"id":...}]};
// Gemini uses {"models":[{"name":"models/gemini-..."}]}.
func parseModelList(body []byte, protocol string) []string {
	if protocol == "gemini" {
		var gr struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if jsonx.Unmarshal(body, &gr) != nil {
			return nil
		}
		ids := make([]string, 0, len(gr.Models))
		for _, m := range gr.Models {
			ids = append(ids, strings.TrimPrefix(m.Name, "models/"))
		}
		return ids
	}
	var or struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if jsonx.Unmarshal(body, &or) != nil {
		return nil
	}
	ids := make([]string, 0, len(or.Data))
	for _, m := range or.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

// importProviderRequest is the JSON body for POST /providers/import.
// It creates (or reuses) a provider and, for each selected model, either
// creates a product model + route or attaches a route to the existing model.
type importProviderRequest struct {
	PresetID string `json:"preset_id"`
	// ProviderID reuses an existing provider; when empty a new one is created.
	ProviderID string `json:"provider_id"`
	// Name/Slug override preset defaults for a new provider.
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	BaseURL  string `json:"base_url"`
	Protocol string `json:"protocol"`
	APIKey   string `json:"api_key"`
	// Models is the list of upstream model IDs ticked in the wizard.
	Models []importModelSpec `json:"models"`
	// Markup multiplies reference cost to produce sell prices for models
	// created by this import. 0 = use DefaultMarkup.
	Markup float64 `json:"markup"`
	// MakePublic marks created models as storefront-visible.
	MakePublic bool `json:"make_public"`
}

// importModelSpec is one ticked model in the import request.
type importModelSpec struct {
	// UpstreamID is the model name at the provider ("deepseek-chat").
	UpstreamID string `json:"upstream_id"`
	// ProductName is the customer-facing model name. Empty = UpstreamID.
	ProductName string `json:"product_name"`
	// Cost overrides the preset reference cost for the route.
	Cost *provider.CostRef `json:"cost"`
}

// DefaultMarkup is the sell price multiplier applied when the import request
// does not specify one.
const DefaultMarkup = 1.5

// ImportProvider handles POST /api/v1/providers/import (system_admin).
// One transaction-ish wizard step: provider + models + routes + prices.
// Individual model failures are reported per-item; the rest proceed.
func (h *Handler) ImportProvider(c fiber.Ctx) error {
	ctx := c.Context()

	var req importProviderRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if len(req.Models) == 0 {
		return apierror.BadRequest(c, "models is required")
	}

	var preset *provider.Preset
	if req.PresetID != "" {
		preset = provider.PresetByID(req.PresetID)
		if preset == nil {
			return apierror.BadRequest(c, "unknown preset_id")
		}
	}

	// Resolve provider fields: request > preset.
	name, slug, baseURL, protocol := req.Name, req.Slug, req.BaseURL, req.Protocol
	logo := ""
	if preset != nil {
		if name == "" {
			name = preset.Name
		}
		if slug == "" {
			slug = preset.ID
		}
		if baseURL == "" {
			baseURL = preset.BaseURL
		}
		if protocol == "" {
			protocol = preset.Protocol
		}
		logo = preset.Logo
	}
	if protocol == "" {
		protocol = "openai"
	}
	if !provider.ValidProviders[protocol] {
		return apierror.BadRequest(c, "invalid protocol")
	}
	if req.ProviderID == "" {
		if name == "" {
			return apierror.BadRequest(c, "name is required when creating a provider")
		}
		if baseURL == "" {
			return apierror.BadRequest(c, "base_url is required when creating a provider")
		}
	}
	if slug != "" && !slugRe.MatchString(slug) {
		return apierror.BadRequest(c, "slug must be 1-32 lowercase letters, digits, or hyphens")
	}

	// Step 1: create or load the provider.
	var prov *db.Provider
	var err error
	if req.ProviderID != "" {
		prov, err = h.DB.GetProvider(ctx, req.ProviderID)
		if err != nil {
			return apierror.NotFound(c, "provider not found")
		}
		if baseURL == "" {
			baseURL = prov.BaseURL
		}
	} else {
		var slugPtr *string
		if slug != "" {
			slugPtr = &slug
		}
		prov, err = h.DB.CreateProvider(ctx, db.CreateProviderParams{
			Name: name, Status: "active",
			Slug: slugPtr, Protocol: protocol, Logo: logo, BaseURL: baseURL,
		})
		if err != nil {
			if errors.Is(err, db.ErrConflict) {
				return apierror.Send(c, fiber.StatusConflict, "conflict", "slug already in use")
			}
			h.Log.ErrorContext(ctx, "import: create provider", slog.String("error", err.Error()))
			return apierror.InternalError(c, "failed to create provider")
		}
		if req.APIKey != "" {
			enc, encErr := crypto.EncryptString(req.APIKey, h.EncryptionKey, providerAAD(prov.ID))
			if encErr != nil {
				return apierror.InternalError(c, "failed to store api key")
			}
			if prov, err = h.DB.UpdateProvider(ctx, prov.ID, db.UpdateProviderParams{APIKeyEncrypted: &enc}); err != nil {
				return apierror.InternalError(c, "failed to store api key")
			}
		}
	}

	// Optional: store API key as first provider connection.
	if req.APIKey != "" {
		if _, connErr := h.createOneConnection(ctx, prov.ID, "Primary", req.APIKey, "apikey", 1); connErr != nil {
			h.Log.WarnContext(ctx, "import: create connection", slog.String("error", connErr.Error()))
		}
	}

	// Step 2: upsert upstream inventory only — no global product models.
	type importResult struct {
		UpstreamID string `json:"upstream_id"`
		ID         string `json:"inventory_id,omitempty"`
		Error      string `json:"error,omitempty"`
	}
	results := make([]importResult, 0, len(req.Models))

	for _, spec := range req.Models {
		res := importResult{UpstreamID: spec.UpstreamID}
		if spec.UpstreamID == "" {
			res.Error = "upstream_id is required"
			results = append(results, res)
			continue
		}
		var cost *provider.CostRef
		if spec.Cost != nil {
			cost = spec.Cost
		} else if preset != nil {
			if cRef, ok := preset.DefaultCost[spec.UpstreamID]; ok {
				cc := cRef
				cost = &cc
			}
		}
		inCost, outCost, cachedInCost, cacheWriteCost := provider.OptionalCostFields(cost)
		m, upsertErr := h.DB.UpsertProviderUpstreamModel(ctx, db.UpsertProviderUpstreamModelParams{
			ProviderID: prov.ID, UpstreamID: spec.UpstreamID,
			IsEnabled: true, CostInputPer1M: inCost, CostOutputPer1M: outCost,
			CostCachedInputPer1M: cachedInCost, CostCacheWritePer1M: cacheWriteCost,
		})
		if upsertErr != nil {
			res.Error = "failed to import upstream model"
			results = append(results, res)
			continue
		}
		res.ID = m.ID
		results = append(results, res)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"provider": providerToJSON(prov),
		"results":  results,
	})
}

// nextRoutePriority returns max(priority)+1 among the model's routes so a new
// route always lands at the end of the fallback order (backup position).
func nextRoutePriority(ctx context.Context, d *db.DB, modelID string) int {
	deps, err := d.ListDeployments(ctx, modelID)
	if err != nil || len(deps) == 0 {
		return 0
	}
	max := 0
	for _, dep := range deps {
		if dep.Priority > max {
			max = dep.Priority
		}
	}
	return max + 1
}
