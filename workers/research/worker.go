// Package research provides a worker for discovering sources via web search.
package research

import (
	"context"
	"fmt"
	"strings"
	"time"

	worker "github.com/plexusone/omniagent-worker"

	"github.com/plexusone/agent-team-stats/pkg/config"
	"github.com/plexusone/agent-team-stats/pkg/models"
	"github.com/plexusone/agent-team-stats/pkg/search"
)

// Worker finds relevant sources using web search.
// This worker does NOT use LLM - it's pure search.
type Worker struct {
	worker.BaseWorker
	searchSvc *search.Service
	cfg       *config.Config
}

// Config configures the research worker.
type Config struct {
	worker.WorkerConfig
	AppConfig *config.Config
}

// New creates a new research worker.
func New(cfg Config) *Worker {
	if cfg.ID == "" {
		cfg.ID = "research"
	}
	if cfg.Type == "" {
		cfg.Type = "research"
	}
	if cfg.Version == "" {
		cfg.Version = "2.0.0"
	}

	return &Worker{
		BaseWorker: worker.NewBaseWorker(cfg.WorkerConfig),
		cfg:        cfg.AppConfig,
	}
}

// Init initializes the search service.
func (w *Worker) Init(ctx context.Context) error {
	if err := w.BaseWorker.Init(ctx); err != nil {
		return err
	}

	searchSvc, err := search.NewService(w.cfg)
	if err != nil {
		return fmt.Errorf("search service required: %w", err)
	}
	w.searchSvc = searchSvc

	w.Logger().Info("research worker initialized",
		"search_provider", w.cfg.SearchProvider)
	return nil
}

// Execute performs web search for the given topic.
func (w *Worker) Execute(ctx context.Context, req *worker.Request) (*worker.Response, error) {
	// Extract input parameters
	topic, _ := req.Input["topic"].(string)
	if topic == "" {
		return nil, worker.NewValidationError("topic is required")
	}

	numResults := 20
	if n, ok := req.Input["num_results"].(float64); ok {
		numResults = int(n)
	}

	reputableOnly := false
	if r, ok := req.Input["reputable_only"].(bool); ok {
		reputableOnly = r
	}

	w.Logger().Info("searching for sources",
		"topic", topic,
		"num_results", numResults,
		"reputable_only", reputableOnly)

	// Perform search
	searchResp, err := w.searchSvc.SearchForStatistics(ctx, topic, numResults)
	if err != nil {
		return nil, worker.NewExecutionError("search failed", err)
	}

	w.Logger().Info("search completed", "results", searchResp.Total)

	// Convert to SearchResult models
	results := make([]models.SearchResult, 0, len(searchResp.Results))
	for i, result := range searchResp.Results {
		// Filter for reputable sources if requested
		if reputableOnly && !isReputableSource(result.DisplayLink) {
			w.Logger().Debug("filtering non-reputable source", "domain", result.DisplayLink)
			continue
		}

		results = append(results, models.SearchResult{
			URL:      result.URL,
			Title:    result.Title,
			Snippet:  result.Snippet,
			Domain:   result.DisplayLink,
			Position: i + 1,
		})
	}

	w.Logger().Info("sources found", "count", len(results))

	return worker.NewResponse(req.ID, map[string]any{
		"search_results": results,
		"topic":          topic,
		"timestamp":      time.Now(),
	}), nil
}

// isReputableSource checks if a domain is from a reputable source.
func isReputableSource(domain string) bool {
	reputableDomains := []string{
		".gov", ".edu", // Government and education
		"who.int", "un.org", "worldbank.org", // International orgs
		"pewresearch.org", "gallup.com", // Research organizations
		"nature.com", "science.org", "nejm.org", // Journals
	}

	domainLower := strings.ToLower(domain)
	for _, rep := range reputableDomains {
		if strings.Contains(domainLower, rep) {
			return true
		}
	}
	return false
}
