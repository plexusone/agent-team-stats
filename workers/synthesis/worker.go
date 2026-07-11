// Package synthesis provides a worker for extracting statistics from web content.
package synthesis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	worker "github.com/plexusone/omniagent-worker"
	"google.golang.org/adk/model"
	"google.golang.org/genai"

	"github.com/plexusone/agent-team-stats/pkg/config"
	"github.com/plexusone/agent-team-stats/pkg/llm"
	"github.com/plexusone/agent-team-stats/pkg/models"
)

// Worker extracts statistics from webpage content using LLM.
type Worker struct {
	worker.BaseWorker
	llmModel   model.LLM
	httpClient *http.Client
	cfg        *config.Config
}

// Config configures the synthesis worker.
type Config struct {
	worker.WorkerConfig
	AppConfig *config.Config
}

// New creates a new synthesis worker.
func New(cfg Config) *Worker {
	if cfg.ID == "" {
		cfg.ID = "synthesis"
	}
	if cfg.Type == "" {
		cfg.Type = "synthesis"
	}
	if cfg.Version == "" {
		cfg.Version = "2.0.0"
	}

	return &Worker{
		BaseWorker: worker.NewBaseWorker(cfg.WorkerConfig),
		cfg:        cfg.AppConfig,
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

// Init initializes the LLM model.
func (w *Worker) Init(ctx context.Context) error {
	if err := w.BaseWorker.Init(ctx); err != nil {
		return err
	}

	// Create LLM model using factory
	factory := llm.NewModelFactory(ctx, w.cfg)
	llmModel, err := factory.CreateModel(ctx)
	if err != nil {
		return fmt.Errorf("failed to create LLM model: %w", err)
	}
	w.llmModel = llmModel

	w.Logger().Info("synthesis worker initialized",
		"provider", w.cfg.LLMProvider)
	return nil
}

// Execute extracts statistics from the provided search results.
func (w *Worker) Execute(ctx context.Context, req *worker.Request) (*worker.Response, error) {
	// Extract input parameters
	topic, _ := req.Input["topic"].(string)
	if topic == "" {
		return nil, worker.NewValidationError("topic is required")
	}

	searchResultsRaw, ok := req.Input["search_results"]
	if !ok {
		return nil, worker.NewValidationError("search_results is required")
	}

	// Convert search results
	searchResults, err := convertSearchResults(searchResultsRaw)
	if err != nil {
		return nil, worker.NewValidationError(fmt.Sprintf("invalid search_results: %v", err))
	}

	minStats := 5
	if n, ok := req.Input["min_statistics"].(float64); ok {
		minStats = int(n)
	}

	maxStats := 20
	if n, ok := req.Input["max_statistics"].(float64); ok {
		maxStats = int(n)
	}

	w.Logger().Info("synthesizing statistics",
		"topic", topic,
		"sources", len(searchResults),
		"min", minStats,
		"max", maxStats)

	// Process search results
	candidates := make([]models.CandidateStatistic, 0)
	pagesProcessed := 0
	minPagesToProcess := 15

	for _, result := range searchResults {
		if len(candidates) >= maxStats && pagesProcessed >= minPagesToProcess {
			break
		}

		// Fetch webpage content
		content, err := w.fetchURL(ctx, result.URL)
		if err != nil {
			w.Logger().Warn("failed to fetch URL", "url", result.URL, "error", err)
			continue
		}

		// Extract statistics using LLM
		stats, err := w.extractStatisticsWithLLM(ctx, topic, result, content)
		if err != nil {
			w.Logger().Warn("failed to extract statistics", "url", result.URL, "error", err)
			continue
		}

		pagesProcessed++
		candidates = append(candidates, stats...)

		w.Logger().Debug("extracted statistics",
			"domain", result.Domain,
			"extracted", len(stats),
			"total", len(candidates))

		// Stop early if we have well exceeded the minimum
		if len(candidates) >= minStats*5 && pagesProcessed >= minPagesToProcess {
			break
		}
	}

	w.Logger().Info("synthesis completed",
		"candidates", len(candidates),
		"pages_processed", pagesProcessed)

	return worker.NewResponse(req.ID, map[string]any{
		"candidates":       candidates,
		"topic":            topic,
		"sources_analyzed": pagesProcessed,
		"timestamp":        time.Now(),
	}), nil
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

// extractStatisticsWithLLM uses LLM to extract statistics from content.
func (w *Worker) extractStatisticsWithLLM(ctx context.Context, topic string, result models.SearchResult, content string) ([]models.CandidateStatistic, error) {
	// Truncate content if too long
	maxContentLen := 30000
	if len(content) > maxContentLen {
		content = content[:maxContentLen]
	}

	prompt := fmt.Sprintf(`Analyze the following webpage content and extract ALL numerical statistics related to "%s".

IMPORTANT RULES:
1. Extract EVERY statistic you find, not just one or two. Be thorough and comprehensive.
2. The "value" field MUST be the exact number that appears in the excerpt - do not approximate or round
3. The "excerpt" MUST be a verbatim quote containing the exact number you put in "value"
4. If the excerpt says "1.5°C", the value must be 1.5, not 1
5. If you cannot find an exact number in the text, skip that statistic

For each statistic found, provide:
1. name: A brief descriptive name
2. value: The EXACT numerical value from the text (as a number, not string)
3. unit: The unit of measurement
4. excerpt: The verbatim excerpt from the text containing this EXACT statistic (50-200 characters)

Return valid JSON array. Return empty array [] ONLY if absolutely no statistics are found.

Webpage URL: %s
Domain: %s

Content:
%s

JSON output with ALL statistics:`, topic, result.URL, result.Domain, content)

	// Call LLM
	llmReq := &model.LLMRequest{
		Contents: genai.Text(prompt),
	}

	var response string
	for llmResp, err := range w.llmModel.GenerateContent(ctx, llmReq, false) {
		if err != nil {
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}
		if llmResp.Content != nil && llmResp.Content.Parts != nil {
			for _, part := range llmResp.Content.Parts {
				if part.Text != "" {
					response += part.Text
				}
			}
		}
	}

	// Parse JSON response
	type StatExtraction struct {
		Name    string  `json:"name"`
		Value   float32 `json:"value"`
		Unit    string  `json:"unit"`
		Excerpt string  `json:"excerpt"`
	}

	var extractions []StatExtraction
	if err := json.Unmarshal([]byte(response), &extractions); err != nil {
		// Try to extract JSON from markdown
		response = extractJSONFromMarkdown(response)
		if err := json.Unmarshal([]byte(response), &extractions); err != nil {
			return nil, fmt.Errorf("failed to parse LLM response: %w", err)
		}
	}

	// Convert to CandidateStatistic
	candidates := make([]models.CandidateStatistic, 0, len(extractions))
	for _, ext := range extractions {
		if ext.Value == 0 || ext.Excerpt == "" {
			continue
		}
		candidates = append(candidates, models.CandidateStatistic{
			Name:      ext.Name,
			Value:     ext.Value,
			Unit:      ext.Unit,
			Source:    result.Domain,
			SourceURL: result.URL,
			Excerpt:   ext.Excerpt,
		})
	}

	return candidates, nil
}

// extractJSONFromMarkdown removes markdown code fences from LLM response.
func extractJSONFromMarkdown(response string) string {
	response = strings.TrimSpace(response)

	startIdx := strings.Index(response, "[")
	if startIdx == -1 {
		return response
	}

	endIdx := strings.LastIndex(response, "]")
	if endIdx == -1 || endIdx < startIdx {
		return response
	}

	return strings.TrimSpace(response[startIdx : endIdx+1])
}

// convertSearchResults converts raw input to SearchResult slice.
func convertSearchResults(raw any) ([]models.SearchResult, error) {
	// Handle []any (from JSON unmarshaling)
	if arr, ok := raw.([]any); ok {
		results := make([]models.SearchResult, 0, len(arr))
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			results = append(results, models.SearchResult{
				URL:     getString(m, "url"),
				Title:   getString(m, "title"),
				Snippet: getString(m, "snippet"),
				Domain:  getString(m, "domain"),
			})
		}
		return results, nil
	}

	// Handle []models.SearchResult directly
	if results, ok := raw.([]models.SearchResult); ok {
		return results, nil
	}

	return nil, fmt.Errorf("unsupported type: %T", raw)
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
