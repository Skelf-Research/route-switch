package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/skelf-research/route-switch/internal/analytics"
	"github.com/skelf-research/route-switch/internal/config"
	"github.com/skelf-research/route-switch/internal/core"
	"github.com/skelf-research/route-switch/internal/models"
	"github.com/skelf-research/route-switch/internal/optimizer"
	"github.com/skelf-research/route-switch/internal/storage/dataset"
)

func TestGatewayEndToEnd(t *testing.T) {
	provider := models.NewMockModelProvider()
	evaluator := models.NewSimilarityEvaluationStrategy()
	bayes, err := optimizer.NewGoptunaBayesianOptimizer(map[string]interface{}{"num_trials": 2})
	if err != nil {
		t.Fatalf("bayesian optimizer: %v", err)
	}

	appConfig := &config.Config{
		MiproV2: config.MiproV2Config{
			NumCandidates:            2,
			MaxBootstrappedDemos:     2,
			MaxLabeledDemos:          2,
			NumTrials:                2,
			MinibatchSize:            2,
			MinibatchFullEvalSteps:   1,
			NumInstructionCandidates: 2,
		},
		Evaluation: config.EvaluationConfig{
			DefaultStrategy: "similarity",
			Threshold:       0.7,
			MaxRetries:      1,
		},
		Gateway: config.GatewayConfig{
			Strategy: "round_robin",
			Combinations: []config.PromptCombinationConfig{
				{
					ID:       "combo-1",
					Name:     "combo-1",
					Prompt:   "Assist the user",
					Model:    "gpt-4",
					Provider: "mock",
					Enabled:  true,
					Weight:   10,
				},
			},
			Optimization: config.OptimizationConfig{
				Enabled: false,
			},
		},
	}

	ds := &memoryDatasetStore{records: make(map[string][]*dataset.Record)}
	as := &memoryAnalyticsStore{}

	serviceConfig := &core.ServiceConfig{
		ModelProvider: provider,
		Evaluator:     evaluator,
		Optimizer:     optimizer.NewMIPROv2(provider, evaluator, bayes, appConfig.MiproV2),
		Config:        appConfig,
		DatasetStore:  ds,
	}

	gwConfig := &GatewayConfig{
		Addr:                 ":0",
		LoadBalancerStrategy: RoundRobinStrategy,
		OptimizationEnabled:  false,
	}

	gw, err := NewGateway(serviceConfig, gwConfig, appConfig, ds, as)
	if err != nil {
		t.Fatalf("gateway: %v", err)
	}
	gw.RegisterProvider("mock", provider)

	reqPayload := map[string]interface{}{
		"model": "gpt-4",
		"messages": []Message{
			{Role: "user", Content: "Hello"},
		},
	}
	body, _ := json.Marshal(reqPayload)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	gw.proxy.handleChatCompletions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rr.Code, rr.Body.String())
	}

	if len(ds.records) == 0 {
		t.Fatal("expected dataset records to be captured")
	}

	if as.count == 0 {
		t.Fatal("expected analytics invocation")
	}
}

type memoryDatasetStore struct {
	mu      sync.Mutex
	records map[string][]*dataset.Record
}

func (m *memoryDatasetStore) AddRecord(ctx context.Context, promptID string, record *dataset.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records[promptID] = append(m.records[promptID], record)
	return nil
}

func (m *memoryDatasetStore) ListRecent(ctx context.Context, promptID string, limit int) ([]*dataset.Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.records[promptID], nil
}

func (m *memoryDatasetStore) TotalCount(ctx context.Context, promptID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.records[promptID])), nil
}

func (m *memoryDatasetStore) Close() error { return nil }

type memoryAnalyticsStore struct {
	mu    sync.Mutex
	count int
}

func (m *memoryAnalyticsStore) RecordInvocation(ctx context.Context, record *analytics.InvocationRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.count++
	return nil
}

func (m *memoryAnalyticsStore) QueryPromptStats(ctx context.Context, filter analytics.StatsFilter) (*analytics.PromptStats, error) {
	return &analytics.PromptStats{
		PromptID:      filter.PromptID,
		TemplateID:    filter.TemplateID,
		TotalRequests: int64(m.count),
		SuccessRate:   1,
		AvgLatencyMS:  1,
	}, nil
}

func (m *memoryAnalyticsStore) QuerySystemStats(ctx context.Context) (*analytics.SystemStats, error) {
	return &analytics.SystemStats{
		TotalPrompts:  1,
		TotalRequests: int64(m.count),
		SuccessRate:   1,
	}, nil
}

func (m *memoryAnalyticsStore) Close() error { return nil }
