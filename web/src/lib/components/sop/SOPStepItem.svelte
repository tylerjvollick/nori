<script lang="ts">
  import type { SOPStep } from '$lib/api/sop';
  import { Button } from '$lib/components/ui/button';

  interface Props {
    step: SOPStep;
    stepIndex: number;
    displayIndex: number;
    isDraftMode: boolean;
    expanded: boolean;
    editing: boolean;
    ontoggle: () => void;
    onsave: () => void;
    oncancel: () => void;
    onedit: () => void;
  }

  let {
    step = $bindable(),
    stepIndex,
    displayIndex,
    isDraftMode,
    expanded,
    editing,
    ontoggle,
    onsave,
    oncancel,
    onedit
  }: Props = $props();

  // State for inline title editing
  let editingTitle = $state(false);
  let editedTitle = $state('');
  let titleInputRef: HTMLInputElement | null = $state(null);

  function startTitleEdit() {
    editedTitle = step.title;
    editingTitle = true;
    // Focus the input after it's rendered
    setTimeout(() => {
      titleInputRef?.focus();
      titleInputRef?.select();
    }, 0);
  }

  function saveTitleEdit() {
    if (editedTitle.trim()) {
      step.title = editedTitle.trim();
    }
    editingTitle = false;
  }

  function cancelTitleEdit() {
    editingTitle = false;
    editedTitle = '';
  }

  function handleTitleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      saveTitleEdit();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      cancelTitleEdit();
    }
  }
</script>

<div class="step-item border rounded-lg overflow-hidden group border-border">
  <!-- Collapsed View -->
  <div class="flex items-center justify-between p-4 hover:bg-secondary/50 dark:hover:bg-secondary/900">
    <div class="flex items-center gap-3 flex-1">
      <!-- Drag handle (always visible) -->
      <div 
        class="cursor-grab active:cursor-grabbing text-muted-foreground hover:text-muted-foreground600 dark:hover:text-muted-foreground300"
        aria-label="Drag to reorder"
        title="Drag to reorder"
      >
        <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path d="M7 2a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 2zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 8zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 14zm6-8a2 2 0 1 0-.001-4.001A2 2 0 0 0 13 6zm0 2a2 2 0 1 0 .001 4.001A2 2 0 0 0 13 8zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 13 14z" />
        </svg>
      </div>

      <span class="inline-flex items-center justify-center w-8 h-8 bg-primary text-primary-foreground rounded-full font-bold text-sm flex-shrink-0">
        {displayIndex}
      </span>

      {#if editingTitle}
        <!-- Title Edit Mode -->
        <div class="flex items-center gap-2 flex-1">
          <input
            bind:this={titleInputRef}
            bind:value={editedTitle}
            onkeydown={handleTitleKeydown}
            type="text"
            class="flex-1 bg-card border border-border rounded px-2 py-1 text-base font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          />
          <Button
            variant="ghost"
            size="icon-sm"
            onclick={saveTitleEdit}
            aria-label="Accept"
            type="button"
            class="text-green-600 hover:text-green-700 hover:bg-green-100 dark:hover:bg-green-900"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            onclick={cancelTitleEdit}
            aria-label="Cancel"
            type="button"
            class="text-red-600 hover:text-red-700 hover:bg-red-100 dark:hover:bg-red-900"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </Button>
        </div>
      {:else}
        <!-- Title View Mode with hover edit -->
        <div class="flex items-center gap-2 flex-1 group/title">
          <h3 class="text-base font-semibold text-foreground">
            {step.title}
          </h3>
          <button
            onclick={startTitleEdit}
            type="button"
            class="opacity-0 group-hover/title:opacity-100 transition-opacity p-1 hover:bg-accent rounded"
            aria-label="Edit title"
          >
            <svg class="w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
            </svg>
          </button>
          {#if step.instructions}
            <svg 
              class="w-4 h-4 text-muted-foreground dark:text-muted-foreground500 flex-shrink-0" 
              fill="none" 
              stroke="currentColor" 
              viewBox="0 0 24 24"
              aria-label="Has instructions"
            >
              <title>Has instructions</title>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          {/if}
        </div>
      {/if}
    </div>
    
    <div class="flex items-center gap-3">
      {#if step.estimatedTimeMinutes}
        <span class="bg-primary/10 text-primary-foreground dark:bg-primary dark:text-primary-foreground px-2 py-1 rounded text-xs">
          {step.estimatedTimeMinutes} min
        </span>
      {/if}
      <Button
        variant="ghost"
        size="icon"
        onclick={ontoggle}
        aria-label="Toggle step details"
        type="button"
      >
        <svg
          class="w-5 h-5 transform transition-transform {expanded ? 'rotate-180' : ''}"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
      </Button>
    </div>
  </div>

  <!-- Expanded View -->
  {#if expanded}
    <div class="border-t border-border p-4 bg-background">
      {#if editing}
        <!-- Edit Mode -->
        <div class="space-y-4">
          <div>
            <label for="step-title-{stepIndex}" class="block text-sm font-medium text-foreground mb-1">Title</label>
            <input
              id="step-title-{stepIndex}"
              type="text"
              bind:value={step.title}
              class="w-full bg-card border border-border rounded-lg px-3 py-2 text-foreground"
            />
          </div>
          
          <div>
            <label for="step-instructions-{stepIndex}" class="block text-sm font-medium text-foreground mb-1">Instructions</label>
            <textarea
              id="step-instructions-{stepIndex}"
              bind:value={step.instructions}
              rows="4"
              class="w-full bg-card border border-border rounded-lg px-3 py-2 text-foreground"
            ></textarea>
          </div>
          
          <div>
            <label for="step-time-{stepIndex}" class="block text-sm font-medium text-foreground mb-1">Estimated Time (minutes)</label>
            <input
              id="step-time-{stepIndex}"
              type="number"
              bind:value={step.estimatedTimeMinutes}
              class="w-full bg-card border border-border rounded-lg px-3 py-2 text-foreground"
            />
          </div>
          
          <div class="flex items-center gap-2">
            <input
              type="checkbox"
              bind:checked={step.requiresApproval}
              id="approval-{stepIndex}"
              class="rounded"
            />
            <label for="approval-{stepIndex}" class="text-sm text-foreground">
              Requires Approval
            </label>
          </div>
          
          <div class="flex gap-2">
            <Button
              onclick={onsave}
              type="button"
              size="sm"
            >
              Save
            </Button>
            <Button
              onclick={oncancel}
              type="button"
              variant="secondary"
              size="sm"
            >
              Cancel
            </Button>
          </div>
        </div>
      {:else}
        <!-- View Mode -->
        <div class="space-y-3">
          {#if step.instructions}
            <div>
              <h4 class="text-sm font-medium text-foreground mb-1">Instructions</h4>
              <p class="text-foreground whitespace-pre-wrap">
                {step.instructions}
              </p>
            </div>
          {/if}

          <div class="flex gap-2">
            {#if step.requiresApproval}
              <span class="bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200 px-2 py-1 rounded text-xs">
                Approval Required
              </span>
            {/if}
          </div>

          <Button
            onclick={onedit}
            type="button"
            variant="link"
            size="sm"
            class="h-auto p-0"
          >
            Edit Step
          </Button>
        </div>
      {/if}
    </div>
  {/if}
</div>
