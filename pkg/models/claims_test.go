package models

import (
	"testing"
	"time"

	"github.com/plexusone/structured-evaluation/claims"
)

// assertLintClean runs claims.Lint over report and fails the test with the
// full finding list if any claim marked verified doesn't actually earn the
// label (missing quote/URL, insufficient corroboration, etc.) — the same
// gate 'sevaluation lint' applies in CI to the case-study ledgers this
// pipeline's output feeds. Wired into every conversion test below so a
// regression in ToClaimsReport/ToClaimsReportWithFailures/
// VerificationResponse.ToClaimsReport that stops setting QuotedText or a URL
// on a verified claim fails go test, not just a manual audit.
func assertLintClean(t *testing.T, report *claims.ClaimsReport) {
	t.Helper()
	findings := claims.Lint(report)
	if claims.HasErrors(findings) {
		t.Errorf("claims.Lint found %d finding(s), including errors:", len(findings))
		for _, f := range findings {
			t.Errorf("  [%s] %s: %s", f.Severity, f.ClaimID, f.Message)
		}
	}
}

func TestOrchestrationResponse_ToClaimsReport(t *testing.T) {
	now := time.Now()
	resp := &OrchestrationResponse{
		Topic: "climate change statistics",
		Statistics: []Statistic{
			{
				Name:      "Global temperature increase",
				Value:     1.1,
				Unit:      "°C",
				Source:    "NASA",
				SourceURL: "https://climate.nasa.gov/vital-signs/global-temperature/",
				Excerpt:   "Global temperature has increased by 1.1°C since pre-industrial times",
				Verified:  true,
				DateFound: now,
			},
			{
				Name:      "Sea level rise",
				Value:     3.3,
				Unit:      "mm/year",
				Source:    "NOAA",
				SourceURL: "https://www.noaa.gov/sea-level",
				Excerpt:   "Sea level is rising at 3.3mm per year",
				Verified:  true,
				DateFound: now,
			},
		},
		TotalCandidates: 5,
		VerifiedCount:   2,
		FailedCount:     3,
		Timestamp:       now,
	}

	report := resp.ToClaimsReport()

	// Verify report metadata
	if report.Metadata.DocumentTitle != "Statistics: climate change statistics" {
		t.Errorf("expected title 'Statistics: climate change statistics', got %q", report.Metadata.DocumentTitle)
	}

	// Verify claims count
	if len(report.Claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(report.Claims))
	}

	// Verify first claim
	claim1 := report.Claims[0]
	if claim1.ID != "stat-1" {
		t.Errorf("expected claim ID 'stat-1', got %q", claim1.ID)
	}
	if claim1.Category != claims.ClaimStatistical {
		t.Errorf("expected category ClaimStatistical, got %v", claim1.Category)
	}
	if claim1.Verdict != claims.VerdictVerified {
		t.Errorf("expected verdict VerdictVerified, got %v", claim1.Verdict)
	}
	if claim1.Validation == nil || claim1.Validation.External == nil {
		t.Fatal("expected external validation to be set")
	}
	if claim1.Validation.External.URL != "https://climate.nasa.gov/vital-signs/global-temperature/" {
		t.Errorf("expected URL 'https://climate.nasa.gov/vital-signs/global-temperature/', got %q", claim1.Validation.External.URL)
	}
	if !claim1.Validation.External.VerifiedMatch {
		t.Error("expected VerifiedMatch to be true")
	}
	if claim1.Validation.External.Reliability != claims.ReliabilityHigh {
		t.Errorf("expected reliability ReliabilityHigh, got %v", claim1.Validation.External.Reliability)
	}
	if claim1.Statistical == nil {
		t.Fatal("expected Statistical detail to be set")
	}
	// Value round-trips through the source Statistic's float32, so compare
	// with a tolerance rather than exact float64 equality.
	if diff := claim1.Statistical.Value - 1.1; diff > 0.0001 || diff < -0.0001 {
		t.Errorf("expected Statistical.Value ~1.1, got %v", claim1.Statistical.Value)
	}
	if claim1.Statistical.Unit != "°C" {
		t.Errorf("expected Statistical.Unit °C, got %q", claim1.Statistical.Unit)
	}

	assertLintClean(t, report)
}

