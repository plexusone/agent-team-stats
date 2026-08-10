// Package verification provides a worker for validating statistics.
package verification

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	worker "github.com/plexusone/omniagent-worker"

	"github.com/plexusone/agent-team-stats/pkg/config"
	"github.com/plexusone/agent-team-stats/pkg/models"
)

// Worker verifies that statistics actually exist in their claimed sources.
type Worker struct {
	worker.BaseWorker
	httpClient *http.Client
	cfg        *config.Config
}

// Config configures the verification worker.
type Config struct {
	worker.WorkerConfig
	AppConfig *config.Config
}

// New creates a new verification worker.
func New(cfg Config) *Worker {
	if cfg.ID == "" {
		cfg.ID = "verification"
	}
	if cfg.Type == "" {
		cfg.Type = "verification"
	}
	if cfg.Version == "" {
		cfg.Version = "2.0.0"
	}

	return &Worker{
		BaseWorker: worker.NewBaseWorker(cfg.WorkerConfig),
		cfg:        cfg.AppConfig,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Init initializes the verification worker.
func (w *Worker) Init(ctx context.Context) error {
	if err := w.BaseWorker.Init(ctx); err != nil {
		return err
	}

	w.Logger().Info("verification worker initialized")
	return nil
}

// Execute verifies the provided candidate statistics.
func (w *Worker) Execute(ctx context.Context, req *worker.Request) (*worker.Response, error) {
	// Extract candidates from input
	candidatesRaw, ok := req.Input["candidates"]
	if !ok {
		return nil, worker.NewValidationError("candidates is required")
	}

	candidates, err := convertCandidates(candidatesRaw)
	if err != nil {
		return nil, worker.NewValidationError(fmt.Sprintf("invalid candidates: %v", err))
	}

	w.Logger().Info("verifying candidates", "count", len(candidates))

	results := make([]models.VerificationResult, 0, len(candidates))
	verifiedCount := 0
	failedCount := 0

	for _, candidate := range candidates {
		result := w.verifyStatistic(ctx, candidate)
		results = append(results, result)

		if result.Verified {
			verifiedCount++
		} else {
			failedCount++
		}
	}

	w.Logger().Info("verification completed",
		"verified", verifiedCount,
		"failed", failedCount)

	// Convert results to verified statistics
	verifiedStats := make([]models.Statistic, 0)
	for _, r := range results {
		if r.Verified && r.Statistic != nil {
			verifiedStats = append(verifiedStats, *r.Statistic)
		}
	}

	return worker.NewResponse(req.ID, map[string]any{
		"results":        results,
		"statistics":     verifiedStats,
		"verified_count": verifiedCount,
		"failed_count":   failedCount,
		"timestamp":      time.Now(),
	}), nil
}

// verifyStatistic verifies a single candidate.
func (w *Worker) verifyStatistic(ctx context.Context, candidate models.CandidateStatistic) models.VerificationResult {
	w.Logger().Debug("verifying statistic", "url", candidate.SourceURL)

	// Fetch source content
	sourceContent, err := w.fetchURL(ctx, candidate.SourceURL)
	if err != nil {
		w.Logger().Warn("failed to fetch source", "url", candidate.SourceURL, "error", err)
		return models.VerificationResult{
			Statistic: &models.Statistic{
				Name:      candidate.Name,
				Value:     candidate.Value,
				Unit:      candidate.Unit,
				Precision: candidate.Precision,
				Source:    candidate.Source,
				SourceURL: candidate.SourceURL,
				Excerpt:   candidate.Excerpt,
				Verified:  false,
				DateFound: time.Now(),
				AsOfDate:  candidate.AsOfDate,
			},
			Verified: false,
			Reason:   fmt.Sprintf("Failed to fetch source: %v", err),
		}
	}

	// Simple verification: check if excerpt appears in source
	verified := strings.Contains(sourceContent, candidate.Excerpt)
	reason := ""
	if !verified {
		reason = "Excerpt not found in source content"
	}

	stat := &models.Statistic{
		Name:      candidate.Name,
		Value:     candidate.Value,
		Unit:      candidate.Unit,
		Precision: candidate.Precision,
		Source:    candidate.Source,
		SourceURL: candidate.SourceURL,
		Excerpt:   candidate.Excerpt,
		Verified:  verified,
		DateFound: time.Now(),
		AsOfDate:  candidate.AsOfDate,
	}

	return models.VerificationResult{
		Statistic: stat,
		Verified:  verified,
		Reason:    reason,
	}
}

// fetchURL retrieves content from a URL.
func (w *Worker) fetchURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; StatsBot/1.0)")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024)) // 100KB limit
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// convertCandidates converts raw input to CandidateStatistic slice.
func convertCandidates(raw any) ([]models.CandidateStatistic, error) {
	// Handle []any (from JSON unmarshaling)
	if arr, ok := raw.([]any); ok {
		candidates := make([]models.CandidateStatistic, 0, len(arr))
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			candidates = append(candidates, models.CandidateStatistic{
				Name:      getString(m, "name"),
				Value:     getFloat32(m, "value"),
				Unit:      getString(m, "unit"),
				Source:    getString(m, "source"),
				SourceURL: getString(m, "source_url"),
				Excerpt:   getString(m, "excerpt"),
			})
		}
		return candidates, nil
	}

	// Handle []models.CandidateStatistic directly
	if candidates, ok := raw.([]models.CandidateStatistic); ok {
		return candidates, nil
	}

	return nil, fmt.Errorf("unsupported type: %T", raw)
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat32(m map[string]any, key string) float32 {
	if v, ok := m[key].(float64); ok {
		return float32(v)
	}
	if v, ok := m[key].(float32); ok {
		return v
	}
	return 0
}
