<script lang="ts">
  import { dndzone } from 'svelte-dnd-action';
  import { sopApi } from '$lib/api/sop';
  import type { SOPStep } from '$lib/api/sop';
  import SOPStepItem from './SOPStepItem.svelte';
  import { Button } from '$lib/components/ui/button';

  interface Props {
    steps: SOPStep[];
    sopId: number;
    isDraftMode: boolean;
    onStepsChange?: (steps: SOPStep[]) => void;
    onEnsureDraft?: () => Promise<void>;
  }

  let { steps = [], sopId, isDraftMode = false, onStepsChange, onEnsureDraft }: Props = $props();

  // Local state
  let localSteps = $state<SOPStep[]>([...steps]);
  let expandedSteps = $state(new Set<number>());
  let editingSteps = $state(new Set<number>());
  let creatingNewStep = $state(false);
  let newStepTitle = $state('');
  let newStepTitleInput = $state<HTMLInputElement>();
  let isDragging = $state(false);

  // Update local state when props change
  $effect(() => {
    localSteps = [...steps];
  });

  // Notify parent of changes
  function notifyChange() {
    if (onStepsChange) {
      onStepsChange(localSteps);
    }
  }

  function toggleStep(stepId: number) {
    if (expandedSteps.has(stepId)) {
      expandedSteps = new Set([...expandedSteps].filter(id => id !== stepId));
    } else {
      expandedSteps = new Set([...expandedSteps, stepId]);
    }
  }

  function startAddingStep() {
    creatingNewStep = true;
    // Focus the input after state updates
    setTimeout(() => {
      newStepTitleInput?.focus();
    }, 0);
  }

  function cancelAddingStep() {
    creatingNewStep = false;
    newStepTitle = '';
  }

  async function saveNewStep() {
    if (!newStepTitle.trim()) return;

    try {
      // Ensure we have a draft before creating a step
      if (onEnsureDraft) {
        await onEnsureDraft();
      }
      
      const newStep = await sopApi.createStep(sopId, {
        title: newStepTitle.trim(),
        afterStepId: localSteps.length > 0 ? localSteps[localSteps.length - 1].id : undefined
      });

      // Optimistically add the step to local state
      localSteps = [...localSteps, newStep];
      notifyChange();

      // Reset form
      newStepTitle = '';
      creatingNewStep = false;
    } catch (error) {
      console.error('Failed to create step:', error);
    }
  }

  function handleNewStepKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      saveNewStep();
    } else if (e.key === 'Tab') {
      e.preventDefault();
      saveNewStep();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      cancelAddingStep();
    }
  }

  async function saveStep(stepIndex: number) {
    if (!localSteps[stepIndex]) return;

    const step = localSteps[stepIndex];
    
    try {
      // Ensure we have a draft before updating a step
      if (onEnsureDraft) {
        await onEnsureDraft();
      }
      
      await sopApi.updateStep(sopId, step.id, {
        title: step.title,
        instructions: step.instructions,
        estimatedTimeMinutes: step.estimatedTimeMinutes,
        requiresApproval: step.requiresApproval
      });

      // Optimistic update - already in localSteps
      notifyChange();

      // Exit edit mode
      editingSteps = new Set([...editingSteps].filter(id => id !== stepIndex));
    } catch (error) {
      console.error('Failed to update step:', error);
    }
  }

  function startEditingStep(stepIndex: number) {
    editingSteps = new Set([...editingSteps, stepIndex]);
  }

  function cancelEditingStep(stepIndex: number) {
    editingSteps = new Set([...editingSteps].filter(id => id !== stepIndex));
    // Reset to original values
    localSteps = [...steps];
  }

  // DnD event handlers
  function handleDndConsider(e: CustomEvent) {
    // Update localSteps based on the new order during dragging
    localSteps = e.detail.items;
    console.log('handleDndConsider', localSteps.map(e => ({ id: e.id, order: e.order })))

    // Track if we're actively dragging
    if (!isDragging) {
      isDragging = true;
      console.log('Started dragging');
    }
  }

  async function handleDndFinalize(e: CustomEvent) {
    // Update localSteps based on the final order
    const newLocalSteps = e.detail.items;
    console.log('handleDndFinalize', { newLocalSteps }) 
    // Find which item was moved
    const info = e.detail.info;
    console.log('DnD finalize info:', info);
    
    // Check if this was a real drag (not just a click)
    if (!info) {
      console.log('No info, skipping reorder');
      localSteps = newLocalSteps;
      return;
    }
    
    const movedItemId = info.id;
    const newIndex = newLocalSteps.findIndex((item: any) => item.id === movedItemId);
    
    console.log('Moved item:', movedItemId, 'to index:', newIndex);
    
    if (newIndex === -1) {
      console.log('Invalid index, skipping reorder');
      localSteps = newLocalSteps;
      return;
    }
    
    // Reset dragging state first
    isDragging = false;
    console.log('Stopped dragging');
    
    // Call backend to reorder (will auto-create draft if needed)
    await handleReorder(movedItemId, newIndex, newLocalSteps);
  }

  async function handleReorder(stepId: number, newIndex: number, newLocalSteps: SOPStep[]) {
    try {
      isReordering = true;
      
      // Ensure we have a draft before reordering
      if (onEnsureDraft) {
        await onEnsureDraft();
      }
      
      // Determine beforeStepId and afterStepId based on newIndex
      // beforeStepId: the step with LOWER lexicographic order (comes first in sort)
      // afterStepId: the step with HIGHER lexicographic order (comes second in sort)
      let beforeStepId: number | undefined;
      let afterStepId: number | undefined;
      
      if (newIndex > 0) {
        // The step at index-1 has a lower order value
        beforeStepId = newLocalSteps[newIndex - 1].id;
      }
      
      if (newIndex < newLocalSteps.length - 1) {
        // The step at index+1 has a higher order value
        afterStepId = newLocalSteps[newIndex + 1].id;
      }
      
      console.log('Reorder API call:', { stepId, beforeStepId, afterStepId, newIndex });
      
      // Call backend API
      await sopApi.reorderStep(sopId, stepId, {
        beforeStepId,
        afterStepId
      });
      
      // Update local state optimistically
      localSteps = newLocalSteps;
      notifyChange();
      
      isReordering = false;
    } catch (error) {
      console.error('Failed to reorder step:', error);
      isReordering = false;
      // Reset to original order on error
      localSteps = [...steps];
      notifyChange();
    }
  }
