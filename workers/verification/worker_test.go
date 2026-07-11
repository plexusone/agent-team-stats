package verification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	worker "github.com/plexusone/omniagent-worker"

	"github.com/plexusone/agent-team-stats/pkg/models"
)

func TestWorker_Execute(t *testing.T) {
	// Create mock server that returns content with excerpts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/found":
			_, _ = w.Write([]byte("The global temperature has risen by 1.1 degrees Celsius since pre-industrial times."))
		case "/notfound":
			_, _ = w.Write([]byte("This page contains different content entirely."))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	w := New(Config{})
	ctx := context.Background()

	if err := w.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	candidates := []models.CandidateStatistic{
		{
			Name:      "Global temperature increase",
			Value:     1.1,
			Unit:      "°C",
			Source:    "Test",
			SourceURL: server.URL + "/found",
			Excerpt:   "1.1 degrees Celsius",
		},
		{
			Name:      "Not found stat",
			Value:     99,
			Unit:      "%",
			Source:    "Test",
			SourceURL: server.URL + "/notfound",
			Excerpt:   "This excerpt does not exist",
		},
	}

	req := worker.NewRequest(map[string]any{
		"candidates": candidates,
	})

	resp, err := w.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	verifiedCount, ok := resp.Output["verified_count"].(int)
	if !ok {
		t.Fatal("verified_count not found or wrong type")
	}
	if verifiedCount != 1 {
		t.Errorf("expected verified_count=1, got %d", verifiedCount)
	}

	failedCount, ok := resp.Output["failed_count"].(int)
	if !ok {
		t.Fatal("failed_count not found or wrong type")
	}
	if failedCount != 1 {
		t.Errorf("expected failed_count=1, got %d", failedCount)
	}

	statistics, ok := resp.Output["statistics"].([]models.Statistic)
	if !ok {
		t.Fatal("statistics not found or wrong type")
	}
	if len(statistics) != 1 {
		t.Errorf("expected 1 verified statistic, got %d", len(statistics))
	}
}

func TestWorker_ExecuteMissingCandidates(t *testing.T) {
	w := New(Config{})
	ctx := context.Background()

	if err := w.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	req := worker.NewRequest(map[string]any{})

	_, err := w.Execute(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing candidates")
	}
}

func TestWorker_Health(t *testing.T) {
	w := New(Config{})
	ctx := context.Background()

	if err := w.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	status := w.Health(ctx)
	if status.Status != worker.HealthStatusHealthy {
		t.Errorf("expected healthy status, got %s", status.Status)
	}
}

func TestConvertCandidates(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantLen int
		wantErr bool
	}{
		{
			name: "slice of maps",
			input: []any{
				map[string]any{
					"name":       "Test stat",
					"value":      float64(42),
					"unit":       "units",
					"source":     "Test",
					"source_url": "https://example.com",
					"excerpt":    "42 units",
				},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "direct slice",
			input: []models.CandidateStatistic{
				{Name: "Test", Value: 1, Unit: "x", Source: "S", SourceURL: "http://x", Excerpt: "1"},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   "not a slice",
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertCandidates(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertCandidates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result) != tt.wantLen {
				t.Errorf("convertCandidates() len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}
