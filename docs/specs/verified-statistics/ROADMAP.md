# Verified Statistics — Evidence-Gated Claim Verification — Roadmap

**Initiative:** `INIT-AGENTTEAMSTATS-001`
**Repository:** `github.com/plexusone/agent-team-stats`
**Status:** Phase 1 in progress

> RMI IDs are stable and permanent. Commits implementing an item carry the trailer `Refs: RMI-<REPOSLUG>-<NNN>`. Phase status is derived from member RMIs — a phase is complete only when all its required RMIs are complete.

## Phase 1 — structured-evaluation: Evidence-Gated Verdicts (v0.13.0)

**Theme:** Tighten what earns 'verified': quote-with-value lint, source-role, corroboration, and staleness — additive/non-breaking (strictness in a new lint + criteria knobs, not DetermineVerdict yet). Ships as v0.13.0.
**Status:** In progress — 1 of 5 items completed

- [x] `RMI-AGENTTEAMSTATS-001` claims.Lint + 'sevaluation lint' for claims reports
  - Add claims.Lint() and wire 'sevaluation lint <claims.json>'. Flags any verified claim lacking a resolving URL + quotedText, and where StatisticalDetail.Value does not appear in the quote. Advisory/gating validation, separate from DetermineVerdict (non-breaking). This is the exact check that would have caught the '$3B ARR' false positive.
- [ ] `RMI-AGENTTEAMSTATS-002` Add sourceRole to ExternalValidation
  - New sourceRole enum (primary | secondary-relay | secondary-analysis | self-reported) on ExternalValidation; regenerate claims.schema.json. Lint caps self-reported and secondary-analysis at needs-review unless corroborated.
- [ ] `RMI-AGENTTEAMSTATS-003` Corroboration criteria + relatedClaimIds enforcement
  - minCorroboratingSources knob in ClaimsCriteria; lint/EvaluateClaims require N corroborating sources (via relatedClaimIds) for high-stakes categories. Single reputable source -> needs-review when threshold > 1.
- [ ] `RMI-AGENTTEAMSTATS-004` Codify staleness (asOfDate age -> needs-review)
  - maxClaimAge criteria knob; lint flags claims whose asOfDate exceeds the threshold and are presented as current, downgrading to needs-review (e.g. the 2022-23 Copilot study, 2023 Spotify BOM figure).
- [ ] `RMI-AGENTTEAMSTATS-005` Fix accessedAt/validatedAt omitempty; cut v0.13.0
  - Make ExternalValidation.AccessedAt and InternalValidation.ValidatedAt *time.Time so omitempty works (retires the 0001-01-01 zero-value serialization). Tag and release v0.13.0 after CI is green.

## Phase 2 — agent-team-stats: Evidence Producer Hardening

**Theme:** Make the producer supply real evidence: rendered-text fetch + numeric match + source archiving, populate source-role, gather and link corroborating sources, and detect aggregators beyond a hardcoded list. Consumes structured-evaluation v0.13.0.
**Status:** Planned — 0 of 5 items completed

- [ ] `RMI-AGENTTEAMSTATS-006` Bump to structured-evaluation v0.13.0; wire lint into CI
  - go get structured-evaluation@v0.13.0 and go mod tidy. Add 'sevaluation lint' over produced ClaimsReports to CI so a verified-without-quote claim fails the build.
- [ ] `RMI-AGENTTEAMSTATS-007` Verification worker: rendered fetch + numeric match + archive
  - Replace strings.Contains(rawHTML, excerpt) with headless-rendered text (reuse the dss-render already in the PDF pipeline); confirm the numeric value, not just a substring; populate ExternalValidation.archiveUrl with a snapshot. Treat 403/paywall/JS-only pages as can't-verify -> needs-review, never a silent pass.
- [ ] `RMI-AGENTTEAMSTATS-008` Populate sourceRole in classifySourceType
  - classifySourceType sets sourceRole (primary vs secondary-relay vs secondary-analysis vs self-reported) on every external validation, so structured-evaluation's gate can act on it. Company self-pages (e.g. cursor.com/enterprise) tag self-reported.
- [ ] `RMI-AGENTTEAMSTATS-009` Corroboration in REAL/VEAL; link relatedClaimIds
  - Discovery/verification gathers >= N independent sources per numeric claim and links them via relatedClaimIds, so structured-evaluation's corroboration criteria can pass instead of relying on a single source.
- [ ] `RMI-AGENTTEAMSTATS-010` Aggregator detection beyond hardcoded list
  - Heuristic aggregator detection (listicle / 'Statistics 20XX' patterns, no primary attribution, cross-citing-only domains) augmenting knownAggregatorDomains, so a new aggregator is caught without a manual list edit.
