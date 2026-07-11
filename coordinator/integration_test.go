//go:build integration

package coordinator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	worker "github.com/plexusone/omniagent-worker"

	"github.com/plexusone/agent-team-stats/pkg/config"
	"github.com/plexusone/agent-team-stats/pkg/models"
)

// TestFullWorkflow_Integration tests the complete research→synthesis→verification pipeline.
// This test uses mock workers that simulate the real behavior.
func TestFullWorkflow_Integration(t *testing.T) {
	// Create mock HTTP server for verification
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return content with the expected excerpt
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
			<html>
			<body>
			<p>According to recent studies, the global temperature has increased by 1.1 degrees Celsius since pre-industrial times.</p>
			<p>Sea levels are rising at approximately 3.3 mm per year according to satellite data.</p>
			</body>
			</html>
		`))
	}))
	defer mockServer.Close()

	cfg := &config.Config{}
	workflow := NewStatsWorkflow(cfg)

	// Create mock dispatcher that simulates the workers
	dispatcher := &integrationMockDispatcher{
		mockServerURL: mockServer.URL,
	}

	ctx := context.Background()

	// Execute workflow
	output, err := workflow.Execute(ctx, map[string]any{
		"topic":          "climate change statistics",
		"min_verified":   2,
		"max_candidates": 10,
		"reputable_only": false,
	}, dispatcher)

	if err != nil {
		t.Fatalf("workflow execution failed: %v", err)
	}

	// Verify output
	topic, ok := output["topic"].(string)
	if !ok || topic != "climate change statistics" {
		t.Errorf("expected topic 'climate change statistics', got %v", output["topic"])
	}

	verifiedCount, ok := output["verified_count"].(int)
	if !ok {
		t.Error("verified_count not found in output")
	}

	if verifiedCount < 1 {
		t.Errorf("expected at least 1 verified statistic, got %d", verifiedCount)
	}

	t.Logf("Integration test passed: %d verified statistics", verifiedCount)
}

// integrationMockDispatcher simulates the worker behavior for integration tests.
type integrationMockDispatcher struct {
	mockServerURL string
}

func (d *integrationMockDispatcher) Dispatch(ctx context.Context, workerID string, req *worker.Request) (*worker.Response, error) {
	switch workerID {
	case "research":
		return d.handleResearch(req)
	case "synthesis":
		return d.handleSynthesis(req)
	case "verification":
		return d.handleVerification(req)
	default:
		return nil, worker.NewNotFoundError("unknown worker: " + workerID)
	}
}

func (d *integrationMockDispatcher) handleResearch(req *worker.Request) (*worker.Response, error) {
	// Return mock search results
	results := []models.SearchResult{
		{
			URL:      d.mockServerURL + "/climate",
			Title:    "Climate Change Statistics",
			Snippet:  "Global temperature has increased by 1.1 degrees Celsius...",
			Domain:   "climate.nasa.gov",
			Position: 1,
		},
		{
			URL:      d.mockServerURL + "/sea-level",
			Title:    "Sea Level Rise Data",
			Snippet:  "Sea levels are rising at 3.3 mm per year...",
			Domain:   "noaa.gov",
			Position: 2,
		},
	}

	return worker.NewResponse(req.ID, map[string]any{
		"search_results": results,
		"topic":          req.Input["topic"],
		"timestamp":      time.Now(),
	}), nil
}

func (d *integrationMockDispatcher) handleSynthesis(req *worker.Request) (*worker.Response, error) {
	// Return mock extracted candidates
	candidates := []models.CandidateStatistic{
		{
			Name:      "Global temperature increase",
			Value:     1.1,
			Unit:      "degrees Celsius",
			Source:    "climate.nasa.gov",
			SourceURL: d.mockServerURL + "/climate",
			Excerpt:   "1.1 degrees Celsius since pre-industrial times",
		},
		{
			Name:      "Sea level rise rate",
			Value:     3.3,
			Unit:      "mm per year",
			Source:    "noaa.gov",
			SourceURL: d.mockServerURL + "/sea-level",
			Excerpt:   "3.3 mm per year according to satellite data",
		},
	}

	return worker.NewResponse(req.ID, map[string]any{
		"candidates":       candidates,
		"topic":            req.Input["topic"],
		"sources_analyzed": 2,
		"timestamp":        time.Now(),
	}), nil
}

func (d *integrationMockDispatcher) handleVerification(req *worker.Request) (*worker.Response, error) {
	candidatesRaw, ok := req.Input["candidates"]
	if !ok {
		return nil, worker.NewValidationError("candidates is required")
	}

	var candidates []models.CandidateStatistic
	if c, ok := candidatesRaw.([]models.CandidateStatistic); ok {
		candidates = c
	} else if arr, ok := candidatesRaw.([]any); ok {
		for _, item := range arr {
			if c, ok := item.(models.CandidateStatistic); ok {
				candidates = append(candidates, c)
			}
		}
	}

	// Simulate verification - mark all as verified for testing
	statistics := make([]models.Statistic, 0, len(candidates))
	for _, c := range candidates {
		statistics = append(statistics, models.Statistic{
			Name:      c.Name,
			Value:     c.Value,
			Unit:      c.Unit,
			Source:    c.Source,
			SourceURL: c.SourceURL,
			Excerpt:   c.Excerpt,
			Verified:  true,
			DateFound: time.Now(),
		})
	}

	return worker.NewResponse(req.ID, map[string]any{
		"statistics":     statistics,
		"verified_count": len(statistics),
		"failed_count":   0,
		"timestamp":      time.Now(),
	}), nil
}
