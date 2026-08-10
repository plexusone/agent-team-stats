// Package orchestration implements the deterministic statistics-research
// workflow as two composed loops, following the org's Loop Engineering
// REAL/VEAL patterns (see productbuildershq.github.io's loop-engineering
// framework doc):
//
//   - REAL (Read -> Evaluate -> Act -> Loop): the outer discovery loop.
//     Mission-driven — "find at least MinVerifiedStats statistics about
//     Topic" — with a single actor per round (search, then extract). It
//     reads how many verified statistics exist so far, evaluates the
//     shortfall against the target, acts by running a discovery round
//     (excluding domains that previous rounds already rejected), and loops
//     until the mission is complete or realMaxAttempts is exhausted.
//
//   - VEAL (Validate -> Evaluate -> Act -> Loop): the inner verification
//     loop, run on each batch the REAL loop produces. A read-only validator
//     (the verification agent, plus a local aggregator-source check) checks
//     each candidate and reports GO/NO-GO with a reason. For fixable NO-GO
//     reasons, a separate actor attempts a targeted correction — never a
//     blind retry — before the batch is re-validated. Bounded by
//     vealMaxAttempts; whatever doesn't converge is explicitly rejected,
//     never silently kept or silently dropped from the count.
//
// REAL creates the candidate pool; VEAL converges it to verified-or-rejected.
// If VEAL rejects enough that the round falls short, that shortfall is what
// drives the next REAL iteration.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"

	"github.com/plexusone/agent-team-stats/pkg/config"
	"github.com/plexusone/agent-team-stats/pkg/httpclient"
	"github.com/plexusone/agent-team-stats/pkg/logging"
	"github.com/plexusone/agent-team-stats/pkg/models"
)

const (
	// realMaxAttempts bounds the REAL discovery loop — how many rounds of
	// (search -> extract) run while short of the verified-stats target.
	// Loop Engineering guidance for REAL is 5-100 for open-ended missions;
	// discovery here is narrowly scoped (one topic, one target count), so
	// this uses the low end.
	realMaxAttempts = 5

	// vealMaxAttempts bounds the VEAL verification loop per candidate
	// batch — how many (validate -> act -> re-validate) cycles run before
	// giving up on whatever hasn't converged. Loop Engineering guidance for
	// VEAL is 3-5: if it can't verify in a few tries, escalate (here:
	// reject the claim rather than publish it unverified).
	vealMaxAttempts = 3
)

// rejectionReason classifies why a VEAL validation step said NO-GO, so the
// actor step can respond with a targeted fix instead of a blind retry.
type rejectionReason string

const (
	rejectionAggregatorSource rejectionReason = "aggregator-source"
	rejectionFetchFailed      rejectionReason = "fetch-failed"
	rejectionExcerptNotFound  rejectionReason = "excerpt-not-found"
	rejectionUnknown          rejectionReason = "unknown"
)

// EinoOrchestrationAgent uses the Eino framework for the deterministic
// discovery step (Research -> Synthesis), with the REAL and VEAL loops
// implemented as bounded Go control flow around it — Eino's graph gives
// compile-time-typed steps within a round; the loops decide how many rounds
// to run and how to react to what a round finds.
type EinoOrchestrationAgent struct {
	cfg    *config.Config
	client *http.Client
	logger *slog.Logger
}

// discoveryInput is the typed input to the discovery graph: the request plus
// the domains this Orchestrate() call has already rejected, so a later REAL
// round doesn't re-surface a source VEAL just rejected.
type discoveryInput struct {
	Request         *models.OrchestrationRequest
	ExcludedDomains map[string]bool
}

// NewEinoOrchestrationAgent creates a new Eino-based orchestration agent.
func NewEinoOrchestrationAgent(cfg *config.Config, logger *slog.Logger) *EinoOrchestrationAgent {
	if logger == nil {
		logger = logging.NewAgentLogger("eino-orchestrator")
	}

	return &EinoOrchestrationAgent{
		cfg:    cfg,
		client: &http.Client{Timeout: time.Duration(cfg.HTTPTimeoutSeconds) * time.Second},
		logger: logger,
	}
}

