package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// phaseResult holds the metrics from a single benchmark phase.
type phaseResult struct {
	Name    string
	Metrics *vegeta.Metrics
}

// benchResult holds all results from a complete scenario run.
type benchResult struct {
	Scenario     string
	StartedAt    time.Time
	Duration     time.Duration
	PhaseResults []phaseResult
}

// warmup sends a few requests to each target to prime caches and connection pools.
func warmup(endpoints *endpointSet) {
	targets := []vegeta.Target{
		{Method: "POST", URL: endpoints.mockLLM + "/v1/chat/completions",
			Header: http.Header{"Content-Type": []string{"application/json"}},
			Body:   []byte(`{"model":"mock","messages":[{"role":"user","content":"warmup"}]}`)},
		{Method: "POST", URL: endpoints.proxy + "/v1/chat/completions",
			Header: http.Header{"Content-Type": []string{"application/json"}, "Authorization": []string{"Bearer " + endpoints.apiKey}},
			Body:   []byte(`{"model":"mock","messages":[{"role":"user","content":"warmup"}]}`)},
	}

	atk := vegeta.NewAttacker(vegeta.Timeout(10 * time.Second))
	for _, t := range targets {
		targeter := vegeta.NewStaticTargeter(t)
		for res := range atk.Attack(targeter, vegeta.Rate{Freq: 50, Per: time.Second}, 1*time.Second, "warmup") {
			_ = res
		}
	}
}

// runScenario executes all phases of a scenario and returns results.
func runScenario(s *scenario, endpoints *endpointSet) *benchResult {
	result := &benchResult{
		Scenario:  s.Name,
		StartedAt: time.Now(),
	}

	// Check if mixed scenario (parallel phases)
	isMixed := s.Name == "mixed"

	if isMixed {
		results := runPhasesParallel(s.Phases, endpoints)
		result.PhaseResults = results
	} else {
		for _, p := range s.Phases {
			fmt.Printf("  ▸ %s\n", p.Name)
			metrics := runPhase(p, endpoints)
			result.PhaseResults = append(result.PhaseResults, phaseResult{
				Name:    p.Name,
				Metrics: metrics,
			})
		}
	}

	result.Duration = time.Since(result.StartedAt)
	return result
}

// runPhase executes a single load test phase.
func runPhase(p phase, ep *endpointSet) *vegeta.Metrics {
	targeter := buildTargeter(p, ep)

	// Scale Workers and Connections to the peak rate.
	// Rule of thumb: maxWorkers = peakRPS * maxLatency(200ms) with a floor of 1000.
	maxWorkers := uint64(1000)
	maxConns := 1000
	if p.MaxRate > 0 {
		scaled := uint64(p.MaxRate) * 200 / 1000 // peakRPS * 0.2s
		if scaled > maxWorkers {
			maxWorkers = scaled
		}
		if int(scaled) > maxConns {
			maxConns = int(scaled)
		}
	}

	atk := vegeta.NewAttacker(
		vegeta.Workers(10),
		vegeta.MaxWorkers(maxWorkers),
		vegeta.Connections(maxConns),
		vegeta.Timeout(30*time.Second),
		vegeta.KeepAlive(true),
	)

	var metrics vegeta.Metrics
	for res := range atk.Attack(targeter, p.Pacer, p.Duration, p.Name) {
		metrics.Add(res)
	}
	metrics.Close()
	return &metrics
}

// runPhasesParallel runs multiple phases concurrently (for mixed workload).
func runPhasesParallel(phases []phase, ep *endpointSet) []phaseResult {
	var wg sync.WaitGroup
	results := make([]phaseResult, len(phases))

	for i, p := range phases {
		wg.Add(1)
		go func(idx int, ph phase) {
			defer wg.Done()
			fmt.Printf("  ▸ %s (parallel)\n", ph.Name)
			metrics := runPhase(ph, ep)
			results[idx] = phaseResult{Name: ph.Name, Metrics: metrics}
		}(i, p)
	}
	wg.Wait()
	return results
}

// buildTargeter creates a vegeta.Targeter for the given phase.
func buildTargeter(p phase, ep *endpointSet) vegeta.Targeter {
	var url string
	var headers http.Header
	var body []byte

	switch p.Target {
	case "llm-direct":
		url = ep.mockLLM + "/v1/chat/completions"
		headers = http.Header{"Content-Type": []string{"application/json"}}
		if p.Stream {
			body = []byte(`{"model":"mock","stream":true,"messages":[{"role":"user","content":"Summarize Harry Potter"}]}`)
		} else {
			body = makeLLMBody(p.BodySize)
		}

	case "llm-proxy":
		url = ep.proxy + "/v1/chat/completions"
		headers = http.Header{
			"Content-Type":  []string{"application/json"},
			"Authorization": []string{"Bearer " + ep.apiKey},
		}
		if p.Stream {
			body = []byte(`{"model":"mock","stream":true,"messages":[{"role":"user","content":"Summarize Harry Potter"}]}`)
		} else {
			body = makeLLMBody(p.BodySize)
		}

	default:
		panic(fmt.Sprintf("unknown phase target: %q", p.Target))
	}

	target := vegeta.Target{
		Method: "POST",
		URL:    url,
		Header: headers,
		Body:   body,
	}

	return vegeta.NewStaticTargeter(target)
}

// makeLLMBody creates an OpenAI chat completion request body.
// If size > 0, pads the system prompt to reach the target size.
func makeLLMBody(size int) []byte {
	if size <= 0 {
		return []byte(`{"model":"mock","messages":[{"role":"user","content":"hello"}]}`)
	}

	// Build a large system prompt to hit target body size
	padding := strings.Repeat("x", size)
	return []byte(fmt.Sprintf(
		`{"model":"mock","messages":[{"role":"system","content":"%s"},{"role":"user","content":"hello"}]}`,
		padding,
	))
}