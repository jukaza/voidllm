// Package provider defines the canonical set of supported LLM provider names
// and the preset catalog used by the source-setup wizard.
package provider

// Preset is one entry in the setup-wizard catalog: a well-known upstream
// source with its connection defaults pre-filled so an admin only has to
// paste an API key.
type Preset struct {
	// ID doubles as the default slug ("openai", "deepseek").
	ID string `json:"id"`
	// Name is the display name shown on the wizard card.
	Name string `json:"name"`
	// Logo is an @lobehub/icons key (e.g. "OpenAI.Color") stored in DB and
	// rendered by the admin SPA. Legacy URL/asset paths still work.
	Logo string `json:"logo"`
	// Protocol is the wire protocol deployments created from this preset use.
	// Must be a key of ValidProviders.
	Protocol string `json:"protocol"`
	// BaseURL is the upstream API base.
	BaseURL string `json:"base_url"`
	// KeyHint shows the expected key shape ("sk-...").
	KeyHint string `json:"key_hint"`
	// DocsURL points at the provider's key-management page.
	DocsURL string `json:"docs_url"`
	// ModelsPath is the relative path of the model-listing endpoint used by
	// discover. Empty means model discovery is not supported (custom/local).
	ModelsPath string `json:"models_path"`
	// DefaultCost maps upstream model name → reference cost prices in USD
	// per 1M tokens. Used to pre-fill route cost prices on import; the admin
	// can override. Prices drift — treat as reference, not billing truth.
	DefaultCost map[string]CostRef `json:"default_cost,omitempty"`
}

// CostRef is a reference cost price pair (USD per 1M tokens).
type CostRef struct {
	In  float64 `json:"in"`
	Out float64 `json:"out"`
	// CachedIn is the cache-hit input price. Zero = no cache discount known.
	CachedIn float64 `json:"cached_in,omitempty"`
	// CacheWrite is the cache-write price. Zero = no separate write price.
	CacheWrite float64 `json:"cache_write,omitempty"`
}

