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

Every bead MUST include acceptance criteria **at creation time**. Acceptance
criteria are part of the bead definition, not something added after the fact.

- Acceptance criteria MUST be specific: "migration 000043 runs successfully
  on a clean database" not "migration works"
- Acceptance criteria MUST reference concrete verification: a test name that
  passes, a CLI command that succeeds, an API response that matches, a
  Playwright test that exercises the flow
- "It works" is NOT acceptance criteria
- Beads that touch multiple layers (backend + frontend) MUST have acceptance
  criteria for each layer
- **Layer-specific test requirements in acceptance criteria:**
  - Migration beads: `make migrate-up` succeeds on clean database
  - Backend beads: Go unit tests added/updated and passing
  - Frontend beads: Playwright e2e tests added/updated and passing

**Rationale**: Beads in the recipe epic were all marked closed with acceptance
criteria like "service implemented and tested" — but the tests were mocked and
the UI was never verified end-to-end. Beads were also created without
acceptance criteria and had them retrofitted later, which meant the criteria
were written to match what was already built rather than defining what done
looks like. Writing criteria at creation time forces clarity about what the
bead actually needs to deliver.

### VIII. Beads Are the Planning and Tracking System

All work MUST be planned and tracked through beads. Beads are not just a
record of what happened — they are the plan for what will happen.

- Use beads to plan work BEFORE writing code, not just to track it after
- Create an **epic** for any multi-step effort
- Create child beads under the epic using the **epic's prefix** as a
  namespace (e.g., `nori-abc.1`, `nori-abc.2` for epic `nori-abc`). This
  makes `bv` search, filtering, and graph traversal work correctly
- Every bead MUST have acceptance criteria at creation time (see VI above)
- **A bead MUST NOT be closed until all tests in its acceptance criteria are
  passing.** Run the relevant test suite, confirm green output, THEN close.
  This applies universally — not just frontend beads
- Do NOT use TodoWrite, TaskCreate, markdown TODO lists, or any other
  tracking mechanism. Beads are the single source of truth
- **ALWAYS run `bd` from the repo root** — never from `server/` or `web/`.
  Running from a subdirectory creates a stray `.beads/` database
- Use `bv` for triage decisions (what to work on next). **CRITICAL: use ONLY
  `--robot-*` flags.** Bare `bv` launches an interactive TUI that blocks
  agent sessions. Always export first:
  ```bash
  bd export -o .beads/beads.jsonl && bv --robot-triage --db .beads/beads.jsonl
  ```

**Rationale**: Beads were closed without running tests, and beads were created
without acceptance criteria. Both failures stem from treating beads as a
logging system rather than a planning system. When the plan (acceptance
criteria) is defined upfront and the gate (tests pass) is enforced at close,
the quality of shipped work improves.

### VII. Commit When Closing a Bead

When a bead is closed, the code changes for that bead MUST be committed in
the same action. The workflow is: tests pass → `bd close` → `git commit`.
This keeps beads and commits in sync — a closed bead always corresponds to
committed code.

- Do NOT push to remote — the user decides when to push
- Do NOT batch multiple beads into one commit unless they are trivially related
- Beads that are in-progress at session end should have notes for the next
  session

**Rationale**: Beads and code drifting out of sync makes it impossible to
understand what state the project is in. Committing at bead close is the
natural checkpoint.

## Bead Checklist

Every bead MUST pass these quality gates before it can be closed. Use this as
a checklist when reviewing bead completion.

**Universal gate (ALL bead types):** All tests referenced in the bead's
acceptance criteria MUST be passing before `bd close`. Run the test suite,
confirm green, then close. No exceptions.

### Migration Beads

- [ ] `make migrate-up` succeeds on clean database
- [ ] `make migrate-down` succeeds (reversible)
- [ ] Go models updated to match new schema
- [ ] `specs/{topic}/architecture.md` updated if schema changed

### Backend Service Beads

- [ ] Service logic implemented
- [ ] Go unit tests added/updated and passing against real PostgreSQL (not mocked)
- [ ] API endpoint wired and returns expected responses
- [ ] `specs/{topic}/spec.md` acceptance scenarios marked complete

### Frontend UI Beads

- [ ] Component implemented and renders correctly
- [ ] Playwright test in `e2e/` added/updated to exercise the user flow
- [ ] Playwright tests PASS (`npx playwright test <relevant-files>`)
- [ ] `specs/{topic}/spec.md` acceptance scenarios marked complete
- [ ] `seed_demo.go` updated to showcase the new feature

### CLI Beads

- [ ] Command implemented with cobra
- [ ] Unit tests added/updated and passing
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
2. Create epic for multi-step work; create child beads with epic prefix
   (e.g., `nori-abc.1` under epic `nori-abc`)
3. Write acceptance criteria on every bead at creation time (see VI)
4. Claim the bead (`bd update <id> --claim`)
5. Implement the feature
6. Write/update tests (Go unit tests for backend, Playwright for frontend)
7. Update `seed_demo.go` if user-facing
8. Update spec files (mark scenarios complete, note deviations)
9. Run tests, confirm all pass
10. Close the bead

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

- Entry point: `CLAUDE.md` (pointers to specs)
- Spec index: `specs/readme.md`
- Build/test/style: `specs/dev-guide.md`
- Bead commands: `bd prime`

**Version**: 2.0.0 | **Ratified**: 2026-04-23 | **Last Amended**: 2026-04-24
