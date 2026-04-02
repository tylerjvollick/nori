# SOP Versioning

## Who

- **Shop owners / managers**: Iterate on SOPs without disrupting active work.
- **Operators**: Always see the latest published version during execution.

## What

A version control system for SOPs. Managers can edit SOPs as drafts, test
changes, and publish when ready — without affecting jobs currently in progress.
Full version history is maintained so changes can be reviewed and rolled back.

## Where

- Backend: `server/internal/models/sop_template_version.model.go`, version API
  endpoints
- Frontend: SOP editor (draft/publish controls), version history viewer
- Data model: see data-model.md

## Why

From the original backlog:

> As a floor manager, I need to be able to work on SOPs (continuous improvement)
> without disrupting the flow of current work being done on the floor.

This is a real problem in manufacturing. You discover a better way to do step 4
while building an order, but there are two more units of that order in progress
using the old process. You can't change the SOP out from under them.

The sushi restaurant metaphor from the requirements doc captures this perfectly:
the chef updates the SOP for the special mid-service, and Nori asks if they want
to merge the change back to the template. In-flight orders of the same type can
optionally pick up the new version.

## How

### Version Lifecycle

```
[Draft] → [Published] → [Archived]
                ↓
           [New Draft] → [Published] → ...
```

- **Draft**: Work in progress. Not visible to operators on the floor. Only the
  author (and managers) can see and edit drafts.
- **Published**: The active version. When a Job is created, it snapshots the
  current published version. Operators executing jobs see this version.
- **Archived**: Previous published versions. Read-only. Available for
  comparison and rollback.

### Version Numbers

Simple incrementing integers: v1, v2, v3, etc. Each publish bumps the version
number by 1, regardless of how many intermediate saves happened during the
draft phase.

### Draft Behavior

- Only **one active draft** per SOPTemplate at a time.
- Multiple edits to a draft (adding steps, changing instructions, uploading
  photos) are all saved to the same draft version — not one version per edit.
- Drafts are accessible from the SOP list page (filtered view) so managers
  can return to them after switching tasks.
- A draft can be discarded without publishing.

### Publishing

When a draft is published:
1. The draft's status changes from `draft` to `published`.
2. The SOPTemplate's `CurrentVersionID` is updated to point to this version.
3. The previously published version's status changes to `archived` (or
   remains `published` with an `IsActive=false` flag).
4. New Jobs created after this point will use the new version.
5. **In-flight Jobs are not affected** — they continue using the version that
   was current when the Job was created.

### Propagation to In-Flight Jobs

After publishing a new version, the system can optionally prompt:
> "There are 3 in-flight jobs using v2. Would you like to update them to v3?"

This is opt-in per job. Some jobs may be too far along to switch versions
safely. The manager decides per-job.

### Change Summary

Each version has an optional `ChangeSummary` field (already in the existing
model). When publishing, the system prompts for a summary. If AI features
are enabled (see ai-features.md), Nori can auto-generate a summary by
diffing the steps between versions.

### Version Diff View

A side-by-side or inline diff showing what changed between two versions:
- Steps added, removed, or reordered
- Instruction text changes
- Media added or removed
- Material changes

This is a nice-to-have for v1 but important for continuous improvement — you
need to see *what* changed and *why* over time.

### Prior Art

The existing model already supports this well:
- `SOPTemplateVersion` has `Status` (draft/published), `VersionNumber`,
  `ChangeSummary`, `IsActive`
- `SOPTemplate` has `CurrentVersionID` pointing to the active version
- Steps are tied to versions, so each version has its own set of steps

The foundation is solid. The main additions are:
- UI for draft management (list drafts, return to editing)
- Publish flow with in-flight job notification
- Version diff view (later phase)

### API Surface

```
POST   /api/sops/:id/versions                — Create new draft from current published
PUT    /api/sop-versions/:versionId/publish   — Publish a draft
DELETE /api/sop-versions/:versionId           — Discard a draft
GET    /api/sops/:id/versions                 — List all versions with status
GET    /api/sops/:id/versions/:v1/diff/:v2    — Diff two versions (future)
```

## Open Questions

- Should publishing require approval from a second person (owner/manager), or
  can any manager self-publish? (Leaning toward self-publish for v1 — small
  shops don't have approval chains.)
- When propagating a new version to in-flight jobs, should it update all
  remaining steps or only steps the job hasn't started yet?
- Should archived versions be deletable, or always retained for audit trail?
