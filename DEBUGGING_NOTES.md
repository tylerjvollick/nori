# SOP Step Reordering - Flash Bug Fix

## Problem Statement
When reordering SOP steps via drag-and-drop, items would flash back to their original position before settling into the new position.

## Root Cause
Multiple layers of state management were fighting each other:
1. Child component had `localSteps` with guards against updates during reordering
2. Parent component had its own `localSteps` that synced from the store WITHOUT guards
3. Store reloads during the reorder flow triggered updates that overrode optimistic UI changes

The flash occurred because:
- Child updated optimistically ✅
- Store operations (`ensureDraft`, `loadSOP`) triggered during reorder flow
- Parent's `$effect` reset its `localSteps` from potentially stale store data
- This flowed down to child as `steps` prop, causing the flash

## Changes Made (Session 2)

### 1. Store Logging (`web/src/lib/stores/sop.ts`)
Added console logging to track when `loadSOP()` and `ensureDraft()` are called and when they update the store:

```typescript
async loadSOP(id: number) {
  console.log('[sopStore.loadSOP] CALLED for SOP ID:', id);
  // ... logs when store is updated
}

async ensureDraft(sopId: number) {
  console.log('[sopStore.ensureDraft] CALLED for SOP ID:', sopId);
  // ... logs each step of draft creation
}
```

### 2. Child Component Changes (`web/src/lib/components/sop/SOPStepList.svelte`)

#### a) Fixed Equality Check in $effect
**Before:** Compared both `id` and `order` fields
```typescript
const areEqual = steps.length === localSteps.length && 
  steps.every((step, i) => step.id === localSteps[i]?.id && step.order === localSteps[i]?.order);
```

**After:** Only compare `id` sequence (order field changes after reorder)
```typescript
const areIDsIdentical = steps.length === localSteps.length && 
  steps.every((step, i) => step.id === localSteps[i]?.id);
```

**Why:** The `order` field (lexicographic ordering like "a", "b", "c") changes after reordering, but as long as the step IDs are in the correct UI sequence, we don't want to trigger an update.

#### b) Removed ensureDraft() Call from handleReorder
**Before:**
```typescript
if (!isDraftMode) {
  await sopStore.ensureDraft(sopId);
}
```

**After:** Removed entirely

**Why:** The backend `reorderStep` endpoint automatically creates a draft if one doesn't exist, so we don't need to call `ensureDraft` beforehand. This eliminates an unnecessary store reload.

#### c) Added Conditional loadSOP() After Reorder
**After:**
```typescript
// If we weren't in draft mode before, we are now (backend auto-created draft)
// Reload SOP to update the parent's draft status without affecting step order
if (!isDraftMode) {
  console.log('[handleReorder] Was not in draft mode - reloading SOP to update draft status');
  await sopStore.loadSOP(sopId);
}
```

**Why:** When not in draft mode, the backend creates a draft during reorder. We need to reload the SOP to update the UI's draft status (show "Publish" button, etc.). However, this reload should NOT cause a flash because:
1. The step IDs are in the correct sequence in `localSteps`
2. The child's `$effect` now only checks ID sequence (not order values)
3. The check will see IDs match and SKIP the update

## Expected Behavior After Fix

### When Already in Draft Mode:
1. User drags step → optimistic update
2. API call (`reorderStep`) completes
3. Updated step with new `order` value merged into `localSteps`
4. **No store reload** (we skip `loadSOP` since already in draft mode)
5. No flash ✅

### When NOT in Draft Mode:
1. User drags step → optimistic update
2. API call (`reorderStep`) completes (backend auto-creates draft)
3. Updated step with new `order` value merged into `localSteps`
4. **Store reload** via `loadSOP()` to update draft status in UI
5. Parent's `stepsFromStore` updates with steps from store
6. Child's `$effect` fires but checks: "Are IDs in same sequence?" → YES → SKIP
7. No flash ✅

## Testing Instructions

### Manual Test 1: Reorder in Draft Mode
1. Navigate to an SOP page (should auto-create draft or already be in draft)
2. Verify draft badge/status is shown
3. Open browser console (F12)
4. Drag a step to a new position
5. Observe console logs:
   - `[handleDndConsider]` - during drag
   - `[handleReorder] START` - when dropped
   - `[handleReorder] Already in draft mode, skipping ensureDraft`
   - `[handleReorder] Calling reorderStep API`
   - `[handleReorder] SUCCESS`
   - Child `$effect` should log: `SKIPPED - step IDs are in identical order`
6. Verify NO flash (step should smoothly move to new position)
7. Verify step stays in new position

### Manual Test 2: Reorder from Published State
1. Create a new SOP or publish an existing draft
2. Navigate away and back to the SOP (to start fresh)
3. Verify NO draft badge (should be in published state)
4. Open browser console (F12)
5. Drag a step to a new position
6. Observe console logs:
   - `[handleDndConsider]` - during drag
   - `[handleReorder] START` - when dropped
   - `[handleReorder] Calling reorderStep API` (no ensureDraft call)
   - `[handleReorder] Was not in draft mode - reloading SOP to update draft status`
   - `[sopStore.loadSOP] CALLED`
   - `[sopStore.loadSOP] Updating store with new SOP data`
   - Parent `$effect` logs: `stepsFromStore changed`
   - Child `$effect` should log: `SKIPPED - step IDs are in identical order`
   - `[handleReorder] SUCCESS`
