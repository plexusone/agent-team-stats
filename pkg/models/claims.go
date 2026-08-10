package models

import (
	"fmt"
	"strings"

	"github.com/plexusone/structured-evaluation/claims"
)

// knownAggregatorDomains lists third-party "stats roundup" sites that repost
// figures without independent reporting or a traceable primary source (often
// AI-generated SEO content). A URL on one of these domains is classified as
// claims.ExternalAggregator regardless of the source label, which defaults to
// claims.ReliabilityLow (auto-reject) rather than the claims.ReliabilityMedium
// (requires-review) that an unrecognized-but-real community source gets.
//
// This is a starting list, not exhaustive — add a domain here whenever a
// verification pass (case-study audits, statistic research) finds one
// producing unverifiable or distorted numbers.
var knownAggregatorDomains = []string{
	"getpanto.ai",
}

// IsKnownAggregatorURL reports whether sourceURL's host matches (or is a
// subdomain of) a known aggregator domain. Exported so callers outside this
// package (the orchestrator's VEAL verification loop) can reject an
// aggregator-sourced candidate before it ever reaches classifySourceType.
func IsKnownAggregatorURL(sourceURL string) bool {
	lower := strings.ToLower(sourceURL)
	for _, domain := range knownAggregatorDomains {
		if strings.Contains(lower, "//"+domain) || strings.Contains(lower, "."+domain) {
			return true
		}
	}
	return false
}

// ToClaimsReport converts OrchestrationResponse to a ClaimsReport.
// This provides a standardized output format compatible with structured-evaluation.
func (r *OrchestrationResponse) ToClaimsReport() *claims.ClaimsReport {
	report := claims.NewClaimsReport("statistics-research")
	report.Metadata.DocumentTitle = fmt.Sprintf("Statistics: %s", r.Topic)
	report.Metadata.GeneratedAt = r.Timestamp

	for i, stat := range r.Statistics {
		claimText := formatStatisticClaim(stat)

		claim := claims.NewClaim(
			fmt.Sprintf("stat-%d", i+1),
			claimText,
			claims.ClaimStatistical,
			claims.Location{Section: "verified-statistics"},
		)

		// Set external validation from URL source
		validation := claims.NewExternalValidation(
			stat.SourceURL,
			classifySourceType(stat.Source, stat.SourceURL),
		)
		validation.External.QuotedText = stat.Excerpt
		validation.External.VerifiedMatch = stat.Verified
		validation.External.Reliability = claims.ReliabilityHigh // Verified sources
		claim.SetValidation(validation)
		claim.SetStatistical(statisticalDetail(stat))

		if stat.Verified {
			claim.Verdict = claims.VerdictVerified
			claim.Rationale = fmt.Sprintf("Excerpt verified in source: %s", stat.Source)
		} else {
			claim.Verdict = claims.VerdictUnverified
			claim.Rationale = "Statistic not verified against source"
		}

		report.AddClaim(*claim)
	}

	report.Finalize()
	return report
}

// ToClaimsReportWithFailures includes both verified and failed verification results.
// This provides a complete audit trail of all statistics that were checked.
func (r *OrchestrationResponse) ToClaimsReportWithFailures(
	failures []VerificationResult,
) *claims.ClaimsReport {
	report := r.ToClaimsReport()

	for i, fail := range failures {
		if fail.Statistic == nil {
			continue
		}
		stat := fail.Statistic
		claimText := formatStatisticClaim(*stat)

		claim := claims.NewClaim(
			fmt.Sprintf("fail-%d", i+1),
			claimText,
			claims.ClaimStatistical,
			claims.Location{Section: "unverified-statistics"},
		)

		validation := claims.NewExternalValidation(
			stat.SourceURL,
			classifySourceType(stat.Source, stat.SourceURL),
		)
		validation.External.QuotedText = stat.Excerpt
		validation.External.VerifiedMatch = false
		validation.External.Reliability = claims.ReliabilityLow // Failed verification
		claim.SetValidation(validation)
		claim.SetStatistical(statisticalDetail(*stat))

		claim.Verdict = claims.VerdictRejected
		claim.Rationale = fail.Reason

		report.AddClaim(*claim)
	}

	report.Finalize()
	return report
}