// Presets is the built-in setup-wizard catalog, ordered by popularity.
// Reference prices as of mid-2026; update opportunistically with releases.
var Presets = []Preset{
	{
		ID: "openai", Name: "OpenAI", Logo: "OpenAI.Color",
		Protocol: "openai", BaseURL: "https://api.openai.com/v1",
		KeyHint: "sk-...", DocsURL: "https://platform.openai.com/api-keys",
		ModelsPath: "/models",
		DefaultCost: map[string]CostRef{
			"gpt-4o":      {In: 2.5, Out: 10, CachedIn: 1.25},
			"gpt-4o-mini": {In: 0.15, Out: 0.6, CachedIn: 0.075},
			"gpt-4.1":     {In: 2, Out: 8, CachedIn: 0.5},
			"o3":          {In: 2, Out: 8, CachedIn: 0.5},
		},
	},
	{
		ID: "anthropic", Name: "Anthropic", Logo: "Claude.Color",
		Protocol: "anthropic", BaseURL: "https://api.anthropic.com",
		KeyHint: "sk-ant-...", DocsURL: "https://console.anthropic.com/settings/keys",
		ModelsPath: "/v1/models",
		DefaultCost: map[string]CostRef{
			"claude-sonnet-4-5": {In: 3, Out: 15, CachedIn: 0.3, CacheWrite: 3.75},
			"claude-haiku-4-5":  {In: 1, Out: 5, CachedIn: 0.1, CacheWrite: 1.25},
			"claude-opus-4-8":   {In: 15, Out: 75, CachedIn: 1.5, CacheWrite: 18.75},
		},
	},
	{
		ID: "gemini", Name: "Google Gemini", Logo: "Gemini.Color",
		Protocol: "gemini", BaseURL: "https://generativelanguage.googleapis.com",
		KeyHint: "AIza...", DocsURL: "https://aistudio.google.com/apikey",
		ModelsPath: "/v1beta/models",
		DefaultCost: map[string]CostRef{
			"gemini-2.5-pro":   {In: 1.25, Out: 10, CachedIn: 0.31},
			"gemini-2.5-flash": {In: 0.3, Out: 2.5, CachedIn: 0.075},
		},
	},
	{
		ID: "deepseek", Name: "DeepSeek", Logo: "DeepSeek.Color",
		Protocol: "openai", BaseURL: "https://api.deepseek.com",
		KeyHint: "sk-...", DocsURL: "https://platform.deepseek.com/api_keys",
		ModelsPath: "/models",
		DefaultCost: map[string]CostRef{
			"deepseek-chat":     {In: 0.27, Out: 1.1, CachedIn: 0.07},
			"deepseek-reasoner": {In: 0.55, Out: 2.19, CachedIn: 0.14},
		},
	},
	{
		ID: "openrouter", Name: "OpenRouter", Logo: "OpenRouter",
		Protocol: "openai", BaseURL: "https://openrouter.ai/api/v1",
		KeyHint: "sk-or-...", DocsURL: "https://openrouter.ai/settings/keys",
		ModelsPath: "/models",
	},
	{
		ID: "groq", Name: "Groq", Logo: "Groq",
		Protocol: "openai", BaseURL: "https://api.groq.com/openai/v1",
		KeyHint: "gsk_...", DocsURL: "https://console.groq.com/keys",
		ModelsPath: "/models",
	},
	{
		ID: "xai", Name: "xAI", Logo: "XAI",
		Protocol: "openai", BaseURL: "https://api.x.ai/v1",
		KeyHint: "xai-...", DocsURL: "https://console.x.ai",
		ModelsPath: "/models",
	},
	{
		ID: "mistral", Name: "Mistral", Logo: "Mistral.Color",
		Protocol: "openai", BaseURL: "https://api.mistral.ai/v1",
		KeyHint: "...", DocsURL: "https://console.mistral.ai/api-keys",
		ModelsPath: "/models",
	},
	{
		ID: "siliconflow", Name: "SiliconFlow", Logo: "SiliconCloud.Color",
		Protocol: "openai", BaseURL: "https://api.siliconflow.cn/v1",
		KeyHint: "sk-...", DocsURL: "https://cloud.siliconflow.cn/account/ak",
		ModelsPath: "/models",
	},
	{
		ID: "moonshot", Name: "Moonshot (Kimi)", Logo: "Kimi.Color",
		Protocol: "openai", BaseURL: "https://api.moonshot.ai/v1",
		KeyHint: "sk-...", DocsURL: "https://platform.moonshot.ai/console/api-keys",
		ModelsPath: "/models",
	},
	{
		ID: "zhipu", Name: "Zhipu (GLM)", Logo: "Zhipu.Color",
		Protocol: "openai", BaseURL: "https://open.bigmodel.cn/api/paas/v4",
		KeyHint: "...", DocsURL: "https://open.bigmodel.cn/usercenter/apikeys",
		ModelsPath: "/models",
	},
	{
		ID: "ollama-cloud", Name: "Ollama Cloud", Logo: "Ollama",
		Protocol: "openai", BaseURL: "https://ollama.com/v1",
		KeyHint: "ollama.com API key", DocsURL: "https://ollama.com/settings/keys",
		ModelsPath: "/models",
	},
	{
		// Self-hosted only; excluded from the public wizard catalog.
		ID: "ollama", Name: "Ollama (self-hosted)", Logo: "Ollama",
		Protocol: "ollama", BaseURL: "http://localhost:11434/v1",
		KeyHint: "(không cần key)", DocsURL: "https://ollama.com",
		ModelsPath: "/models",
	},
	{
		ID: "azure", Name: "Azure OpenAI", Logo: "AzureAI",
		Protocol: "azure", BaseURL: "",
		KeyHint: "...", DocsURL: "https://portal.azure.com",
		// Azure model listing needs deployment enumeration, not /models.
		ModelsPath: "",
	},
	{
		ID: "vertex", Name: "Vertex AI", Logo: "Gemini.Color",
		Protocol: "vertex", BaseURL: "",
		KeyHint: "(service account)", DocsURL: "https://console.cloud.google.com/vertex-ai",
		ModelsPath: "",
	},
	{
		ID: "custom", Name: "Custom (OpenAI-compatible)", Logo: "LobeHub",
		Protocol: "custom", BaseURL: "",
		KeyHint: "...", DocsURL: "",
		ModelsPath: "/models",
	},
}

// PresetByID returns the preset with the given ID, or nil.
func PresetByID(id string) *Preset {
	for i := range Presets {
		if Presets[i].ID == id {
			return &Presets[i]
		}
	}
	return nil
}