7. Verify NO flash (step should smoothly move to new position)
8. Verify draft badge appears (now in draft mode)
9. Verify "Publish" button is enabled

### Manual Test 3: Rapid Multiple Reorders
1. In an SOP with 5+ steps in draft mode
2. Quickly drag steps to different positions (3-4 moves in quick succession)
3. Verify no flashing or jumping around
4. Verify all steps end up in the correct final positions
5. Check console for any errors or unexpected behavior

### Manual Test 4: Error Handling
1. In an SOP, start developer tools
2. Go to Network tab and set it to "Offline" mode
3. Try to reorder a step
4. Verify error is logged
5. Verify step returns to original position (via `loadSOP` in error handler)

## Console Log Sequence (Expected)

### When Reordering in Draft Mode:
```
[SOPStepList $effect] Triggered { isReordering: false, isDragging: false, ... }
[handleDndConsider] Started dragging - isDragging = true
[handleDndConsider] Updated localSteps during drag [...]
[handleReorder] START - setting isReordering = true
[handleReorder] Setting localSteps optimistically [...]
[Parent handleStepsChange] Received steps update from child [...]
[handleReorder] Calling reorderStep API: { stepId: X, beforeStepId: Y, afterStepId: Z }
[handleReorder] API call complete, updated step: { id: X, order: "new_order" }
[Parent handleStepsChange] Received steps update from child [...]
[handleReorder] SUCCESS - setting isReordering = false
[SOPStepList $effect] Triggered { isReordering: false, isDragging: false, ... }
[SOPStepList $effect] SKIPPED - step IDs are in identical order
```

## Additional Notes

### Backend Auto-Draft Creation
The backend's `ReorderStepForTemplate` service method automatically creates a draft if one doesn't exist:
```go
// 2. Get or create draft for this template
draft, err := s.versionRepo.GetDraftByTemplateID(templateID)
if err != nil || draft == nil {
    // No draft exists, create one based on current version
    log.Printf("ReorderStep - no draft found for template %d, creating one", templateID)
    // ... creates draft
}
```

This is why we removed the frontend `ensureDraft()` call - it was redundant and causing unnecessary store reloads.

### Lexicographic Ordering
Steps use lexicographic ordering (strings like "a", "b", "c", "aa", "ab", etc.) for their `order` field. When a step is reordered, the backend calculates a new order value between the `beforeStepId` and `afterStepId`. This is why the `order` field changes, but the UI should still display steps by their position in the array, not by sorting by `order` value.

## Session 3: The Real Fix - Backend Ordering

### Problem Identified
After Session 2 changes, the issue **got worse** - steps would revert to their original position after dropping, but persist correctly after a page refresh. This revealed the true root cause:

**The backend was NOT returning steps sorted by the `order` field!**

When `loadSOP()` was called after reordering:
1. Optimistic `localSteps`: `[step3, step1, step2]` (IDs in UI order)
2. Server returned steps: `[step1, step2, step3]` (different ID sequence!)
3. Child's `areIDsIdentical` check failed → `$effect` updated → visual revert

### The Fix
Updated all GORM `Preload` calls to explicitly order steps by the `order` field:

**Files Changed:**
- `server/internal/repositories/sop_template.repository.go`
- `server/internal/repositories/sop_template_version.repository.go`

**Before:**
```go
Preload("CurrentVersion.Steps")
Preload("Steps")
```

**After:**
```go
Preload("CurrentVersion.Steps", func(db *gorm.DB) *gorm.DB {
    return db.Order("sop_step.order ASC")
})
Preload("Steps", func(db *gorm.DB) *gorm.DB {
    return db.Order("sop_step.order ASC")
})
```

### Why This Works
1. Backend now **always** returns steps sorted by their lexicographic `order` field
2. When a step is reordered, the backend calculates a new `order` value and the array position changes accordingly
3. Frontend's optimistic `localSteps` will have the same ID sequence as the server response
4. Child's `areIDsIdentical` check passes → no update → no flash ✅

### Testing After This Fix

The previous testing instructions still apply, but now the expected behavior is:

1. Drag step to new position
2. Step moves smoothly to new position
3. **No visual revert** 
4. **No flash**
5. Refresh page → step remains in correct position

If you still see issues, verify:
1. Server was rebuilt after changes: `cd server && go build -o nori .`
2. Server was restarted
3. Browser cache was cleared (Cmd+Shift+R)
4. Check console logs to verify step order from API responses

## Next Steps

1. Rebuild and restart the server
2. Test the application manually using the testing instructions above
3. Review console logs to verify steps are returned in correct order from API
4. If successful, consider removing some of the verbose logging added in Session 2

## Previous Session Changes (Session 1)

From the previous session, these changes were made to the parent component:

### Parent Component (`web/src/routes/sops/[id]/+page.svelte`)
- Removed parent's `localSteps` state management for steps
- Added `currentSteps` state (receives updates from child via callback)
- Added `stepsFromStore` derived value (passes steps from store to child)
- Removed steps from parent's `$effect` sync
- Added `handleStepsChange` callback
- Updated all save operations to use `currentSteps`

These changes remain in place and work together with Session 2 changes.
