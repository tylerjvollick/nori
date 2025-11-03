<script lang="ts">
  import type { SOPStep } from '$lib/api/sop';

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
</script>

<div class="step-item border rounded-lg overflow-hidden group border-gray-200 dark:border-gray-700">
  <!-- Collapsed View -->
  <div class="flex items-center justify-between p-4 hover:bg-gray-50 dark:hover:bg-gray-900">
    <div class="flex items-center gap-3 flex-1">
      <!-- Drag handle (always visible) -->
      <div 
        class="cursor-grab active:cursor-grabbing text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
        aria-label="Drag to reorder"
        title="Drag to reorder"
      >
        <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path d="M7 2a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 2zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 8zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 7 14zm6-8a2 2 0 1 0-.001-4.001A2 2 0 0 0 13 6zm0 2a2 2 0 1 0 .001 4.001A2 2 0 0 0 13 8zm0 6a2 2 0 1 0 .001 4.001A2 2 0 0 0 13 14z" />
        </svg>
      </div>
      <button 
        class="flex items-center gap-3 flex-1 text-left"
        onclick={ontoggle}
        type="button"
      >
        <span class="inline-flex items-center justify-center w-8 h-8 bg-blue-600 text-white rounded-full font-bold text-sm flex-shrink-0">
          {displayIndex}
        </span>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">
          {step.title}
        </h3>
        {#if step.instructions}
          <svg 
            class="w-4 h-4 text-gray-400 dark:text-gray-500 flex-shrink-0" 
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24"
            aria-label="Has instructions"
          >
            <title>Has instructions</title>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        {/if}
      </button>
    </div>
    
    <div class="flex items-center gap-3">
      {#if step.estimatedTimeMinutes}
        <span class="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200 px-2 py-1 rounded text-xs">
          {step.estimatedTimeMinutes} min
        </span>
      {/if}
      <button
        class="text-gray-600 dark:text-gray-400"
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
      </button>
    </div>
  </div>

  <!-- Expanded View -->
  {#if expanded}
    <div class="border-t border-gray-200 dark:border-gray-700 p-4 bg-gray-50 dark:bg-gray-900">
      {#if editing}
        <!-- Edit Mode -->
        <div class="space-y-4">
          <div>
            <label for="step-title-{stepIndex}" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Title</label>
            <input
              id="step-title-{stepIndex}"
              type="text"
              bind:value={step.title}
              class="w-full bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-gray-900 dark:text-white"
            />
          </div>
          
          <div>
            <label for="step-instructions-{stepIndex}" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Instructions</label>
            <textarea
              id="step-instructions-{stepIndex}"
              bind:value={step.instructions}
              rows="4"
              class="w-full bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-gray-900 dark:text-white"
            ></textarea>
          </div>
          
          <div>
            <label for="step-time-{stepIndex}" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Estimated Time (minutes)</label>
            <input
              id="step-time-{stepIndex}"
              type="number"
              bind:value={step.estimatedTimeMinutes}
              class="w-full bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-gray-900 dark:text-white"
            />
          </div>
          
          <div class="flex items-center gap-2">
            <input
              type="checkbox"
              bind:checked={step.requiresApproval}
              id="approval-{stepIndex}"
              class="rounded"
            />
            <label for="approval-{stepIndex}" class="text-sm text-gray-700 dark:text-gray-300">
              Requires Approval
            </label>
          </div>
          
          <div class="flex gap-2">
            <button
              onclick={onsave}
              type="button"
              class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-sm"
            >
              Save
            </button>
            <button
              onclick={oncancel}
              type="button"
              class="bg-gray-600 hover:bg-gray-700 text-white px-3 py-1 rounded text-sm"
            >
              Cancel
            </button>
          </div>
        </div>
      {:else}
        <!-- View Mode -->
        <div class="space-y-3">
          {#if step.instructions}
            <div>
              <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Instructions</h4>
              <p class="text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
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

          <button
            onclick={onedit}
            type="button"
            class="text-blue-600 hover:text-blue-700 text-sm font-medium"
          >
            Edit Step
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>