// buildDiscoveryGraph builds the single-round Eino graph: Research ->
// Synthesis. This is the REAL loop's "Act" step — one graph invocation per
// round, with a fresh exclusion set each time.
func (oa *EinoOrchestrationAgent) buildDiscoveryGraph() *compose.Graph[*discoveryInput, *SynthesisState] {
	g := compose.NewGraph[*discoveryInput, *SynthesisState]()

	const (
		nodeResearch  = "research"
		nodeSynthesis = "synthesis"
	)

	researchLambda := compose.InvokableLambda(func(ctx context.Context, in *discoveryInput) (*ResearchState, error) {
		logger := logging.FromContext(ctx)
		logger.Info("REAL/discovery: research", "topic", in.Request.Topic, "excluded_domains", len(in.ExcludedDomains))

		resp, err := oa.callResearchAgent(ctx, &models.ResearchRequest{
			Topic:         in.Request.Topic,
			MinStatistics: in.Request.MinVerifiedStats,
			MaxStatistics: in.Request.MaxCandidates,
			ReputableOnly: in.Request.ReputableOnly,
		})
		if err != nil {
			return nil, fmt.Errorf("research failed: %w", err)
		}

		filtered := make([]models.SearchResult, 0, len(resp.SearchResults))
		for _, r := range resp.SearchResults {
			if isExcludedDomain(r.Domain, in.ExcludedDomains) {
				logger.Debug("REAL/discovery: excluding previously-rejected domain", "domain", r.Domain)
				continue
			}
			filtered = append(filtered, r)
		}

		logger.Info("REAL/discovery: research completed", "sources", len(filtered), "filtered_out", len(resp.SearchResults)-len(filtered))
		return &ResearchState{Request: in.Request, SearchResults: filtered}, nil
	})
	if err := g.AddLambdaNode(nodeResearch, researchLambda); err != nil {
		oa.logger.Warn("failed to add research node", "error", err)
	}

	synthesisLambda := compose.InvokableLambda(func(ctx context.Context, state *ResearchState) (*SynthesisState, error) {
		logger := logging.FromContext(ctx)
		logger.Info("REAL/discovery: synthesis", "sources", len(state.SearchResults))

		resp, err := oa.callSynthesisAgent(ctx, &models.SynthesisRequest{
			Topic:         state.Request.Topic,
			SearchResults: state.SearchResults,
			MinStatistics: state.Request.MinVerifiedStats,
			MaxStatistics: state.Request.MaxCandidates,
		})
		if err != nil {
			return nil, fmt.Errorf("synthesis failed: %w", err)
		}

		logger.Info("REAL/discovery: synthesis completed", "candidates", len(resp.Candidates))
		return &SynthesisState{Request: state.Request, SearchResults: state.SearchResults, Candidates: resp.Candidates}, nil
	})
	if err := g.AddLambdaNode(nodeSynthesis, synthesisLambda); err != nil {
		oa.logger.Warn("failed to add synthesis node", "error", err)
	}

	_ = g.AddEdge(compose.START, nodeResearch)
	_ = g.AddEdge(nodeResearch, nodeSynthesis)
	_ = g.AddEdge(nodeSynthesis, compose.END)

	return g
}

// Orchestrate runs the REAL discovery loop. Each round produces a batch of
// candidates via the discovery graph, then hands that batch to the VEAL
// verification loop before counting anything toward the target.
func (oa *EinoOrchestrationAgent) Orchestrate(ctx context.Context, req *models.OrchestrationRequest) (*models.OrchestrationResponse, error) {
	ctx = logging.WithLogger(ctx, oa.logger)

	if req.MinVerifiedStats == 0 {
		req.MinVerifiedStats = 10
	}
	if req.MaxCandidates == 0 {
		req.MaxCandidates = 30
	}

	oa.logger.Info("REAL loop: starting", "topic", req.Topic, "target", req.MinVerifiedStats, "max_attempts", realMaxAttempts)

	discoveryGraph := oa.buildDiscoveryGraph()
	compiledDiscovery, err := discoveryGraph.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to compile discovery graph: %w", err)
	}

	var verified []models.Statistic
	totalCandidates := 0
	failedCount := 0
	excludedDomains := map[string]bool{}
	attemptsUsed := 0

	for attempt := 1; attempt <= realMaxAttempts; attempt++ {
		attemptsUsed = attempt
		shortfall := req.MinVerifiedStats - len(verified)
		if shortfall <= 0 {
			oa.logger.Info("REAL loop: target met", "verified", len(verified), "attempts_used", attempt-1)
			break
		}

		oa.logger.Info("REAL loop: round", "attempt", attempt, "max_attempts", realMaxAttempts, "shortfall", shortfall, "excluded_domains", len(excludedDomains))

		synthState, err := compiledDiscovery.Invoke(ctx, &discoveryInput{Request: req, ExcludedDomains: excludedDomains})
		if err != nil {
			oa.logger.Warn("REAL loop: round failed, will retry if attempts remain", "attempt", attempt, "error", err)
			continue
		}
		if len(synthState.Candidates) == 0 {
			oa.logger.Warn("REAL loop: no new candidates this round", "attempt", attempt)
			continue
		}

		totalCandidates += len(synthState.Candidates)

		roundVerified, roundFailed, rejectedDomains := oa.vealVerify(ctx, synthState.Candidates)
		verified = append(verified, roundVerified...)
		failedCount += roundFailed
		for d := range rejectedDomains {
			excludedDomains[d] = true
		}

		oa.logger.Info("REAL loop: round complete", "attempt", attempt, "round_candidates", len(synthState.Candidates), "round_verified", len(roundVerified), "total_verified", len(verified))
	}

	verifiedCount := len(verified)
	isPartial := verifiedCount < req.MinVerifiedStats
	if isPartial {
		oa.logger.Warn("REAL loop: exhausted attempts short of target", "verified", verifiedCount, "target", req.MinVerifiedStats, "attempts_used", attemptsUsed)
	}

	return &models.OrchestrationResponse{
		Topic:           req.Topic,
		Statistics:      verified,
		TotalCandidates: totalCandidates,
		VerifiedCount:   verifiedCount,
		FailedCount:     failedCount,
		Timestamp:       time.Now(),
		Partial:         isPartial,
		TargetCount:     req.MinVerifiedStats,
	}, nil
}

