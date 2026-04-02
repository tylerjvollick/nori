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

- **Specification**: `specs/{name}.md` — the who/what/where/why/how of a feature.
- **Implementation checklist**: `specs/{name}-implementation.md` — a checklist
  of small, committable units of work. Created before implementation begins.

### Workflow

1. **Before implementing a feature**, read its spec file thoroughly.
2. **Create `{name}-implementation.md`** with a checklist of small, shippable,
   committable units of work. Each item should be something you can complete
   and commit independently.
3. **Work through the checklist** item by item. Commit after each item.
4. **When all checklist items are complete**, move the spec from the Planned
   table to the Implemented table below. Add the completion date.

### Updating This File

When you move a spec to Implemented:
1. Remove the row from the **Planned** table.
2. Add it to the **Implemented** table with the date completed.
3. Keep the Implemented table sorted by completion date (newest first).

When you add a new spec:
1. Add a row to the **Planned** table in the appropriate priority position.
2. Create the spec file following the template in any existing spec.
3. Include relevant keywords so agents can find it via search.

---

## Planned

Specs listed in priority order. Work top-down.

| # | Spec | Description | Keywords |
|---|------|-------------|----------|
| 1 | [data-model.md](data-model.md) | Foundational data model for all entities | schema, database, entities, relations, postgresql, gorm, models |
| 2 | [auth-and-tenancy.md](auth-and-tenancy.md) | Multi-tenant spaces, user roles, authentication | auth, login, spaces, roles, permissions, tenancy, accounts, users |
| 3 | [stations.md](stations.md) | Configurable shop stations with WIP limits | stations, WIP, capacity, buffer, shop floor, layout, workstations |
| 4 | [sop-authoring.md](sop-authoring.md) | Creating and editing SOPs with steps, media, materials | SOP, create, edit, steps, photos, video, instructions, procedures |
| 5 | [sop-versioning.md](sop-versioning.md) | Draft/publish workflow, version history | SOP, versions, draft, publish, diff, history, continuous improvement |
| 6 | [orders.md](orders.md) | Customer orders, line items, due dates, lead time | orders, customers, quotes, due date, lead time, sales, pipeline |
| 7 | [job-flow.md](job-flow.md) | Jobs through stations, pull system, drum-buffer-rope | jobs, kanban, pull, drum, buffer, rope, TOC, flow, bottleneck, WIP |
| 8 | [sop-execution.md](sop-execution.md) | Running a live job against an SOP, step progression | execution, live, run, capture, deviations, notes, first-time |
| 9 | [materials-and-bom.md](materials-and-bom.md) | Bill of materials, stock thresholds, pull signals | materials, BOM, inventory, lumber, hardware, replenish, stock |
| 10 | [time-tracking.md](time-tracking.md) | Time event store, multiple input sources | time, clock, check-in, checkout, duration, events, sources, logging |
| 11 | [bottleneck-analytics.md](bottleneck-analytics.md) | Constraint identification, WIP reports, throughput | bottleneck, analytics, constraint, throughput, reports, TOC, metrics |
| 12 | [cli.md](cli.md) | The `nori` CLI for terminal-native workflows | CLI, terminal, commands, checkin, checkout, status, cobra |
| 13 | [mcp-server.md](mcp-server.md) | MCP protocol for LLM client integration | MCP, LLM, AI, tools, resources, opencode, claude, protocol |
| 14 | [ai-features.md](ai-features.md) | Embedded Ollama: SOP suggestions, capture prompts | AI, ollama, local, suggestions, prompts, summaries, inference |
| 15 | [passive-observation.md](passive-observation.md) | Camera/sensor integration, presence detection | camera, sensors, passive, presence, frigate, vision, detection |
| 16 | [sop-marketplace.md](sop-marketplace.md) | Public SOP sharing, forking, community | marketplace, public, fork, share, community, open source |

## Implemented

Specs listed by completion date (newest first).

| Spec | Description | Completed |
|------|-------------|-----------|
| — | — | — |
