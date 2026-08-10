# REAL and VEAL Loops (Eino Orchestrator)

The Eino orchestrator (`pkg/orchestration/eino.go`) implements statistics
research as two composed, bounded loops, following the org's [Loop
Engineering](https://productbuildershq.com/frameworks/loop-engineering)
REAL/VEAL patterns rather than a single linear pipeline with an ad-hoc retry.

This page explains what each loop does, why they're separate, and how they
interact. If you haven't read the Loop Engineering framework, the short
version: **REAL creates, VEAL maintains.** REAL is a mission-driven loop with
a single actor that works toward a goal. VEAL is a state-driven loop that
separates a read-only validator from a fix-it actor, converging a batch of
work to a valid state.

## Why two loops, not one

Earlier versions of this orchestrator had a single linear graph — Research →
Synthesis → Verification → Quality Check → "Retry Research" (a stub that
didn't actually retry anything) → Format Response. That conflated two
different problems:

1. **Finding enough raw material.** "We don't have enough candidate
   statistics yet — go search again" is an open-ended discovery problem. It
   doesn't matter *why* the candidate pool is short; the fix is always
   "search more, ideally excluding what already failed."
2. **Making sure what was found is actually correct.** "This candidate's
   excerpt doesn't verify, or its source is a known low-trust aggregator" is
   a convergence problem. The fix is specific to *why* it failed — a
   transient fetch failure needs a re-fetch; an aggregator-sourced claim
   needs a different source entirely, not a re-fetch of the same bad one.

Separating them means the retry logic can actually be *targeted* — the
"stats aggregator" problem this loop was built to catch (see below) needs a
different response than "the page timed out."

## The REAL loop — discovery

**Mission:** find at least `MinVerifiedStats` verified statistics about
`Topic`.

```
Read      → How many verified statistics exist so far? Which domains
            has VEAL already rejected?
Evaluate  → shortfall = target - verified. Done if shortfall <= 0.
Act       → Run one discovery round: Research (web search, excluding
            rejected domains) → Synthesis (LLM extraction) → hand the
            batch to the VEAL loop before counting anything.
Loop      → Repeat until the mission is complete or realMaxAttempts
            (5) rounds are exhausted.
```

Each round is one invocation of the Eino discovery graph
(`buildDiscoveryGraph`): a small, type-safe, two-node graph (`Research` →
`Synthesis`) compiled once per `Orchestrate()` call and invoked once per REAL
round. This is where Eino's graph model earns its keep — the loop itself is
plain bounded Go, but each round's steps are still compile-time-typed.

If the mission isn't complete after `realMaxAttempts` rounds, the response is
explicitly marked `Partial: true` with the shortfall — never padded to look
complete.

## The VEAL loop — verification

**State to converge:** every candidate in a REAL round's batch should end up
either `Verified` or explicitly rejected with a reason. Nothing lingers in
between.

```
Validate  → Read-only: call the verification agent (re-fetches the URL,
            checks the excerpt appears verbatim) AND check locally
            whether the source URL is a known aggregator domain
            (models.IsKnownAggregatorURL) — a check independent of
            whether the excerpt happens to match.
Evaluate  → GO (verified, not aggregator-sourced) or NO-GO with a
            specific reason: aggregator-source, fetch-failed,
            excerpt-not-found, or unknown.
Act       → Only for fixable reasons, and only if attempts remain:
              - aggregator-source: search specifically for a
                primary-source replacement for *this one claim*,
                excluding the rejected domain.
              - fetch-failed / excerpt-not-found: resubmit unchanged —
                these are often transient, and re-validating is itself
                the fix.
              - unknown: not fixable. Reject.
Loop      → Re-validate the batch (verified candidates plus any
            successful replacements). Repeat up to vealMaxAttempts (3)
            per batch.
```

An aggregator-sourced candidate is rejected **even if its excerpt verifies
verbatim** — a matching excerpt only proves the aggregator page says what was
extracted, not that the underlying number is correct. This is the specific
gap a 2026-08 verifiability audit of published case studies found by hand:
`getpanto.ai`-sourced statistics had verbatim-matching excerpts and still
turned out to include a distorted figure (a "9.6 → 2.4 day" PR-cycle claim
that didn't match the real customer story it was supposedly summarizing) and
one contradicted by an independent market-sizing source. The VEAL loop's
validator step exists specifically so that pattern is caught automatically,
before publication, instead of by a manual review pass after the fact.

Domains that produce an aggregator-source rejection are returned to the REAL
loop, which excludes them from its next discovery round — so a bad source
doesn't just get one claim rejected, it stops being re-surfaced by search
for the rest of that `Orchestrate()` call.

## How they compose

```
Orchestrate()
  for attempt in 1..realMaxAttempts:        ← REAL
    if verified >= target: stop
    batch = discover(excluding: rejectedDomains)
    verified_batch, rejected_domains = vealVerify(batch)   ← VEAL, per round
    verified += verified_batch
    rejectedDomains += rejected_domains
  return verified (Partial: true if still short)
```

VEAL runs once per REAL round, on that round's fresh batch — not once at the
end over everything. This means a bad source discovered in round 1 is
already excluded from round 2's search, rather than only being flagged after
all rounds complete.

## Bounds and escalation

| Loop | Max attempts | What "exhausted" means |
|------|--------------|-------------------------|
| REAL (discovery) | 5 | Return `Partial: true` with whatever was verified — never silently pad the count |
| VEAL (verification) | 3 per batch | Reject the candidate outright — never silently keep an unverified or aggregator-sourced statistic |

Both bounds are intentionally on the low end of the framework's guidance
(REAL: 5–100 for open-ended missions; VEAL: 3–5). Discovery here is narrowly
scoped — one topic, one target count — and verification failures that don't
resolve in a few targeted attempts usually indicate the source genuinely
isn't verifiable, not that one more retry would help.

## Source

`pkg/orchestration/eino.go` — see `Orchestrate` (REAL), `vealVerify` /
`vealAct` (VEAL), and `buildDiscoveryGraph` (the per-round Eino graph).