func TestOrchestrationResponse_ToClaimsReport_PrecisionAndAsOfDate(t *testing.T) {
	asOf := time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC)
	resp := &OrchestrationResponse{
		Topic: "example",
		Statistics: []Statistic{
			{
				Name:      "Approximate figure",
				Value:     1000000,
				Unit:      "users",
				Precision: claims.PrecisionApproximate,
				Source:    "Example",
				SourceURL: "https://example.com",
				Excerpt:   "1M+ users",
				Verified:  true,
				DateFound: time.Now(),
				AsOfDate:  &asOf,
			},
		},
	}

	report := resp.ToClaimsReport()
	if len(report.Claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(report.Claims))
	}

	stat := report.Claims[0].Statistical
	if stat == nil {
		t.Fatal("expected Statistical detail to be set")
	}
	if stat.Precision != claims.PrecisionApproximate {
		t.Errorf("expected precision approximate, got %q", stat.Precision)
	}
	if stat.AsOfDate == nil || !stat.AsOfDate.Equal(asOf) {
		t.Errorf("expected AsOfDate %v, got %v", asOf, stat.AsOfDate)
	}

	assertLintClean(t, report)
}

func TestParseAsOfDate(t *testing.T) {
	t.Run("empty string returns nil, no error", func(t *testing.T) {
		got, err := ParseAsOfDate("")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("valid date parses", func(t *testing.T) {
		got, err := ParseAsOfDate("2026-01-26")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC)
		if got == nil || !got.Equal(want) {
			t.Errorf("expected %v, got %v", want, got)
		}
	})

	t.Run("malformed date returns an error, not a guess", func(t *testing.T) {
		got, err := ParseAsOfDate("not-a-date")
		if err == nil {
			t.Error("expected an error for malformed date")
		}
		if got != nil {
			t.Errorf("expected nil on error, got %v", got)
		}
	})
}

func TestOrchestrationResponse_ToClaimsReportWithFailures(t *testing.T) {
	now := time.Now()
	resp := &OrchestrationResponse{
		Topic: "test topic",
		Statistics: []Statistic{
			{
				Name:      "Verified stat",
				Value:     100,
				Unit:      "%",
				Source:    "Test Source",
				SourceURL: "https://example.com/verified",
				Excerpt:   "This is 100% verified",
				Verified:  true,
				DateFound: now,
			},
		},
		TotalCandidates: 2,
		VerifiedCount:   1,
		FailedCount:     1,
		Timestamp:       now,
	}

	failures := []VerificationResult{
		{
			Statistic: &Statistic{
				Name:      "Failed stat",
				Value:     50,
				Unit:      "%",
				Source:    "Bad Source",
				SourceURL: "https://example.com/failed",
				Excerpt:   "This excerpt was not found",
				Verified:  false,
				DateFound: now,
			},
			Verified: false,
			Reason:   "Excerpt not found in source content",
		},
	}

	report := resp.ToClaimsReportWithFailures(failures)

	// Should have 2 claims (1 verified + 1 failed)
	if len(report.Claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(report.Claims))
	}

	// Find the failed claim
	var failedClaim *claims.Claim
	for i := range report.Claims {
		if report.Claims[i].ID == "fail-1" {
			failedClaim = &report.Claims[i]
			break
		}
	}

	if failedClaim == nil {
		t.Fatal("expected to find failed claim with ID 'fail-1'")
	}

	if failedClaim.Verdict != claims.VerdictRejected {
		t.Errorf("expected verdict VerdictRejected, got %v", failedClaim.Verdict)
	}
	if failedClaim.Rationale != "Excerpt not found in source content" {
		t.Errorf("expected rationale 'Excerpt not found in source content', got %q", failedClaim.Rationale)
	}

	assertLintClean(t, report)
}