</script>

<div>
  {#if localSteps && localSteps.length > 0}
    <div>
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-semibold text-foreground">Steps</h2>
        <Button
          onclick={startAddingStep}
          size="sm"
        >
          + Add Step
        </Button>
      </div>
    
      <div 
        class="space-y-2 {isDragging ? 'dragging-active' : ''}"
        use:dndzone={{ items: localSteps, flipDurationMs: 200, dropTargetStyle: { outline: 'none' }, dragDisabled: false }}
        onconsider={handleDndConsider}
        onfinalize={handleDndFinalize}
      >
        {#each localSteps as step, index (step.id)}
          <SOPStepItem
            bind:step={localSteps[index]}
            stepIndex={index}
            displayIndex={index + 1}
            {isDraftMode}
            expanded={expandedSteps.has(step.id)}
            editing={editingSteps.has(index)}
            ontoggle={() => toggleStep(step.id)}
            onsave={() => saveStep(index)}
            oncancel={() => cancelEditingStep(index)}
            onedit={() => startEditingStep(index)}
          />
        {/each}

        <!-- New Step Form -->
        {#if creatingNewStep}
          <div class="border border-primary/30 rounded-lg overflow-hidden bg-primary/5">
            <div class="flex items-center gap-3 p-4">
              <span class="inline-flex items-center justify-center w-8 h-8 bg-primary text-primary-foreground rounded-full font-bold text-sm flex-shrink-0">
                {localSteps.length + 1}
              </span>
              <input
                type="text"
                bind:this={newStepTitleInput}
                bind:value={newStepTitle}
                onkeydown={handleNewStepKeydown}
                placeholder="Enter step title and press Enter or Tab..."
                class="flex-1 bg-card border border-border rounded-lg px-3 py-2 text-foreground focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <Button
                onclick={cancelAddingStep}
                variant="ghost"
                size="icon"
                aria-label="Cancel"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </Button>
            </div>
          </div>
        {/if}
      </div>
    </div>
  {:else}
    <!-- No steps yet -->
    <div>
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-semibold text-foreground">Steps</h2>
        <Button
          onclick={startAddingStep}
          size="sm"
        >
          + Add Step
        </Button>
      </div>
      
      {#if creatingNewStep}
        <div class="border border-primary/30 rounded-lg overflow-hidden bg-primary/5">
          <div class="flex items-center gap-3 p-4">
            <span class="inline-flex items-center justify-center w-8 h-8 bg-primary text-primary-foreground rounded-full font-bold text-sm flex-shrink-0">
              1
            </span>
            <input
              type="text"
              bind:this={newStepTitleInput}
              bind:value={newStepTitle}
              onkeydown={handleNewStepKeydown}
              placeholder="Enter step title and press Enter or Tab..."
              class="flex-1 bg-card border border-border rounded-lg px-3 py-2 text-foreground focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
            <Button
              onclick={cancelAddingStep}
              variant="ghost"
              size="icon"
              aria-label="Cancel"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </Button>
          </div>
        </div>
      {:else}
        <p class="text-muted-foreground text-sm">No steps yet. Click "Add Step" to create your first step.</p>
      {/if}
    </div>
  {/if}
</div>

<style>
  /* Drop indicator styling */
  :global([data-is-dnd-shadow-item-hint="true"]) {
    border: 2px solid #3b82f6 !important;
    background: transparent !important;
    height: 4px !important;
    border-radius: 2px;
    margin: 8px 0;
  }

  /* Make dragged item semi-transparent */
  :global(.step-item[aria-grabbed="true"]) {
    opacity: 0.5;
  }

  /* Smooth transitions */
  :global(.dragging-active .step-item) {
    transition: transform 200ms ease;
  }
</style>
