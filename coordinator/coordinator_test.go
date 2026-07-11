package coordinator

import (
	"context"
	"testing"

	worker "github.com/plexusone/omniagent-worker"

	"github.com/plexusone/agent-team-stats/pkg/config"
)

func TestStatsWorkflow_Execute_MissingTopic(t *testing.T) {
	cfg := &config.Config{}
	workflow := NewStatsWorkflow(cfg)

	// Create a mock dispatcher
	dispatcher := &mockDispatcher{}

	ctx := context.Background()
	_, err := workflow.Execute(ctx, map[string]any{}, dispatcher)

	if err == nil {
		t.Fatal("expected error for missing topic")
	}

	if err.Error() != "topic is required" {
		t.Errorf("expected 'topic is required', got %q", err.Error())
	}
}

func TestStatsWorkflow_Execute_ResearchFailure(t *testing.T) {
	cfg := &config.Config{}
	workflow := NewStatsWorkflow(cfg)

	// Create a mock dispatcher that fails on research
	dispatcher := &mockDispatcher{
		failOn: "research",
	}

	ctx := context.Background()
	_, err := workflow.Execute(ctx, map[string]any{
		"topic": "test topic",
	}, dispatcher)

	if err == nil {
		t.Fatal("expected error for research failure")
	}
}

func TestNew(t *testing.T) {
	cfg := Config{
		AppConfig: &config.Config{},
	}

	coord := New(cfg)

	if coord.ID() != "stats-coordinator" {
		t.Errorf("expected ID 'stats-coordinator', got %q", coord.ID())
	}

	// Check that workers are registered
	pool := coord.Pool()
	if pool.Size() != 3 {
		t.Errorf("expected 3 workers in pool, got %d", pool.Size())
	}

	ids := pool.IDs()
	expectedIDs := map[string]bool{
		"research":     false,
		"synthesis":    false,
		"verification": false,
	}

	for _, id := range ids {
		if _, ok := expectedIDs[id]; ok {
			expectedIDs[id] = true
		}
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("expected worker %q not found in pool", id)
		}
	}
}

// mockDispatcher implements worker.WorkerDispatcher for testing
type mockDispatcher struct {
	failOn string
	calls  []string
}

func (m *mockDispatcher) Dispatch(ctx context.Context, workerID string, req *worker.Request) (*worker.Response, error) {
	m.calls = append(m.calls, workerID)

	if m.failOn == workerID {
		return nil, worker.NewExecutionError("mock failure", nil)
	}

	// Return mock responses based on worker
	switch workerID {
	case "research":
		return worker.NewResponse(req.ID, map[string]any{
			"search_results": []any{},
		}), nil
	case "synthesis":
		return worker.NewResponse(req.ID, map[string]any{
			"candidates": []any{},
		}), nil
	case "verification":
		return worker.NewResponse(req.ID, map[string]any{
			"statistics":     []any{},
			"verified_count": 0,
			"failed_count":   0,
		}), nil
	default:
		return nil, worker.NewNotFoundError("unknown worker: " + workerID)
	}
}
