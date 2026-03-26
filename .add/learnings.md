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

## Architecture Decisions
<!-- Decisions made and their rationale -->
<!-- Format: - {date}: Chose {X} over {Y} because {reason}. -->

- 2026-03-25: Chose Go 1.22+ over Python/FastAPI for consistency with drug-cash and drug-gate stack.
- 2026-03-25: Chose Chi v5 router over net/http for middleware support and consistency with drug-gate.
- 2026-03-25: Chose PostgreSQL 16+ with GIN indexes for full-text search over Elasticsearch — simpler ops, sufficient for ~140K products.

## What Worked
<!-- Patterns, approaches, tools that proved effective -->

## What Didn't Work
<!-- Patterns, approaches, tools that caused problems -->

## Agent Checkpoints
<!-- Automatic entries from verification, TDD cycles, deploys, away sessions -->
<!-- These are processed and archived during /add:retro -->

## Profile Update Candidates
<!-- Cross-project patterns flagged for promotion to ~/.claude/add/profile.md -->
<!-- Only promoted during /add:retro with human confirmation -->
