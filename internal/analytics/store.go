package analytics

import (
	"context"
	"time"
)

// AnalyticsStore persists invocation analytics for prompts/templates.
type AnalyticsStore interface {
	RecordInvocation(ctx context.Context, record *InvocationRecord) error
	QueryPromptStats(ctx context.Context, filter StatsFilter) (*PromptStats, error)
	QuerySystemStats(ctx context.Context) (*SystemStats, error)
	Close() error
}

// InvocationRecord captures a single prompt invocation.
type InvocationRecord struct {
	PromptID      string
	TemplateID    string
	CombinationID string
	Provider      string
	Model         string
	Duration      time.Duration
	Success       bool
	Cost          float64
	InputTokens   int
	OutputTokens  int
	Metadata      map[string]interface{}
	CreatedAt     time.Time
}

// StatsFilter scopes analytics queries.
type StatsFilter struct {
	PromptID   string
	TemplateID string
}

// PromptStats aggregates request metrics for a prompt/template.
type PromptStats struct {
	PromptID      string    `json:"prompt_id"`
	TemplateID    string    `json:"template_id"`
	TotalRequests int64     `json:"total_requests"`
	SuccessRate   float64   `json:"success_rate"`
	AvgLatencyMS  float64   `json:"avg_latency_ms"`
	AvgCost       float64   `json:"avg_cost"`
	ErrorCount    int64     `json:"error_count"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
}

// SystemStats aggregates analytics across every prompt.
type SystemStats struct {
	TotalPrompts  int64   `json:"total_prompts"`
	TotalRequests int64   `json:"total_requests"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLatencyMS  float64 `json:"avg_latency_ms"`
	AvgCost       float64 `json:"avg_cost"`
}
