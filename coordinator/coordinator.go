// Package coordinator provides workflow orchestration for the statistics team.
package coordinator

import (
	"context"
	"fmt"
	"time"

	worker "github.com/plexusone/omniagent-worker"

	"github.com/plexusone/agent-team-stats/pkg/config"
	"github.com/plexusone/agent-team-stats/pkg/models"
	"github.com/plexusone/agent-team-stats/workers/research"
	"github.com/plexusone/agent-team-stats/workers/synthesis"
	"github.com/plexusone/agent-team-stats/workers/verification"
)

// Config configures the coordinator.
type Config struct {
	AppConfig *config.Config
	AgentOps  *worker.AgentOpsConfig
}

// New creates a new coordinator with all workers registered.
func New(cfg Config) *worker.Coordinator {
	coord := worker.NewCoordinator(worker.CoordinatorConfig{
		ID:       "stats-coordinator",
		Workflow: NewStatsWorkflow(cfg.AppConfig),
		AgentOps: cfg.AgentOps,
		Timeout:  10 * time.Minute,
	})

	// Register workers in the pool
	researchWorker := research.New(research.Config{
		WorkerConfig: worker.WorkerConfig{
			ID:   "research",
			Type: "research",
		},
		AppConfig: cfg.AppConfig,
	})

	synthesisWorker := synthesis.New(synthesis.Config{
		WorkerConfig: worker.WorkerConfig{
			ID:   "synthesis",
			Type: "synthesis",
		},
		AppConfig: cfg.AppConfig,
	})

	verificationWorker := verification.New(verification.Config{
		WorkerConfig: worker.WorkerConfig{
			ID:   "verification",
			Type: "verification",
		},
		AppConfig: cfg.AppConfig,
	})

	// Register workers - ignoring errors as we control the IDs
	_ = coord.Pool().Register(researchWorker)
	_ = coord.Pool().Register(synthesisWorker)
	_ = coord.Pool().Register(verificationWorker)

	return coord
}

// StatsWorkflow implements the statistics research workflow.
type StatsWorkflow struct {
	cfg *config.Config
}

// NewStatsWorkflow creates a new statistics workflow.
func NewStatsWorkflow(cfg *config.Config) *StatsWorkflow {
	return &StatsWorkflow{cfg: cfg}
}

// Execute runs the statistics research workflow.
func (w *StatsWorkflow) Execute(ctx context.Context, input map[string]any, dispatcher worker.WorkerDispatcher) (map[string]any, error) {
	// Extract input parameters
	topic, _ := input["topic"].(string)
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	minVerified := 10
	if n, ok := input["min_verified"].(float64); ok {
		minVerified = int(n)
	}

	maxCandidates := 50
	if n, ok := input["max_candidates"].(float64); ok {
		maxCandidates = int(n)
	}

	reputableOnly := false
	if r, ok := input["reputable_only"].(bool); ok {
		reputableOnly = r
	}

	// Step 1: Research - find sources
	researchReq := worker.NewRequest(map[string]any{
		"topic":          topic,
		"num_results":    maxCandidates,
		"reputable_only": reputableOnly,
	})

	researchResp, err := dispatcher.Dispatch(ctx, "research", researchReq)
	if err != nil {
		return nil, fmt.Errorf("research failed: %w", err)
	}

	searchResults, ok := researchResp.Output["search_results"]
	if !ok {
		return nil, fmt.Errorf("research did not return search_results")
	}

	// Step 2: Synthesis - extract statistics
	synthesisReq := worker.NewRequest(map[string]any{
		"topic":          topic,
		"search_results": searchResults,
		"min_statistics": minVerified,
		"max_statistics": maxCandidates,
	})

	synthesisResp, err := dispatcher.Dispatch(ctx, "synthesis", synthesisReq)
	if err != nil {
		return nil, fmt.Errorf("synthesis failed: %w", err)
	}

	candidates, ok := synthesisResp.Output["candidates"]
	if !ok {
		return nil, fmt.Errorf("synthesis did not return candidates")
	}

	// Step 3: Verification - validate statistics
	verificationReq := worker.NewRequest(map[string]any{
		"candidates": candidates,
	})

	verificationResp, err := dispatcher.Dispatch(ctx, "verification", verificationReq)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	// Build final response
	statistics, _ := verificationResp.Output["statistics"].([]models.Statistic)
	verifiedCount, _ := verificationResp.Output["verified_count"].(int)
	failedCount, _ := verificationResp.Output["failed_count"].(int)

	// Handle type conversion for counts (may come as float64 from JSON)
	if vc, ok := verificationResp.Output["verified_count"].(float64); ok {
		verifiedCount = int(vc)
	}
	if fc, ok := verificationResp.Output["failed_count"].(float64); ok {
		failedCount = int(fc)
	}

	return map[string]any{
		"topic":            topic,
		"statistics":       statistics,
		"total_candidates": len(candidates.([]models.CandidateStatistic)),
		"verified_count":   verifiedCount,
		"failed_count":     failedCount,
		"target_count":     minVerified,
		"partial":          verifiedCount < minVerified,
		"timestamp":        time.Now(),
	}, nil
}
