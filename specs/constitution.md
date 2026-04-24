<!--
  Sync Impact Report
  Version change: 0.0.0 → 1.0.0
  Modified principles: N/A (initial creation)
  Added sections:
    - I. Migrations Must Pass
    - II. Server Code Requires Real Database Tests
    - III. Frontend Code Requires Playwright Tests
    - IV. Specs Stay In Sync
    - V. Demo Script Stays Current
    - VI. Beads Have Verifiable Acceptance Criteria
    - VII. Session Close Protocol
    - Bead Checklist (quality gates)
    - Development Workflow
  Removed sections: None
  Templates requiring updates:
    - plan-template.md: ✅ Constitution Check section already exists
    - spec-template.md: ✅ Acceptance scenarios format compatible
    - tasks-template.md: ✅ Test task structure compatible
  Follow-up TODOs:
    - Build server/internal/dbtest/ package (testcontainers infrastructure)
    - Build server/internal/dbfactory/ package (test data builders)
    - Set up Playwright in e2e/ directory
    - Add nori init --dev and --demo flags
    - Add nori seed demo command
    - Add /api/test/reset endpoint (dev-only)
-->

# Nori Constitution

## Core Principles

### I. Migrations Must Pass

Every bead that adds or modifies a SQL migration MUST verify the migration
works against a clean database before the bead can be closed.

- Run `make migrate-up` against a fresh PostgreSQL instance (no prior state)
- The up migration MUST succeed without errors
- The down migration MUST succeed without errors
- Migration beads MUST NOT be closed if `migrate-up` fails — the migration
  file must be fixed first, not the database state

**Rationale**: The `root_task_id UUID` vs `VARCHAR(255)` mismatch shipped
because no one ran the migration against a real database. This rule prevents
type mismatches, missing dependencies, and broken foreign keys from reaching
any branch.

### II. Server Code Requires Real Database Tests

All Go service, handler, and repository code MUST have unit tests that run
against a real PostgreSQL instance via testcontainers.

- Use the `server/internal/dbtest/` package for test database lifecycle
  (container startup, connection pooling, transaction isolation)
- Use the `server/internal/dbfactory/` package for test data builders
  (`dbfactory.Recipe()`, `dbfactory.Task()`, `dbfactory.Space()`, etc.)
- Each test gets a transaction that rolls back on cleanup — tests are isolated
- Do NOT mock the database. Mocked DB tests cannot catch schema mismatches,
  constraint violations, or migration bugs
- Target 100% test coverage on new service and repository code
- Handler tests MAY use HTTP-level testing against a real database

**Rationale**: Testcontainers with transaction rollback catches real bugs that
mocks miss. Nori's recipe service had 4,393 lines of tests that did not catch
the UUID/VARCHAR mismatch because they mocked the database.

### III. Frontend Code Requires Playwright Tests

Any meaningful frontend change MUST include new or updated Playwright e2e
tests that exercise the affected behavior. This applies to all commits —
not just bead-tracked work. Bug fixes, refactors, and feature additions all
require test coverage if they change user-visible behavior.

- Tests run against the E2E test account (seeded by the server when
  `E2E_ACCOUNT_ENABLED=true`), which is completely isolated from the admin
  account
- Each test file resets the test account state via `POST /api/test/reset`
  before running (reset-before, not cleanup-after)
- Test files map to user stories: `e2e/recipes/author-recipe.spec.ts`,
  `e2e/recipes/roll-recipe.spec.ts`, etc.
- Acceptance scenarios from spec files become Playwright test cases
- Tests MUST assert on user-visible outcomes (text content, navigation,
  element presence), not implementation details (CSS classes, internal state)
- The `e2e/helpers/` directory provides `reset.ts` (call test reset endpoint)
  and `env.ts` (test user credentials)
- When working on a single task, run only the relevant test files for speed
  (e.g., `npx playwright test e2e/recipes/`)
- Before merging a branch, run the full e2e suite (`npx playwright test`) for
  complete coverage

**Rationale**: The recipe epic shipped 5 UI beads (7.14–7.18) that were all
marked complete, but creating a recipe and adding tasks does not work in the
browser. Playwright tests would have caught this before the beads were closed.

### IV. Specs Stay In Sync

Specification files are the source of truth for what Nori does. They MUST be
updated whenever the system changes.

- When a bead completes or changes a user story, update the corresponding
  `specs/{topic}/spec.md` — mark acceptance scenarios as complete (`[x]`),
  update requirements, note any deviations from the original spec
- When technical architecture changes (new tables, new services, new API
  patterns), update `specs/{topic}/architecture.md`
- When any spec file is added or updated, update `specs/readme.md` to keep
  the index current (descriptions, keywords, done status)
- Spec updates MUST happen in the same commit as the code change, not as a
  separate follow-up task

**Rationale**: Specs that drift from reality are worse than no specs — they
actively mislead. Keeping them in sync with code changes ensures the next
person (or AI agent) reading the spec gets an accurate picture.

### V. Demo Script Stays Current

Any bead that adds a user-facing feature MUST update `server/cmd/seed_demo.go`
to exercise that feature in the demo environment.

- The demo seeder calls real service methods (not raw SQL), so the Go
  compiler catches drift when service signatures change