// vealVerify runs the VEAL loop over one batch of candidates: Validate each
// (verification agent + local aggregator check) -> Evaluate GO/NO-GO with a
// reason -> Act (targeted fix for fixable reasons) -> Loop. Returns the
// converged verified statistics, a count of candidates that never converged,
// and the set of domains that produced an aggregator-source rejection (for
// the REAL loop to exclude from its next round).
func (oa *EinoOrchestrationAgent) vealVerify(ctx context.Context, candidates []models.CandidateStatistic) (verified []models.Statistic, failedCount int, rejectedDomains map[string]bool) {
	rejectedDomains = map[string]bool{}
	pending := candidates

	for attempt := 1; attempt <= vealMaxAttempts && len(pending) > 0; attempt++ {
		oa.logger.Info("VEAL loop: validate", "attempt", attempt, "max_attempts", vealMaxAttempts, "candidates", len(pending))

		resp, err := oa.callVerificationAgent(ctx, &models.VerificationRequest{Candidates: pending})
		if err != nil {
			oa.logger.Warn("VEAL loop: validation call failed, rejecting batch", "attempt", attempt, "error", err)
			failedCount += len(pending)
			return verified, failedCount, rejectedDomains
		}

		var retry []models.CandidateStatistic
		for _, result := range resp.Results {
			isAggregator := result.Statistic != nil && models.IsKnownAggregatorURL(result.Statistic.SourceURL)

			if result.Verified && result.Statistic != nil && !isAggregator {
				verified = append(verified, *result.Statistic)
				continue
			}

			reason := classifyRejection(result, isAggregator)
			if reason == rejectionAggregatorSource && result.Statistic != nil {
				rejectedDomains[domainOf(result.Statistic.SourceURL)] = true
			}

			isLastAttempt := attempt == vealMaxAttempts
			if !isLastAttempt {
				if fixed, ok := oa.vealAct(ctx, result, reason); ok {
					retry = append(retry, fixed)
					continue
				}
			}

			// Exhausted attempts, or unfixable: explicit rejection — never
			// silently kept, never silently dropped from the failed count.
			failedCount++
			oa.logger.Info("VEAL loop: rejecting candidate", "name", candidateName(result), "reason", reason, "attempt", attempt)
		}

		pending = retry
	}

	return verified, failedCount, rejectedDomains
}

