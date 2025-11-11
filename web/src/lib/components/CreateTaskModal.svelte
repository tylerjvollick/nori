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
      class="bg-card rounded-xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col"
      onclick={(e) => e.stopPropagation()}
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-border">
        <div>
          <h2 class="text-2xl font-bold text-foreground">Create New Task</h2>
          <p class="text-sm text-muted-foreground mt-1">
            {#if selectedSOP}
              Configure task details
            {:else}
              Select an SOP template to create a new task
            {/if}
          </p>
        </div>
        <button
          onclick={handleClose}
          class="p-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          aria-label="Close"
        >
          <X class="w-6 h-6" />
        </button>
      </div>

      <!-- Content -->
      <div class="flex-1 overflow-y-auto p-6">
        {#if error}
          <div class="bg-destructive/10 border border-destructive/20 text-destructive px-4 py-3 rounded-lg mb-4">
            <p class="font-medium">Error</p>
            <p class="text-sm">{error}</p>
          </div>
        {/if}

        {#if !selectedSOP}
          <!-- SOP Selection View -->
          <div class="space-y-4">
            <!-- Search Bar -->
            <div class="relative">
              <Search class="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-muted-foreground" />
              <input
                type="text"
                bind:value={searchQuery}
                placeholder="Search SOPs by name or description..."
                class="w-full pl-10 pr-4 py-3 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
              />
            </div>

            {#if loading}
              <div class="text-center py-12">
                <div class="inline-block animate-spin rounded-full h-12 w-12 border-4 border-border border-t-primary"></div>
                <p class="text-muted-foreground mt-4">Loading SOPs...</p>
              </div>
            {:else if filteredSOPs.length === 0}
              <div class="text-center py-12">
                <FileText class="w-16 h-16 text-muted-foreground mx-auto mb-4" />
                <p class="text-muted-foreground text-lg font-medium">
                  {searchQuery ? 'No SOPs match your search' : 'No SOPs available'}
                </p>
                <p class="text-muted-foreground text-sm mt-2">
                  {searchQuery ? 'Try a different search term' : 'Create an SOP template first to create tasks'}
                </p>
              </div>
            {:else}
              <!-- SOP List -->
              <div class="grid gap-3">
                {#each filteredSOPs as sop}
                  <button
                    onclick={() => handleSelectSOP(sop)}
                    class="text-left p-4 border-2 border-border hover:border-primary500 dark:hover:border-primary500 rounded-lg transition-colors bg-card hover:bg-secondary/50 dark:hover:bg-secondary/600"
                  >
                    <div class="flex items-start gap-3">
                      <FileText class="w-5 h-5 text-primary mt-1 flex-shrink-0" />
                      <div class="flex-1 min-w-0">
                        <h3 class="font-semibold text-foreground text-lg">
                          {sop.name}
                        </h3>
                        {#if sop.currentVersion?.description}
                          <p class="text-sm text-muted-foreground mt-1 line-clamp-2">
                            {sop.currentVersion.description}
                          </p>
                        {/if}
                        <div class="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                          <span>{sop.currentVersion?.steps?.length || 0} steps</span>
                          {#if sop.currentVersion?.materials && sop.currentVersion.materials.length > 0}
                            <span>• {sop.currentVersion.materials.length} materials</span>
                          {/if}
                          {#if sop.currentVersion?.equipment && sop.currentVersion.equipment.length > 0}
                            <span>• {sop.currentVersion.equipment.length} equipment</span>
                          {/if}
                        </div>
                      </div>
                      <PlusCircle class="w-5 h-5 text-primary flex-shrink-0" />
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
              class="text-sm text-primary hover:text-accent-foreground dark:hover:text-primary300 font-medium flex items-center gap-2"
            >
              <span>← Back to SOP list</span>
            </button>

            <!-- Selected SOP Info -->
            <div class="bg-primary/10 border border-primary/20 rounded-lg p-4">
              <div class="flex items-start gap-3">
                <FileText class="w-6 h-6 text-primary flex-shrink-0 mt-1" />
                <div>
                  <h3 class="font-semibold text-foreground text-lg">
                    {selectedSOP.name}
                  </h3>
                  {#if selectedSOP.currentVersion?.description}
                    <p class="text-sm text-foreground mt-1">
                      {selectedSOP.currentVersion.description}
                    </p>
                  {/if}
                  <div class="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
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
            <div class="bg-card rounded-lg border border-border p-6">
              <h3 class="text-lg font-semibold text-foreground mb-4">Task Settings</h3>
              
              <div class="space-y-4">
                <div>
                  <label for="assignedTo" class="block text-sm font-medium text-foreground mb-2">
                    Assign To (Optional)
                  </label>
                  <input
                    id="assignedTo"
                    type="text"
                    bind:value={assignedTo}
                    placeholder="Enter name or leave blank..."
                    class="w-full px-4 py-2 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
                  />
                  <p class="text-xs text-muted-foreground mt-1">
                    Optionally assign this task to a team member
                  </p>
                </div>
              </div>
            </div>

            <!-- SOP Preview -->
            <div class="bg-card rounded-lg border border-border p-6">
              <h3 class="text-lg font-semibold text-foreground mb-4">SOP Preview</h3>
              
              {#if selectedSOP.currentVersion?.materials && selectedSOP.currentVersion.materials.length > 0}
                <div class="mb-4">
                  <h4 class="text-sm font-medium text-foreground mb-2">Materials Required:</h4>
                  <ul class="space-y-1">
                    {#each selectedSOP.currentVersion.materials as material}
                      <li class="text-sm text-muted-foreground">• {material}</li>
                    {/each}
                  </ul>
                </div>
              {/if}

              {#if selectedSOP.currentVersion?.equipment && selectedSOP.currentVersion.equipment.length > 0}
                <div class="mb-4">
                  <h4 class="text-sm font-medium text-foreground mb-2">Equipment Required:</h4>
                  <ul class="space-y-1">
                    {#each selectedSOP.currentVersion.equipment as item}
                      <li class="text-sm text-muted-foreground">• {item}</li>
                    {/each}
                  </ul>
                </div>
              {/if}

              {#if selectedSOP.currentVersion?.steps && selectedSOP.currentVersion.steps.length > 0}
                <div>
                  <h4 class="text-sm font-medium text-foreground mb-2">Steps:</h4>
                  <ol class="space-y-2">
                    {#each selectedSOP.currentVersion.steps as step, index}
                      <li class="text-sm text-muted-foreground">
                        <span class="font-medium text-foreground">
                          {index + 1}. {step.title}
                        </span>
                        {#if step.estimatedTimeMinutes}
                          <span class="text-xs text-muted-foreground ml-2">
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
      <div class="px-6 py-4 border-t border-border bg-background/50">
        <div class="flex gap-3 justify-end">
          <button
            onclick={handleClose}
            class="px-6 py-2.5 rounded-lg font-medium text-foreground bg-card border border-border hover:bg-accent transition-colors"
          >
            Cancel
          </button>
          {#if selectedSOP}
            <button
              onclick={handleCreateTask}
              class="px-6 py-2.5 rounded-lg font-medium text-white bg-primary hover:bg-primary/90 transition-colors flex items-center gap-2"
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
