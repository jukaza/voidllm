package main

// @title           VoidLLM API
// @version         0.2.0
// @description     Prepaid LLM API marketplace with OpenAI-compatible proxy, wallet billing, and model catalog.
// @BasePath        /api/v1

// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     API key (sk-) or legacy vl_uk_ / vl_sk_

// @tag.name        auth
// @tag.description Authentication and user profile
// @tag.name        keys
// @tag.description API key management
// @tag.name        users
// @tag.description User management
// @tag.name        models
// @tag.description Model registry management
// @tag.name        model-aliases
// @tag.description Model alias management
// @tag.name        dashboard
// @tag.description Dashboard statistics
// @tag.name        usage
// @tag.description Usage analytics
// @tag.name        wallet
// @tag.description Prepaid wallet and top-ups
// @tag.name        providers
// @tag.description Upstream provider management
// @tag.name        marketplace
// @tag.description Top-up review and marketplace admin