// ToClaimsReport converts a VerificationResponse to a ClaimsReport.
// This is useful when you want to report on verification results directly.
func (r *VerificationResponse) ToClaimsReport(topic string) *claims.ClaimsReport {
	report := claims.NewClaimsReport("statistics-verification")
	report.Metadata.DocumentTitle = fmt.Sprintf("Verification: %s", topic)
	report.Metadata.GeneratedAt = r.Timestamp

	for i, result := range r.Results {
		if result.Statistic == nil {
			continue
		}
		stat := result.Statistic
		claimText := formatStatisticClaim(*stat)

		claim := claims.NewClaim(
			fmt.Sprintf("verify-%d", i+1),
			claimText,
			claims.ClaimStatistical,
			claims.Location{Section: "verification-results"},
		)

		validation := claims.NewExternalValidation(
			stat.SourceURL,
			classifySourceType(stat.Source, stat.SourceURL),
		)
		validation.External.QuotedText = stat.Excerpt
		validation.External.VerifiedMatch = result.Verified
		claim.SetValidation(validation)
		claim.SetStatistical(statisticalDetail(*stat))

		if result.Verified {
			claim.Verdict = claims.VerdictVerified
			validation.External.Reliability = claims.ReliabilityHigh
			claim.Rationale = fmt.Sprintf("Excerpt found in source: %s", stat.Source)
		} else {
			claim.Verdict = claims.VerdictRejected
			validation.External.Reliability = claims.ReliabilityLow
			claim.Rationale = result.Reason
		}

		report.AddClaim(*claim)
	}

	report.Finalize()
	return report
}

// formatStatisticClaim formats a statistic into a claim text string.
func formatStatisticClaim(stat Statistic) string {
	if stat.Unit != "" {
		return fmt.Sprintf("%s: %.2f %s", stat.Name, stat.Value, stat.Unit)
	}
	return fmt.Sprintf("%s: %.2f", stat.Name, stat.Value)
}

// statisticalDetail converts a Statistic's structured numeric fields into a
// claims.StatisticalDetail, so the value survives conversion instead of only
// existing inside formatStatisticClaim's formatted text.
func statisticalDetail(stat Statistic) *claims.StatisticalDetail {
	detail := claims.NewStatisticalDetail(float64(stat.Value), stat.Unit, stat.Precision)
	if stat.AsOfDate != nil {
		detail = detail.WithAsOfDate(*stat.AsOfDate)
	}
	return detail
}

// classifySourceType maps a source name and URL to claims.ExternalSourceType.
// Returns ExternalReputableVendor for known authoritative sources,
// ExternalAggregator for known stats-roundup/aggregator domains (checked via
// sourceURL, since these sites don't have a consistent source-name label),
// or ExternalCommunity for other general sources.
func classifySourceType(source, sourceURL string) claims.ExternalSourceType {
	if IsKnownAggregatorURL(sourceURL) {
		return claims.ExternalAggregator
	}

	// Common authoritative government/research sources
	switch source {
	case "WHO", "World Health Organization":
		return claims.ExternalReputableVendor
	case "CDC", "Centers for Disease Control":
		return claims.ExternalReputableVendor
	case "NIH", "National Institutes of Health":
		return claims.ExternalReputableVendor
	case "FDA", "Food and Drug Administration":
		return claims.ExternalReputableVendor
	case "EPA", "Environmental Protection Agency":
		return claims.ExternalReputableVendor
	case "Census Bureau", "US Census":
		return claims.ExternalReputableVendor
	case "Bureau of Labor Statistics", "BLS":
		return claims.ExternalReputableVendor
	case "Federal Reserve", "The Fed":
		return claims.ExternalReputableVendor
	case "NASA":
		return claims.ExternalReputableVendor
	case "NOAA":
		return claims.ExternalReputableVendor
	case "Pew Research Center", "Pew":
		return claims.ExternalReputableVendor
	case "Gallup":
		return claims.ExternalReputableVendor
	default:
		// General/unknown sources
		return claims.ExternalCommunity
	}
}