func TestVerificationResponse_ToClaimsReport(t *testing.T) {
	now := time.Now()
	resp := &VerificationResponse{
		Results: []VerificationResult{
			{
				Statistic: &Statistic{
					Name:      "Test stat",
					Value:     42,
					Unit:      "units",
					Source:    "Test Source",
					SourceURL: "https://example.com/test",
					Excerpt:   "The value is 42 units",
					Verified:  true,
					DateFound: now,
				},
				Verified: true,
				Reason:   "",
			},
			{
				Statistic: &Statistic{
					Name:      "Bad stat",
					Value:     99,
					Unit:      "",
					Source:    "Bad Source",
					SourceURL: "https://example.com/bad",
					Excerpt:   "Not found",
					Verified:  false,
					DateFound: now,
				},
				Verified: false,
				Reason:   "Excerpt not found",
			},
		},
		Verified:  1,
		Failed:    1,
		Timestamp: now,
	}

	report := resp.ToClaimsReport("test verification")

	if report.Metadata.DocumentTitle != "Verification: test verification" {
		t.Errorf("expected title 'Verification: test verification', got %q", report.Metadata.DocumentTitle)
	}

	if len(report.Claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(report.Claims))
	}

	// Check verified claim
	if report.Claims[0].Verdict != claims.VerdictVerified {
		t.Errorf("expected first claim to be verified, got %v", report.Claims[0].Verdict)
	}

	// Check rejected claim
	if report.Claims[1].Verdict != claims.VerdictRejected {
		t.Errorf("expected second claim to be rejected, got %v", report.Claims[1].Verdict)
	}

	assertLintClean(t, report)
}

// TestClaimsLint_CatchesMissingQuoteOnVerifiedClaim demonstrates that the
// gate assertLintClean applies actually works: a verified claim that loses
// its QuotedText (the invariant ToClaimsReport currently upholds) must be
// caught, not silently pass. This is the exact shape of the false positive
// (a hand-authored "verified" claim with no verbatim excerpt) that
// motivated wiring claims.Lint into this pipeline's tests in the first
// place — see structured-evaluation's RMI-AGENTTEAMSTATS-001.
func TestClaimsLint_CatchesMissingQuoteOnVerifiedClaim(t *testing.T) {
	resp := &OrchestrationResponse{
		Topic: "regression check",
		Statistics: []Statistic{
			{
				Name:      "Broken claim",
				Value:     42,
				Unit:      "units",
				Source:    "Example",
				SourceURL: "https://example.com",
				Excerpt:   "The value is 42 units",
				Verified:  true,
				DateFound: time.Now(),
			},
		},
	}
	report := resp.ToClaimsReport()
	report.Claims[0].Validation.External.QuotedText = "" // simulate the regression

	findings := claims.Lint(report)
	if !claims.HasErrors(findings) {
		t.Fatal("expected claims.Lint to flag a verified claim with no QuotedText, got no errors")
	}
}

func TestFormatStatisticClaim(t *testing.T) {
	tests := []struct {
		name     string
		stat     Statistic
		expected string
	}{
		{
			name: "with unit",
			stat: Statistic{
				Name:  "Temperature",
				Value: 25.5,
				Unit:  "°C",
			},
			expected: "Temperature: 25.50 °C",
		},
		{
			name: "without unit",
			stat: Statistic{
				Name:  "Count",
				Value: 100,
				Unit:  "",
			},
			expected: "Count: 100.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatisticClaim(tt.stat)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestClassifySourceType(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		url      string
		expected claims.ExternalSourceType
	}{
		{"WHO by name", "WHO", "", claims.ExternalReputableVendor},
		{"World Health Organization by name", "World Health Organization", "", claims.ExternalReputableVendor},
		{"CDC by name", "CDC", "", claims.ExternalReputableVendor},
		{"NIH by name", "NIH", "", claims.ExternalReputableVendor},
		{"unrecognized source falls back to community", "Random Blog", "https://random-blog.example.com/post", claims.ExternalCommunity},
		{"Pew Research Center by name", "Pew Research Center", "", claims.ExternalReputableVendor},
		{"known aggregator domain, plain source label", "Cursor AI Statistics 2026", "https://www.getpanto.ai/blog/cursor-ai-statistics", claims.ExternalAggregator},
		{"known aggregator domain, subdomain", "GitHub Copilot Statistics 2026", "https://blog.getpanto.ai/github-copilot-statistics", claims.ExternalAggregator},
		{"aggregator match overrides an authoritative-sounding label", "WHO", "https://getpanto.ai/who-stats", claims.ExternalAggregator},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifySourceType(tt.source, tt.url)
			if result != tt.expected {
				t.Errorf("classifySourceType(%q, %q) = %v, expected %v", tt.source, tt.url, result, tt.expected)
			}
		})
	}
}
