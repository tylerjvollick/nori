<script lang="ts">
  import type { SOPStep, SOPStepMedia } from '$lib/api/sop';
  import { sopApi } from '$lib/api/sop';
  import { Button } from '$lib/components/ui/button';
  import { Badge } from '$lib/components/ui/badge';
  import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '$lib/components/ui/collapsible';
  import { Clock } from '@lucide/svelte';
  import SOPStepMediaGrid from './SOPStepMediaGrid.svelte'; 

  interface Props {
    step: SOPStep;
    stepIndex: number;
    displayIndex: number;
    isDraftMode: boolean;
    expanded: boolean;
    sopId: number;
    onFieldUpdate: (updates: Partial<SOPStep>) => Promise<void>;
  }

  let {
    step = $bindable(),
    stepIndex,
    displayIndex,
    isDraftMode,
    expanded = $bindable(),
    sopId,
    onFieldUpdate
  }: Props = $props();

  // State for inline editing
  let editingTitle = $state(false);
  let editingDescription = $state(false);
  let editingTime = $state(false);
  
  let editedTitle = $state('');
  let editedDescription = $state('');
  let editedTime = $state<number | undefined>();
  
  let titleInputRef: HTMLInputElement | null = $state(null);
  let descriptionTextareaRef: HTMLTextAreaElement | null = $state(null);
  let timeInputRef: HTMLInputElement | null = $state(null);

  // Media state
  let stepPhotos = $state<SOPStepMedia[]>([]);
  let loadingPhotos = $state(false);
  let photosLoaded = $state(false);

  // Load media when expanded
  $effect(() => {
    if (expanded && !photosLoaded && !loadingPhotos) {
      loadPhotos();
    }
  });

  async function loadPhotos() {
    try {
      loadingPhotos = true;
      stepPhotos = await sopApi.getStepMedia(sopId, step.id);
      photosLoaded = true;
    } catch (error) {
      console.error('Failed to load media:', error);
      photosLoaded = true; // Mark as loaded even on error to show the upload button
    } finally {
      loadingPhotos = false;
    }
  }

  function handlePhotosChange(photos: SOPStepMedia[]) {
    stepPhotos = photos;
  }

  // Title editing
  function startTitleEdit() {
    editedTitle = step.title;
    editingTitle = true;
    setTimeout(() => {
      titleInputRef?.focus();
      titleInputRef?.select();
    }, 0);
  }

  async function saveTitleEdit() {
    if (editedTitle.trim() && editedTitle.trim() !== step.title) {
      await onFieldUpdate({ title: editedTitle.trim() });
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

  // Description editing
  function startDescriptionEdit() {
    editedDescription = step.instructions || '';
    editingDescription = true;
    setTimeout(() => {
      descriptionTextareaRef?.focus();
    }, 0);
  }

  async function saveDescriptionEdit() {
    if (editedDescription !== step.instructions) {
      await onFieldUpdate({ instructions: editedDescription });
      step.instructions = editedDescription;
    }
    editingDescription = false;
  }

  function cancelDescriptionEdit() {
    editingDescription = false;
    editedDescription = '';
  }

  // Time editing
  function startTimeEdit() {
    editedTime = step.estimatedTimeMinutes;
    editingTime = true;
    setTimeout(() => {
      timeInputRef?.focus();
      timeInputRef?.select();
    }, 0);
  }

  async function saveTimeEdit() {
    if (editedTime !== step.estimatedTimeMinutes) {
      await onFieldUpdate({ estimatedTimeMinutes: editedTime });
      step.estimatedTimeMinutes = editedTime;
    }
    editingTime = false;
  }

  function cancelTimeEdit() {
    editingTime = false;
    editedTime = undefined;
  }

  function handleTimeKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      saveTimeEdit();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      cancelTimeEdit();
    }
  }

  // Approval toggle
  async function toggleApproval() {
    const newValue = !step.requiresApproval;
    await onFieldUpdate({ requiresApproval: newValue });
    step.requiresApproval = newValue;
  }
</script>

