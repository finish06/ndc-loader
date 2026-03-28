# Project Learnings — ndc-loader

> **Tier 3: Project-Specific Knowledge**
>
> This file is maintained automatically by ADD agents. Entries are added at checkpoints
> (after verify, TDD cycles, deployments, away sessions) and reviewed during retrospectives.
>
> This is one of three knowledge tiers agents read before starting work:
> 1. **Tier 1: Plugin-Global** (`knowledge/global.md`) — universal ADD best practices
> 2. **Tier 2: User-Local** (`~/.claude/add/library.md`) — your cross-project wisdom
> 3. **Tier 3: Project-Specific** (this file) — discoveries specific to this project
>
> **Agents:** Read ALL three tiers before starting any task.
> **Humans:** Review with `/add:retro --agent-summary` or during full `/add:retro`.

## Technical Discoveries
<!-- Things learned about the tech stack, libraries, APIs, infrastructure -->
<!-- Format: - {date}: {discovery}. Source: {how we learned this}. -->

- 2026-03-25: Go's csv.Reader with `TrimLeadingSpace=true` collapses empty fields between consecutive tab delimiters. Remove TrimLeadingSpace when parsing tab-delimited FDA data. Source: integration test failure — empty DATE fields (ENDMARKETINGDATE, DEASCHEDULE) were being consumed.
- 2026-03-25: pgx CopyFrom sends values in binary format. String values cannot be inserted into DATE or BOOLEAN columns — must convert to `time.Time` and `bool` before COPY. Source: integration test failure — "unable to encode into binary format for date".
- 2026-03-25: FDA NDC Directory uses YYYYMMDD date format (e.g., "19950301"). Drugs@FDA uses the same format. Both need parsing to time.Time for PostgreSQL DATE columns.
- 2026-03-25: FDA NDC Directory uses "Y"/"N" for boolean fields (NDC_EXCLUDE_FLAG, SAMPLE_PACKAGE). Need explicit bool conversion.
- 2026-03-26: FDA bulk data contains Windows-1252 encoded bytes (0x92 right single quote, 0xbf inverted question mark). Must sanitize to valid UTF-8 before PostgreSQL COPY. Source: E2E test with real FDA data — products.txt and Submissions.txt both affected.
- 2026-03-26: FDA datasets have orphaned FK references between NDC Directory and Drugs@FDA (different source systems). FK constraints must be removed — use application-level joins instead. Source: E2E load failure — drugsfda_products references appl_no values not in applications table.
- 2026-03-26: NDC application_number format is "ANDA076543" (type prefix + number), but Drugs@FDA appl_no is "076543" (number only, zero-padded to 6 digits). Join requires `regexp_replace` + `LPAD`. Source: E2E join validation returned 0 matches until normalized.
- 2026-03-26: Real FDA NDC file has PROPRIETARYNAMESUFFIX column not in openFDA API docs. Drugs@FDA Applications.txt uses SponsorName (not SponsorApplicant), Submissions has SubmissionClassCodeID and ReviewPriority. Always validate against real data, not API docs alone.
- 2026-03-26: Drugs@FDA ActionTypes_Lookup.txt is NOT active ingredients — it's a lookup table for submission action types. Active ingredients are a column in Products.txt. TE.txt (not TECodes.txt) has MarketingStatusID.

## Architecture Decisions
<!-- Decisions made and their rationale -->
<!-- Format: - {date}: Chose {X} over {Y} because {reason}. -->

- 2026-03-25: Chose Go 1.22+ over Python/FastAPI for consistency with drug-cash and drug-gate stack.
- 2026-03-25: Chose Chi v5 router over net/http for middleware support and consistency with drug-gate.
- 2026-03-25: Chose PostgreSQL 16+ with GIN indexes for full-text search over Elasticsearch — simpler ops, sufficient for ~140K products.

## What Worked
<!-- Patterns, approaches, tools that proved effective -->

- 2026-03-26: Interface-based mocking for orchestrator/store layers was essential for 90%+ unit test coverage without a real database. Define interfaces in the consumer package (loader), implement in the provider package (store).
- 2026-03-26: E2E testing with real FDA data caught 5 bugs that unit tests with fixture data missed (encoding, FK orphans, schema differences, format mismatches, wrong file mapping).
- 2026-03-26: Atomic swap via staging tables works well for bulk data — no downtime, no partial reads.

## What Didn't Work
<!-- Patterns, approaches, tools that caused problems -->

- 2026-03-26: Spec assumed FDA data structure from API docs — real bulk download files have different columns, naming, and relationships. Lesson: always download and inspect real data before finalizing schema.
- 2026-03-26: FK constraints between FDA datasets are unreliable. Cross-system bulk data should use soft references (indexed columns) not hard FKs.

## Agent Checkpoints
<!-- Automatic entries from verification, TDD cycles, deploys, away sessions -->
<!-- These are processed and archived during /add:retro -->

- 2026-03-26: Cycle 2 complete. E2E FDA validation + Query API. 2 features VERIFIED, 5 bugs fixed, 11 E2E tests. M1 closed with 10/10 criteria met.
- 2026-03-27: Cycle 3 complete. openFDA-compat API for drug-cash. 14/14 AC, format parity verified against live openFDA API, 9 E2E + 22 unit tests. M2 closed with 6/6 criteria met.
- 2026-03-27: Performance optimizations applied: streaming parser (single-pass), O(1) table swap (DROP+RENAME vs INSERT SELECT), batch package loading (eliminates N+1 queries).

## Profile Update Candidates
<!-- Cross-project patterns flagged for promotion to ~/.claude/add/profile.md -->
<!-- Only promoted during /add:retro with human confirmation -->
