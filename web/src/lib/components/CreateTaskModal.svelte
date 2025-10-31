<script lang="ts">
  import { taskStore } from '$lib/stores/task';
  import { sopStore } from '$lib/stores/sop';
  import type { SOPTemplate } from '$lib/api/sop';
  import { X, Search, FileText, PlusCircle } from 'lucide-svelte';
  import { onMount } from 'svelte';

  interface CreateTaskModalProps {
    isOpen: boolean;
    onClose: () => void;
  }

  let { isOpen, onClose }: CreateTaskModalProps = $props();

  let sops: SOPTemplate[] = $state([]);
  let loading = $state(true);
  let searchQuery = $state('');
  let selectedSOP: SOPTemplate | null = $state(null);
  let assignedTo = $state('');
  let error = $state('');

  // Subscribe to sopStore to get available SOPs
  sopStore.subscribe((state) => {
    sops = state.sops || [];
    loading = state.loading;
  });

  onMount(() => {
    // Load SOPs when modal opens
    if (isOpen && sops.length === 0) {
      sopStore.loadAllSOPs();
    }
  });

  // Filter SOPs based on search query
  let filteredSOPs = $derived(
    sops.filter(sop => 
      sop.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      sop.currentVersion?.description?.toLowerCase().includes(searchQuery.toLowerCase())
    )
  );

  function handleSelectSOP(sop: SOPTemplate) {
    selectedSOP = sop;
  }

  function handleBackToList() {
    selectedSOP = null;
    error = '';
  }

  function handleCreateTask() {
    error = '';

    if (!selectedSOP) {
      error = 'Please select an SOP template';
      return;
    }

    try {
      taskStore.createTask(selectedSOP, assignedTo.trim() || undefined);
      // Reset and close
      selectedSOP = null;
      assignedTo = '';
      searchQuery = '';
      onClose();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to create task';
    }
  }

  function handleClose() {
    selectedSOP = null;
    assignedTo = '';
    searchQuery = '';
    error = '';
    onClose();
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      handleClose();
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen}
  <div 
    class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4"
    role="dialog"
    aria-modal="true"
    onclick={handleClose}
  >
    <div 
      class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col"
      onclick={(e) => e.stopPropagation()}
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <div>
          <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Create New Task</h2>
          <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {#if selectedSOP}
              Configure task details
            {:else}
              Select an SOP template to create a new task
            {/if}
          </p>
        </div>
        <button
          onclick={handleClose}
          class="p-2 rounded-lg text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          aria-label="Close"
        >
          <X class="w-6 h-6" />
        </button>
      </div>

      <!-- Content -->
      <div class="flex-1 overflow-y-auto p-6">
        {#if error}
          <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-4">
            <p class="font-medium">Error</p>
            <p class="text-sm">{error}</p>
          </div>
        {/if}

        {#if !selectedSOP}
          <!-- SOP Selection View -->
          <div class="space-y-4">
            <!-- Search Bar -->
            <div class="relative">
              <Search class="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                type="text"
                bind:value={searchQuery}
                placeholder="Search SOPs by name or description..."
                class="w-full pl-10 pr-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
              />
            </div>

            {#if loading}
              <div class="text-center py-12">
                <div class="inline-block animate-spin rounded-full h-12 w-12 border-4 border-gray-300 border-t-emerald-600"></div>
                <p class="text-gray-600 dark:text-gray-400 mt-4">Loading SOPs...</p>
              </div>
            {:else if filteredSOPs.length === 0}
              <div class="text-center py-12">
                <FileText class="w-16 h-16 text-gray-400 mx-auto mb-4" />
                <p class="text-gray-600 dark:text-gray-400 text-lg font-medium">
                  {searchQuery ? 'No SOPs match your search' : 'No SOPs available'}
                </p>
                <p class="text-gray-500 dark:text-gray-500 text-sm mt-2">
                  {searchQuery ? 'Try a different search term' : 'Create an SOP template first to create tasks'}
                </p>
              </div>
            {:else}
              <!-- SOP List -->
              <div class="grid gap-3">
                {#each filteredSOPs as sop}
                  <button
                    onclick={() => handleSelectSOP(sop)}
                    class="text-left p-4 border-2 border-gray-200 dark:border-gray-600 hover:border-emerald-500 dark:hover:border-emerald-500 rounded-lg transition-colors bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
                  >
                    <div class="flex items-start gap-3">
                      <FileText class="w-5 h-5 text-emerald-600 mt-1 flex-shrink-0" />
                      <div class="flex-1 min-w-0">
                        <h3 class="font-semibold text-gray-900 dark:text-white text-lg">
                          {sop.name}
                        </h3>
                        {#if sop.currentVersion?.description}
                          <p class="text-sm text-gray-600 dark:text-gray-400 mt-1 line-clamp-2">
                            {sop.currentVersion.description}
                          </p>
                        {/if}
                        <div class="flex items-center gap-4 mt-2 text-xs text-gray-500 dark:text-gray-400">
                          <span>{sop.currentVersion?.steps?.length || 0} steps</span>
                          {#if sop.currentVersion?.materials && sop.currentVersion.materials.length > 0}
                            <span>• {sop.currentVersion.materials.length} materials</span>
                          {/if}
                          {#if sop.currentVersion?.equipment && sop.currentVersion.equipment.length > 0}
                            <span>• {sop.currentVersion.equipment.length} equipment</span>
                          {/if}
                        </div>
                      </div>
                      <PlusCircle class="w-5 h-5 text-emerald-600 flex-shrink-0" />
                    </div>
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        {:else}
          <!-- Task Configuration View -->
          <div class="space-y-6">
            <!-- Back Button -->
            <button
              onclick={handleBackToList}
              class="text-sm text-emerald-600 hover:text-emerald-700 dark:text-emerald-400 dark:hover:text-emerald-300 font-medium flex items-center gap-2"
            >
              <span>← Back to SOP list</span>
            </button>

            <!-- Selected SOP Info -->
            <div class="bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800 rounded-lg p-4">
              <div class="flex items-start gap-3">
                <FileText class="w-6 h-6 text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-1" />
                <div>
                  <h3 class="font-semibold text-gray-900 dark:text-white text-lg">
                    {selectedSOP.name}
                  </h3>
                  {#if selectedSOP.currentVersion?.description}
                    <p class="text-sm text-gray-700 dark:text-gray-300 mt-1">
                      {selectedSOP.currentVersion.description}
                    </p>
                  {/if}
                  <div class="flex items-center gap-4 mt-2 text-xs text-gray-600 dark:text-gray-400">
                    <span class="font-medium">{selectedSOP.currentVersion?.steps?.length || 0} steps</span>
                    {#if selectedSOP.currentVersion?.materials && selectedSOP.currentVersion.materials.length > 0}
                      <span>• {selectedSOP.currentVersion.materials.length} materials</span>
                    {/if}
                    {#if selectedSOP.currentVersion?.equipment && selectedSOP.currentVersion.equipment.length > 0}
                      <span>• {selectedSOP.currentVersion.equipment.length} equipment</span>
                    {/if}
                  </div>
                </div>
              </div>
            </div>

            <!-- Task Settings -->
            <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Task Settings</h3>
              
              <div class="space-y-4">
                <div>
                  <label for="assignedTo" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Assign To (Optional)
                  </label>
                  <input
                    id="assignedTo"
                    type="text"
                    bind:value={assignedTo}
                    placeholder="Enter name or leave blank..."
                    class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                  />
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Optionally assign this task to a team member
                  </p>
                </div>
              </div>
            </div>

            <!-- SOP Preview -->
            <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">SOP Preview</h3>
              
              {#if selectedSOP.currentVersion?.materials && selectedSOP.currentVersion.materials.length > 0}
                <div class="mb-4">
                  <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Materials Required:</h4>
                  <ul class="space-y-1">
                    {#each selectedSOP.currentVersion.materials as material}
                      <li class="text-sm text-gray-600 dark:text-gray-400">• {material}</li>
                    {/each}
                  </ul>
                </div>
              {/if}

              {#if selectedSOP.currentVersion?.equipment && selectedSOP.currentVersion.equipment.length > 0}
                <div class="mb-4">
                  <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Equipment Required:</h4>
                  <ul class="space-y-1">
                    {#each selectedSOP.currentVersion.equipment as item}
                      <li class="text-sm text-gray-600 dark:text-gray-400">• {item}</li>
                    {/each}
                  </ul>
                </div>
              {/if}

              {#if selectedSOP.currentVersion?.steps && selectedSOP.currentVersion.steps.length > 0}
                <div>
                  <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Steps:</h4>
                  <ol class="space-y-2">
                    {#each selectedSOP.currentVersion.steps as step}
                      <li class="text-sm text-gray-600 dark:text-gray-400">
                        <span class="font-medium text-gray-900 dark:text-white">
                          {step.stepNumber}. {step.title}
                        </span>
                        {#if step.estimatedTimeMinutes}
                          <span class="text-xs text-gray-500 dark:text-gray-500 ml-2">
                            (~{step.estimatedTimeMinutes} min)
                          </span>
                        {/if}
                      </li>
                    {/each}
                  </ol>
                </div>
              {/if}
            </div>
          </div>
        {/if}
      </div>

      <!-- Footer -->
      <div class="px-6 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50">
        <div class="flex gap-3 justify-end">
          <button
            onclick={handleClose}
            class="px-6 py-2.5 rounded-lg font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
          >
            Cancel
          </button>
          {#if selectedSOP}
            <button
              onclick={handleCreateTask}
              class="px-6 py-2.5 rounded-lg font-medium text-white bg-emerald-600 hover:bg-emerald-700 transition-colors flex items-center gap-2"
            >
              <PlusCircle class="w-4 h-4" />
              Create Task
            </button>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}
