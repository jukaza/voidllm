package proxy

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/ratelimit"
	"github.com/jukaza/tavo/internal/upstream"
)

type comboRRState struct {
	mu        sync.Mutex
	stepIndex int
	useCount  int
}

var comboRR sync.Map

func connStrategyForProvider(strategy string) string {
	if strategy == "round-robin" {
		return "round-robin"
	}
	return "fill-first"
}

func (p *ProxyHandler) orderComboSteps(modelName string, steps []RouteStep, strategy string, sticky int) []RouteStep {
	if strategy != "round-robin" || len(steps) <= 1 {
		return steps
	}
	if sticky < 1 {
		sticky = 1
	}
	stateAny, _ := comboRR.LoadOrStore(modelName, &comboRRState{})
	state := stateAny.(*comboRRState)
	state.mu.Lock()
	defer state.mu.Unlock()
	idx := state.stepIndex % len(steps)
	state.useCount++
	if state.useCount > sticky {
		state.stepIndex = (state.stepIndex + 1) % len(steps)
		state.useCount = 1
		idx = state.stepIndex
	}
	return []RouteStep{steps[idx]}
}

func (p *ProxyHandler) tryComboModel(
	c fiber.Ctx,
	model Model,
	pickBody func(dep Deployment) []byte,
	envelope requestEnvelope,
) (*http.Response, context.CancelFunc, Adapter, Deployment, error, int, error) {
	if p.UpstreamStore == nil || len(model.RouteSteps) == 0 {
		return p.tryModelDeployments(c, model, pickBody, envelope)
	}

	ctx := c.Context()
	steps := p.orderComboSteps(model.Name, model.RouteSteps, model.RoutingStrategy, model.StickyRoundRobinLimit)

	var (
		lastErr    error
		lastStatus int
	)

	for _, step := range steps {
		if step.ProviderRPMLimit > 0 && p.UpstreamLimiter != nil {
			scope := ratelimit.ScopeProvider(step.ProviderID)
			if !p.UpstreamLimiter.Allow(scope, step.ProviderRPMLimit, 0, 0) {
				p.Log.LogAttrs(ctx, slog.LevelWarn, "combo hop: provider rpm cap reached, skipping step",
					slog.String("model", model.Name),
					slog.String("provider_id", step.ProviderID),
					slog.Int("rpm_limit", step.ProviderRPMLimit),
				)
				continue
			}
		}

		exclude := map[string]struct{}{}
		connStrategy := connStrategyForProvider(step.ConnStrategy)
		sticky := step.ConnSticky
		if sticky < 1 {
			sticky = 1
		}

		for {
			conn, selErr := p.UpstreamStore.Select(ctx, step.ProviderID, step.UpstreamModel, connStrategy, sticky, exclude)
			if selErr != nil {
				if errors.Is(selErr, db.ErrNotFound) || errors.Is(selErr, upstream.ErrProviderPaused) {
					break
				}
				lastErr = selErr
				break
			}

			apiKey, decErr := p.UpstreamStore.DecryptKey(conn)
			if decErr != nil {
				p.Log.LogAttrs(ctx, slog.LevelWarn, "combo hop: failed to decrypt connection key, rotating",
					slog.String("model", model.Name),
					slog.String("connection", conn.Name),
					slog.String("error", decErr.Error()),
				)
				exclude[conn.ID] = struct{}{}
				continue
			}
			if apiKey == "" {
				apiKey = step.ProviderDefaultKey
			}
			dep := Deployment{
				ID:              conn.ID,
				Name:            conn.Name,
				Provider:        step.Provider,
				BaseURL:         step.BaseURL,
				APIKey:          apiKey,
				ProviderID:      step.ProviderID,
				ConnectionID:    conn.ID,
				UpstreamModel:   step.UpstreamModel,
				ProviderRPMLimit: step.ProviderRPMLimit,
				CostInputPer1M:       step.CostInputPer1M,
				CostOutputPer1M:      step.CostOutputPer1M,
				CostCachedInputPer1M: step.CostCachedInputPer1M,
				CostCacheWritePer1M:  step.CostCacheWritePer1M,
				Weight:          1,
				destPrivate:     classifyDestPrivate(step.BaseURL),
			}

			hopModel := model
			hopModel.RouteSteps = nil
			hopModel.Deployments = []Deployment{dep}

			hopPickBody := func(d Deployment) []byte {
				raw := pickBody(d)
				rewritten, err := rewriteModelInBody(raw, step.UpstreamModel)
				if err != nil {
					return raw
				}
				return rewritten
			}

			resp, cancel, adapter, usedDep, hopErr, status, tryErr := p.tryModelDeployments(c, hopModel, hopPickBody, envelope)
			if tryErr != nil {
				return nil, nil, nil, Deployment{}, nil, 0, tryErr
			}

			if resp != nil && !isRetryable(status) && hopErr == nil {
				_ = p.UpstreamStore.ClearSuccess(ctx, conn.ID, step.UpstreamModel)
				return resp, cancel, adapter, usedDep, hopErr, status, nil
			}

			errText := ""
			var retryAfter time.Duration
			if resp != nil {
				lastStatus = status
				retryAfter = upstream.ParseRetryAfter(resp.Header.Get("Retry-After"))
				if resp.Body != nil {
					b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
					errText = string(b)
					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
				}
				if cancel != nil {
					cancel()
				}
			}
			if hopErr != nil {
				lastErr = hopErr
			}

			backoff := conn.BackoffLevel
			_ = p.UpstreamStore.MarkUnavailable(ctx, conn.ID, step.UpstreamModel, status, errText, backoff, retryAfter)

			if resp != nil && !isRetryable(status) {
				return resp, nil, adapter, usedDep, hopErr, status, nil
			}

			exclude[conn.ID] = struct{}{}
			p.Log.LogAttrs(ctx, slog.LevelWarn, "combo hop failed, rotating connection",
				slog.String("model", model.Name),
				slog.String("upstream", step.UpstreamModel),
				slog.String("connection", conn.Name),
				slog.Int("status", status),
			)
		}
	}

	return nil, nil, nil, Deployment{}, lastErr, lastStatus, nil
}