<Collapsible bind:open={expanded}>
  <div class="step-item border rounded-lg overflow-hidden group border-border">
    <!-- Collapsed View -->
    <div class="flex items-center justify-between p-4 hover:bg-secondary/50 dark:hover:bg-secondary/900">
      <div class="flex items-center gap-3 flex-1">
        <!-- Drag handle (always visible) -->
        <div 
          data-drag-handle
          class="cursor-grab active:cursor-grabbing text-muted-foreground hover:text-foreground p-1 rounded transition-colors"
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
          </div>
        {/if}
      </div>
      
      <div class="flex items-center gap-3">
            {#if step.instructions}
              <svg 
                class="w-4 h-4 text-muted-foreground dark:text-muted-foreground500 flex-shrink-0" 
                fill="none" 
                stroke="currentColor" 
                viewBox="0 0 24 24"
                aria-label="Has description"
              >
                <title>Has description</title>
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            {/if}
        {#if step.estimatedTimeMinutes}
          <Badge variant="outline" class="flex items-center gap-1">
            <Clock class="w-3 h-3" />
            {step.estimatedTimeMinutes}m
          </Badge>
        {/if}
        <CollapsibleTrigger 
          class="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground h-9 w-9"
          aria-label="Toggle step details"
        >
          <svg
            class="w-5 h-5 transform transition-transform {expanded ? 'rotate-180' : ''}"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </CollapsibleTrigger>
      </div>
    </div>

    <!-- Expanded View -->
    <CollapsibleContent>
      <div class="border-t border-border p-4 bg-background space-y-4">
        <!-- Media Section (moved to top) -->
        <div>
          <h4 class="text-sm font-medium text-foreground mb-2">Media</h4>
          {#if loadingPhotos}
            <div class="text-sm text-muted-foreground">Loading media...</div>
          {:else}
            <SOPStepMediaGrid
              {sopId}
              stepId={step.id}
              photos={stepPhotos}
              onPhotosChange={handlePhotosChange}
            />
          {/if}
        </div>

        <!-- Description Field -->
        {#if editingDescription}
          <div class="space-y-2">
            <label for="step-description-{stepIndex}" class="block text-sm font-medium text-foreground">Description</label>
            <textarea
              id="step-description-{stepIndex}"
              bind:this={descriptionTextareaRef}
              bind:value={editedDescription}
              rows="4"
              class="w-full bg-card border border-border rounded-lg px-3 py-2 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              placeholder="Add a description for this step..."
            ></textarea>
            <div class="flex gap-2">
              <Button
                onclick={saveDescriptionEdit}
                type="button"
                size="sm"
              >
                Save
              </Button>
              <Button
                onclick={cancelDescriptionEdit}
                type="button"
                variant="secondary"
                size="sm"
              >
                Cancel
              </Button>
            </div>
          </div>
        {:else}
          <button
            onclick={startDescriptionEdit}
            type="button"
            class="w-full text-left p-3 rounded-lg hover:bg-accent/50 cursor-pointer transition-colors group/description"
          >
            <h4 class="text-sm font-medium text-foreground mb-1 flex items-center gap-2">
              Description
              <svg class="w-3.5 h-3.5 text-muted-foreground opacity-0 group-hover/description:opacity-100 transition-opacity" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
            </h4>
            {#if step.instructions}
              <p class="text-foreground whitespace-pre-wrap text-sm">
                {step.instructions}
              </p>
            {:else}
              <p class="text-muted-foreground text-sm italic">
                Click to add a description...
              </p>
            {/if}
          </button>
        {/if}

        <!-- Estimated Time Field -->
        <div>
          <h4 class="text-sm font-medium text-foreground mb-2">Estimated Time</h4>
          {#if editingTime}
            <div class="flex items-center gap-2">
              <input
                id="step-time-{stepIndex}"
                bind:this={timeInputRef}
                bind:value={editedTime}
                onkeydown={handleTimeKeydown}
                type="number"
                min="0"
                class="w-32 bg-card border border-border rounded px-2 py-1 text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                placeholder="Minutes"
              />
              <span class="text-sm text-muted-foreground">minutes</span>
              <Button
                variant="ghost"
                size="icon-sm"
                onclick={saveTimeEdit}
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
                onclick={cancelTimeEdit}
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
            <button
              onclick={startTimeEdit}
              type="button"
              class="inline-flex items-center gap-2 px-3 py-1.5 rounded hover:bg-accent/50 cursor-pointer transition-colors text-sm group/time"
            >
              {#if step.estimatedTimeMinutes}
                <span class="text-foreground font-medium">{step.estimatedTimeMinutes} minutes</span>
              {:else}
                <span class="text-muted-foreground italic">Click to add estimated time...</span>
              {/if}
              <svg class="w-3.5 h-3.5 text-muted-foreground opacity-0 group-hover/time:opacity-100 transition-opacity" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
            </button>
          {/if}
        </div>

        <!-- Requires Approval Toggle -->
        <div>
          <h4 class="text-sm font-medium text-foreground mb-2">Options</h4>
          <label class="flex items-center gap-2 cursor-pointer hover:bg-accent/50 p-2 rounded transition-colors w-fit">
            <input
              type="checkbox"
              checked={step.requiresApproval}
              onchange={toggleApproval}
              class="rounded"
            />
            <span class="text-sm text-foreground">
              Requires Approval
            </span>
          </label>
        </div>
      </div>
    </CollapsibleContent>
  </div>
</Collapsible>
