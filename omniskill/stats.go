// Package omniskill provides the omniskill interface for the statistics team.
package omniskill

import (
	"context"
	"encoding/json"
	"fmt"

	worker "github.com/plexusone/omniagent-worker"
	"github.com/plexusone/omniskill/skill"

	"github.com/plexusone/agent-team-stats/coordinator"
	"github.com/plexusone/agent-team-stats/pkg/config"
)

// StatsSkill wraps the statistics team coordinator as an omniskill.Skill.
type StatsSkill struct {
	coord *worker.Coordinator
	cfg   *config.Config
}

// Config configures the statistics skill.
type Config struct {
	AppConfig *config.Config
	AgentOps  *worker.AgentOpsConfig
}

// New creates a new statistics skill.
func New(cfg Config) *StatsSkill {
	return &StatsSkill{
		cfg: cfg.AppConfig,
		coord: coordinator.New(coordinator.Config{
			AppConfig: cfg.AppConfig,
			AgentOps:  cfg.AgentOps,
		}),
	}
}

// Name returns the skill name.
func (s *StatsSkill) Name() string {
	return "statistics"
}

// Description returns a description of the skill.
func (s *StatsSkill) Description() string {
	return "Research, extract, and verify statistics from the web on any topic"
}

// Version returns the skill version.
func (s *StatsSkill) Version() string {
	return "0.1.0"
}

// Tools returns the tools provided by this skill.
func (s *StatsSkill) Tools() []skill.Tool {
	return []skill.Tool{
		skill.NewTool(
			"research_statistics",
			"Research and verify statistics on a given topic. Returns verified statistics with sources.",
			map[string]skill.Parameter{
				"topic": {
					Type:        "string",
					Description: "The topic to research statistics for",
					Required:    true,
				},
				"min_verified": {
					Type:        "integer",
					Description: "Minimum number of verified statistics to find",
					Default:     10,
				},
				"max_candidates": {
					Type:        "integer",
					Description: "Maximum number of candidate statistics to process",
					Default:     50,
				},
				"reputable_only": {
					Type:        "boolean",
					Description: "Only use sources from reputable domains (.gov, .edu, etc.)",
					Default:     false,
				},
			},
			s.researchStatistics,
		),
		skill.NewTool(
			"get_status",
			"Get the health status of all workers in the statistics team",
			map[string]skill.Parameter{},
			s.getStatus,
		),
	}
}

// Init initializes the skill and its workers.
func (s *StatsSkill) Init(ctx context.Context) error {
	return s.coord.Init(ctx)
}

// Close shuts down the skill and its workers.
func (s *StatsSkill) Close() error {
	ctx := context.Background()
	return s.coord.Shutdown(ctx)
}

// researchStatistics runs the statistics research workflow.
func (s *StatsSkill) researchStatistics(ctx context.Context, params map[string]any) (any, error) {
	// Extract parameters with defaults
	topic, _ := params["topic"].(string)
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	minVerified := 10
	if n, ok := params["min_verified"].(float64); ok {
		minVerified = int(n)
	}

	maxCandidates := 50
	if n, ok := params["max_candidates"].(float64); ok {
		maxCandidates = int(n)
	}

	reputableOnly := false
	if r, ok := params["reputable_only"].(bool); ok {
		reputableOnly = r
	}

	// Execute the coordinator workflow
	req := &worker.CoordinatorRequest{
		Input: map[string]any{
			"topic":          topic,
			"min_verified":   minVerified,
			"max_candidates": maxCandidates,
			"reputable_only": reputableOnly,
		},
	}

	resp, err := s.coord.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("statistics research failed: %w", err)
	}

	// Convert output to JSON-serializable format
	output, err := json.Marshal(resp.Output)
	if err != nil {
		return resp.Output, nil
	}

	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		return resp.Output, nil
	}

	return result, nil
}

// getStatus returns the health status of all workers.
func (s *StatsSkill) getStatus(ctx context.Context, _ map[string]any) (any, error) {
	health := s.coord.Pool().Health(ctx)

	return map[string]any{
		"status":  health.Status,
		"details": health.Details,
	}, nil
}

// Ensure StatsSkill implements skill.Skill.
var _ skill.Skill = (*StatsSkill)(nil)
