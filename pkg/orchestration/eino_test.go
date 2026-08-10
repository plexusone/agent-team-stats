package orchestration

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/plexusone/agent-team-stats/pkg/models"
)

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestClassifyRejection(t *testing.T) {
	tests := []struct {
		name         string
		result       models.VerificationResult
		isAggregator bool
		expected     rejectionReason
	}{
		{
			name:         "aggregator wins regardless of reason text",
			result:       models.VerificationResult{Reason: ""},
			isAggregator: true,
			expected:     rejectionAggregatorSource,
		},
		{
			name:         "fetch failure",
			result:       models.VerificationResult{Reason: "Failed to fetch source: timeout"},
			isAggregator: false,
			expected:     rejectionFetchFailed,
		},
		{
			name:         "excerpt not found",
			result:       models.VerificationResult{Reason: "Excerpt not found in source content"},
			isAggregator: false,
			expected:     rejectionExcerptNotFound,
		},
		{
			name:         "excerpt not verified phrasing",
			result:       models.VerificationResult{Reason: "statistic not verified against source"},
			isAggregator: false,
			expected:     rejectionExcerptNotFound,
		},
		{
			name:         "unrecognized reason",
			result:       models.VerificationResult{Reason: "something else entirely"},
			isAggregator: false,
			expected:     rejectionUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyRejection(tt.result, tt.isAggregator)
			if got != tt.expected {
				t.Errorf("classifyRejection() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestVealAct_TransientReasonsResubmitUnchanged(t *testing.T) {
	oa := &EinoOrchestrationAgent{logger: noopLogger()}
	stat := &models.Statistic{Name: "Stat", Value: 1, SourceURL: "https://example.com"}

	for _, reason := range []rejectionReason{rejectionFetchFailed, rejectionExcerptNotFound} {
		t.Run(string(reason), func(t *testing.T) {
			fixed, ok := oa.vealAct(context.Background(), models.VerificationResult{Statistic: stat}, reason)
			if !ok {
				t.Fatalf("expected vealAct to accept a transient reason (%q) for resubmission", reason)
			}
			if fixed.Name != stat.Name || fixed.SourceURL != stat.SourceURL {
				t.Errorf("expected the candidate to be resubmitted unchanged, got %+v", fixed)
			}
		})
	}
}

func TestVealAct_UnknownReasonRejects(t *testing.T) {
	oa := &EinoOrchestrationAgent{logger: noopLogger()}
	stat := &models.Statistic{Name: "Stat", Value: 1, SourceURL: "https://example.com"}

	_, ok := oa.vealAct(context.Background(), models.VerificationResult{Statistic: stat}, rejectionUnknown)
	if ok {
		t.Error("expected vealAct to reject (not fix) an unknown rejection reason")
	}
}

func TestVealAct_NilStatisticRejects(t *testing.T) {
	oa := &EinoOrchestrationAgent{logger: noopLogger()}

	_, ok := oa.vealAct(context.Background(), models.VerificationResult{Statistic: nil}, rejectionFetchFailed)
	if ok {
		t.Error("expected vealAct to reject a result with no Statistic")
	}
}

func TestIsExcludedDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		excluded map[string]bool
		want     bool
	}{
		{"exact match", "getpanto.ai", map[string]bool{"getpanto.ai": true}, true},
		{"subdomain vs bare domain", "blog.getpanto.ai", map[string]bool{"getpanto.ai": true}, true},
		{"bare domain vs excluded subdomain", "getpanto.ai", map[string]bool{"www.getpanto.ai": true}, true},
		{"no match", "github.blog", map[string]bool{"getpanto.ai": true}, false},
		{"empty domain never matches", "", map[string]bool{"getpanto.ai": true}, false},
		{"empty exclusion set never matches", "getpanto.ai", map[string]bool{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExcludedDomain(tt.domain, tt.excluded); got != tt.want {
				t.Errorf("isExcludedDomain(%q, %v) = %v, want %v", tt.domain, tt.excluded, got, tt.want)
			}
		})
	}
}

func TestDomainOf(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://www.getpanto.ai/blog/cursor-ai-statistics", "www.getpanto.ai"},
		{"https://github.blog/news-insights/product-news/", "github.blog"},
		{"not-a-url", "not-a-url"}, // falls back to raw string
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := domainOf(tt.url); got != tt.want {
				t.Errorf("domainOf(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestCandidateFromStatistic(t *testing.T) {
	stat := models.Statistic{
		Name:      "Test stat",
		Value:     42,
		Unit:      "%",
		Precision: "approximate",
		Source:    "Example",
		SourceURL: "https://example.com",
		Excerpt:   "42% of things",
		Verified:  true,
	}

	cand := candidateFromStatistic(stat)

	if cand.Name != stat.Name || cand.Value != stat.Value || cand.Unit != stat.Unit ||
		cand.Precision != stat.Precision || cand.Source != stat.Source ||
		cand.SourceURL != stat.SourceURL || cand.Excerpt != stat.Excerpt {
		t.Errorf("candidateFromStatistic() did not preserve fields: got %+v from %+v", cand, stat)
	}
}

func TestCandidateName(t *testing.T) {
	withStat := models.VerificationResult{Statistic: &models.Statistic{Name: "Named stat"}}
	if got := candidateName(withStat); got != "Named stat" {
		t.Errorf("candidateName() = %q, want %q", got, "Named stat")
	}

	withoutStat := models.VerificationResult{}
	if got := candidateName(withoutStat); got != "unknown" {
		t.Errorf("candidateName() = %q, want %q", got, "unknown")
	}
}

func TestBuildDiscoveryGraph_Compiles(t *testing.T) {
	oa := &EinoOrchestrationAgent{logger: noopLogger()}
	g := oa.buildDiscoveryGraph()
	if g == nil {
		t.Fatal("buildDiscoveryGraph() returned nil")
	}
	if _, err := g.Compile(context.Background()); err != nil {
		t.Errorf("discovery graph failed to compile: %v", err)
	}
}
