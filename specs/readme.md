# Nori Specs

"A thin layer that holds everything together."

This directory is the **source of truth** for all Nori features. Every feature
has a specification file that documents the who, what, where, why, and how.

---

## How to Use This File

### For AI Agents

When starting work in the Nori codebase, **read this file first**. Use the
Keywords column in the spec tables below to find the relevant spec for your
task. Each spec contains the full context needed to understand and implement a
feature.

### For Humans

This file tracks what's planned and what's shipped. Specs are listed in
priority order — work top-down.

### Spec File Naming

Specs are transitioning to a directory-based structure:

- **New format**: `specs/{feature}/spec.md` — user-focused specification
  (speckit format: user stories, functional requirements, success criteria).
  `specs/{feature}/architecture.md` — technical architecture and
  implementation reference.
- **Legacy format**: `specs/{name}.md` — combined who/what/where/why/how.
  Older specs use this format and will be migrated over time.
- **Implementation checklist**: `specs/{name}-implementation.md` or
  `specs/{feature}/checklists/` — small, committable units of work.

### Workflow

1. **Before implementing a feature**, read its spec file thoroughly.
2. **Create `{name}-implementation.md`** with a checklist of small, shippable,
   committable units of work. Each item should be something you can complete
   and commit independently.
3. **Work through the checklist** item by item. Commit after each item.
4. **When all checklist items are complete**, add a checkmark to the spec in
   the table below.

### Updating This File

When you add a new spec:
1. Add a row to the table in the appropriate priority position.
2. Create the spec file following the template in any existing spec.
3. Include relevant keywords so agents can find it via search.

---

## Specs

Specs listed in priority order. Work top-down.

| # | Spec | Description | Keywords | Done |
|---|------|-------------|----------|------|
| -- | [constitution.md](constitution.md) | Quality gates and development workflow rules for all beads | constitution, quality, testing, playwright, dbtest, migrations, demo, acceptance criteria | :white_check_mark: |
| -- | [dev-guide.md](dev-guide.md) | Build commands, test commands, code style conventions | build, test, lint, format, style, imports, conventions, commands, go, playwright | :white_check_mark: |
| 0 | [architecture.md](architecture.md) | System architecture: components, communication, deployment | architecture, system, docker, CLI, server, web, database, deployment, components, diagram | :white_check_mark: |
| 1 | [data-model.md](data-model.md) | Task, Recipe, and supporting entity data model | schema, database, entities, relations, postgresql, gorm, models, task, recipe | :white_check_mark: |
| 2 | [auth-and-tenancy.md](auth-and-tenancy.md) | Multi-tenant spaces, user roles, authentication | auth, login, spaces, roles, permissions, tenancy, accounts, users | :white_check_mark: |
| 3 | [stations.md](stations.md) | Configurable shop stations with WIP limits | stations, WIP, capacity, buffer, shop floor, layout, workstations | |
| 4 | [recipes/spec.md](recipes/spec.md) | Recipe system: authoring, rolling, versioning, cost tracking | recipe, roll, create, edit, steps, versioning, batch, cost, quoting, save-as-recipe | |
| 4a | [recipes/architecture.md](recipes/architecture.md) | Recipe technical architecture: task-tree model, roll engine, cost pipeline | recipe, architecture, task-tree, roll, pour, batch, fan-in, fan-out, clone, cost, schema | |
| 5 | [orders.md](orders.md) | Customer orders, line items, recipe rolling on confirm | orders, customers, quotes, due date, lead time, sales, pipeline, roll | |
| 6 | [job-flow.md](job-flow.md) | Dependency-graph pull system, ready queue, station view | jobs, tasks, pull, drum, buffer, rope, TOC, flow, bottleneck, WIP, dependencies | |
| 7 | [task-execution.md](task-execution.md) | Running live tasks, ready-work algorithm, gates, capture mode | execution, live, run, capture, deviations, notes, first-time, ready, claim, gates | |
| 8 | [inventory/spec.md](inventory/spec.md) | Materials, BOM, inventory tracking, cost computation (placeholder) | materials, BOM, inventory, lumber, hardware, replenish, stock, cost, finish, custom fields | |
| 9 | [time-tracking.md](time-tracking.md) | Time event store, multiple input sources | time, clock, check-in, checkout, duration, events, sources, logging | |
| 10 | [bottleneck-analytics.md](bottleneck-analytics.md) | Constraint identification, WIP reports, throughput | bottleneck, analytics, constraint, throughput, reports, TOC, metrics | |
| 11 | [cli.md](cli.md) | The `nori` CLI, AI skill for external agents, `nori init` setup | CLI, terminal, commands, task, recipe, ready, roll, cobra, skill, init, agent | |
| 12 | [mcp-server.md](mcp-server.md) | MCP protocol for embedded AI (chat, voice, photo) | MCP, LLM, AI, tools, resources, embedded, chat, voice, internal | |
| 13 | [ai-features.md](ai-features.md) | Embedded AI: recipe suggestions, capture prompts, BYOK | AI, ollama, local, suggestions, prompts, summaries, inference, BYOK, openai, anthropic | |
| 14 | [passive-observation.md](passive-observation.md) | Camera/sensor integration, presence detection | camera, sensors, passive, presence, frigate, vision, detection | |
| 15 | [recipe-marketplace.md](recipe-marketplace.md) | Public Recipe sharing, forking, community | marketplace, public, fork, share, community, open source, recipe |

---

## Roadmap

Future features identified through competitive analysis but not yet promoted
to full specs. These are organized by priority tier and will be turned into
proper specs when the need becomes clear through real usage.

| Document | Description |
|---|---|
| [roadmap-competitive-gaps.md](roadmap-competitive-gaps.md) | Tiered feature roadmap from Odoo/ERPNext cross-analysis: invoicing, purchasing, maintenance, knowledge base, quoting, customer portal, quality inspections, reporting, document management, and integration strategy |