// vealAct is the VEAL actor: given one NO-GO result and why, attempt a
// correction that matches the specific reason rather than a blind retry.
//   - aggregator-source: search specifically for a primary-source
//     replacement for this one claim, excluding the rejected domain. This
//     is the fix that would have caught the getpanto.ai-sourced stats found
//     in the 2026-08 case-study verifiability audit before publication.
//   - fetch-failed / excerpt-not-found: resubmit unchanged for a fresh
//     validation pass — these are frequently transient (a flaky fetch, a
//     page that briefly failed to load), and re-fetching is exactly what
//     the next VEAL iteration's validator does.
//   - anything else: not fixable; the caller rejects it.
func (oa *EinoOrchestrationAgent) vealAct(ctx context.Context, result models.VerificationResult, reason rejectionReason) (models.CandidateStatistic, bool) {
	if result.Statistic == nil {
		return models.CandidateStatistic{}, false
	}
	original := candidateFromStatistic(*result.Statistic)

	switch reason {
	case rejectionFetchFailed, rejectionExcerptNotFound:
		return original, true
	case rejectionUnknown:
		// Not a reason we know how to respond to — reject rather than
		// blindly retry an unclassified failure.
		return models.CandidateStatistic{}, false
	}
	// reason == rejectionAggregatorSource falls through to the replacement
	// search below.

	rejectedDomain := domainOf(original.SourceURL)
	oa.logger.Info("VEAL loop: act — searching for a primary-source replacement", "name", original.Name, "rejected_domain", rejectedDomain)

	researchResp, err := oa.callResearchAgent(ctx, &models.ResearchRequest{
		Topic:         original.Name,
		MinStatistics: 1,
		MaxStatistics: 5,
		ReputableOnly: true,
	})
	if err != nil {
		oa.logger.Warn("VEAL loop: act — replacement research failed", "name", original.Name, "error", err)
		return models.CandidateStatistic{}, false
	}

	filtered := make([]models.SearchResult, 0, len(researchResp.SearchResults))
	for _, r := range researchResp.SearchResults {
		if isExcludedDomain(r.Domain, map[string]bool{rejectedDomain: true}) {
			continue
		}
		filtered = append(filtered, r)
	}
	if len(filtered) == 0 {
		return models.CandidateStatistic{}, false
	}

	synthResp, err := oa.callSynthesisAgent(ctx, &models.SynthesisRequest{
		Topic:         original.Name,
		SearchResults: filtered,
		MinStatistics: 1,
		MaxStatistics: 1,
	})
	if err != nil || len(synthResp.Candidates) == 0 {
		if err != nil {
			oa.logger.Warn("VEAL loop: act — replacement synthesis failed", "name", original.Name, "error", err)
		}
		return models.CandidateStatistic{}, false
	}

	return synthResp.Candidates[0], true
}

// classifyRejection turns a verification result into a rejectionReason the
// actor can respond to specifically.
func classifyRejection(result models.VerificationResult, isAggregator bool) rejectionReason {
	if isAggregator {
		return rejectionAggregatorSource
	}
	lower := strings.ToLower(result.Reason)
	switch {
	case strings.Contains(lower, "fetch"):
		return rejectionFetchFailed
	case strings.Contains(lower, "not found"), strings.Contains(lower, "not verified"):
		return rejectionExcerptNotFound
	default:
		return rejectionUnknown
	}
}

// isExcludedDomain reports whether domain matches any excluded domain,
// tolerant of "www." and subdomain formatting differences between what a
// search API's DisplayLink returns and a URL host parsed elsewhere.
func isExcludedDomain(domain string, excluded map[string]bool) bool {
	d := strings.ToLower(domain)
	for ex := range excluded {
		exLower := strings.ToLower(ex)
		if d == "" || exLower == "" {
			continue
		}
		if strings.Contains(d, exLower) || strings.Contains(exLower, d) {
			return true
		}
	}
	return false
}

// domainOf extracts the host from a URL for exclusion-set bookkeeping,
// falling back to the raw string if it doesn't parse.
func domainOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" {
		return rawURL
	}
	return u.Hostname()
}

func candidateFromStatistic(s models.Statistic) models.CandidateStatistic {
	return models.CandidateStatistic{
		Name:      s.Name,
		Value:     s.Value,
		Unit:      s.Unit,
		Precision: s.Precision,
		Source:    s.Source,
		SourceURL: s.SourceURL,
		Excerpt:   s.Excerpt,
		AsOfDate:  s.AsOfDate,
	}
}

func candidateName(result models.VerificationResult) string {
	if result.Statistic != nil {
		return result.Statistic.Name
	}
	return "unknown"
}

// Helper methods to call research, synthesis, and verification agents.

func (oa *EinoOrchestrationAgent) callResearchAgent(ctx context.Context, req *models.ResearchRequest) (*models.ResearchResponse, error) {
	var resp models.ResearchResponse
	url := fmt.Sprintf("%s/research", oa.cfg.ResearchAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (oa *EinoOrchestrationAgent) callSynthesisAgent(ctx context.Context, req *models.SynthesisRequest) (*models.SynthesisResponse, error) {
	var resp models.SynthesisResponse
	url := fmt.Sprintf("%s/synthesize", oa.cfg.SynthesisAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (oa *EinoOrchestrationAgent) callVerificationAgent(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error) {
	var resp models.VerificationResponse
	url := fmt.Sprintf("%s/verify", oa.cfg.VerificationAgentURL)
	if err := httpclient.PostJSON(ctx, oa.client, url, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// HandleOrchestrationRequest is the HTTP handler for the orchestration
// endpoint.
func (oa *EinoOrchestrationAgent) HandleOrchestrationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.OrchestrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := oa.Orchestrate(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Orchestration failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		oa.logger.Error("failed to encode response", "error", err)
	}
}

// State types for the discovery graph.
type ResearchState struct {
	Request       *models.OrchestrationRequest
	SearchResults []models.SearchResult
}

type SynthesisState struct {
	Request       *models.OrchestrationRequest
	SearchResults []models.SearchResult
	Candidates    []models.CandidateStatistic
}