- The demo environment ("Vollick Woodworks") MUST showcase the latest
  capabilities with realistic data (real product names, realistic times,
  station assignments, partial job completion)
- `nori seed demo` rebuilds the demo from scratch; `nori seed demo --reset`
  wipes and rebuilds
- The demo seeder is NOT a test — it is a curated showcase. Test data uses
  the separate "Test Shop" account

**Rationale**: A demo that doesn't show the latest features is a wasted
investor meeting. Making the seeder a consumer of the service layer means it
breaks at compile time when services change, not at demo time.

### VI. Beads Have Verifiable Acceptance Criteria

Every bead MUST include acceptance criteria that can be mechanically verified.

- Acceptance criteria MUST be specific: "migration 000043 runs successfully
  on a clean database" not "migration works"
- Acceptance criteria MUST reference concrete verification: a test name that
  passes, a CLI command that succeeds, an API response that matches, a
  Playwright test that exercises the flow
- "It works" is NOT acceptance criteria
- Beads that touch multiple layers (backend + frontend) MUST have acceptance
  criteria for each layer

**Rationale**: Beads in the recipe epic were all marked closed with acceptance
criteria like "service implemented and tested" — but the tests were mocked and
the UI was never verified end-to-end. Specific, verifiable criteria prevent
false completion.

### VII. Session Close Protocol

Before ending a work session, ALL of the following MUST be true:

- All Go tests pass (`go test ./...`)
- All Playwright tests pass (if frontend was modified)
- `make migrate-up` succeeds on a clean database (if migrations were modified)
- All code changes are committed and pushed (`git push` — work is NOT done
  until it reaches the remote)
- Beads are updated: completed work is closed, in-progress work has notes
  for the next session
- Beads database is synced (`bd dolt push`)

**Rationale**: Locally committed but unpushed work is effectively lost if the
machine fails. Beads left in stale states mislead the next session about what
is ready to work on.

## Bead Checklist

Every bead MUST pass these quality gates before it can be closed. Use this as
a checklist when reviewing bead completion.

### Migration Beads

- [ ] `make migrate-up` succeeds on clean database
- [ ] `make migrate-down` succeeds (reversible)
- [ ] Go models updated to match new schema
- [ ] `specs/{topic}/architecture.md` updated if schema changed

### Backend Service Beads

- [ ] Service logic implemented
- [ ] `dbtest`-based tests pass against real PostgreSQL (not mocked)
- [ ] API endpoint wired and returns expected responses
- [ ] `specs/{topic}/spec.md` acceptance scenarios marked complete

### Frontend UI Beads

- [ ] Component implemented and renders correctly
- [ ] Playwright test in `e2e/` exercises the user flow
- [ ] `specs/{topic}/spec.md` acceptance scenarios marked complete
- [ ] `seed_demo.go` updated to showcase the new feature

### CLI Beads

- [ ] Command implemented with cobra
- [ ] Unit tests pass
- [ ] `specs/cli.md` updated if new commands added
- [ ] `seed_demo.go` updated if feature is demo-relevant

## Development Workflow

### Test Accounts

Nori maintains two non-interactive seeded accounts for development:

- **Test Shop** (`nori init --dev`): Deterministic test account with minimal
  baseline data. Playwright tests and CLI integration tests run against this
  account. State is reset before each test run via `POST /api/test/reset`.
- **Vollick Woodworks** (`nori seed demo`): Curated demo account with rich,
  realistic data (recipes, jobs, time data, cost summaries). Used for investor
  demos and manual exploration. Rebuilt on demand via `--reset`.

### Test Infrastructure

| Layer | Tool | Location | Runs Against |
|-------|------|----------|-------------|
| Go unit/integration | testcontainers + testify | `*_test.go` (colocated) | Real PostgreSQL |
| Test helpers | dbtest + dbfactory | `server/internal/dbtest/`, `server/internal/dbfactory/` | Real PostgreSQL |
| E2E / UI | Playwright | `e2e/` | Test Shop account |
| Demo seeding | Go service calls | `server/cmd/seed_demo.go` | Demo account |

### Bead Workflow

1. Read the spec (`specs/{topic}/spec.md`) before starting
2. Create bead with verifiable acceptance criteria
3. Implement the feature
4. Write tests (Go unit tests for backend, Playwright for frontend)
5. Update `seed_demo.go` if user-facing
6. Update spec files (mark scenarios complete, note deviations)
7. Verify all quality gates pass
8. Close the bead

## Governance

This constitution supersedes all other development practices for the Nori
project. All beads, commits, and sessions MUST comply with these principles.

### Amendments

- Any team member can propose an amendment by updating this file
- Amendments take effect immediately upon merge to main
- Version follows semantic versioning:
  - MAJOR: Principle removed or fundamentally redefined
  - MINOR: New principle added or existing principle materially expanded
  - PATCH: Clarification, wording fix, or non-semantic refinement
- Amendment commits MUST update the version and last-amended date below

### Compliance

- Bead reviewers MUST verify compliance with the Bead Checklist above
- The `bd preflight` command SHOULD check for common violations
- Non-compliant beads MUST NOT be closed

### Reference

- Runtime development guidance: `CLAUDE.md`
- Spec index: `specs/readme.md`
- Bead commands: `bd prime`

**Version**: 1.1.0 | **Ratified**: 2026-04-23 | **Last Amended**: 2026-04-24
