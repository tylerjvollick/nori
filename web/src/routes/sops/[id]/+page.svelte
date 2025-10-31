<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { sopStore } from '$lib/stores/sop';
  import type { SOPStep } from '$lib/api/sop';

  let sopId: number;
  let showVersionHistory = false;
  let editingTitle = false;
  let editingDescription = false;
  let editingMaterials = false;
  let editingEquipment = false;
  let expandedSteps = new Set<number>();
  let editingSteps = new Set<number>();

  // Local editable state
  let localTitle = '';
  let localDescription = '';
  let localMaterials: string[] = [];
  let localEquipment: string[] = [];
  let localSteps: SOPStep[] = [];
  let newMaterialInput = '';
  let newEquipmentInput = '';

  $: sopId = parseInt($page.params.id || '0');
  $: if ($sopStore.currentSOP) {
    localTitle = $sopStore.currentSOP.name;
    localDescription = $sopStore.currentSOP.currentVersion?.description || '';
    localMaterials = $sopStore.currentSOP.currentVersion?.materials || [];
    localEquipment = $sopStore.currentSOP.currentVersion?.equipment || [];
    localSteps = $sopStore.currentSOP.currentVersion?.steps || [];
  }

  onMount(async () => {
    await sopStore.loadSOP(sopId);
  });

  async function loadVersionHistory() {
    if (!showVersionHistory) {
      await sopStore.loadSOPVersions(sopId);
    }
    showVersionHistory = !showVersionHistory;
  }

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function getTotalEstimatedTime(steps: SOPStep[]): number {
    return steps.reduce((total, step) => total + (step.estimatedTimeMinutes || 0), 0);
  }

  function toggleStep(stepId: number) {
    if (expandedSteps.has(stepId)) {
      expandedSteps.delete(stepId);
    } else {
      expandedSteps.add(stepId);
    }
    expandedSteps = expandedSteps;
  }

  async function saveTitle() {
    if (!$sopStore.currentSOP) return;
    
    try {
      await sopStore.updateSOP(sopId, {
        name: localTitle,
        description: localDescription,
        materials: localMaterials,
        equipment: localEquipment,
        changeSummary: 'Updated SOP title',
        steps: localSteps.map(s => ({
          stepNumber: s.stepNumber,
          title: s.title,
          instructions: s.instructions,
          estimatedTimeMinutes: s.estimatedTimeMinutes,
          imageUrl: s.imageUrl,
          videoUrl: s.videoUrl,
          requiresApproval: s.requiresApproval
        }))
      });
      editingTitle = false;
    } catch (error) {
      console.error('Failed to update title:', error);
    }
  }

  async function saveDescription() {
    if (!$sopStore.currentSOP) return;
    
    try {
      await sopStore.updateSOP(sopId, {
        name: localTitle,
        description: localDescription,
        materials: localMaterials,
        equipment: localEquipment,
        changeSummary: 'Updated SOP description',
        steps: localSteps.map(s => ({
          stepNumber: s.stepNumber,
          title: s.title,
          instructions: s.instructions,
          estimatedTimeMinutes: s.estimatedTimeMinutes,
          imageUrl: s.imageUrl,
          videoUrl: s.videoUrl,
          requiresApproval: s.requiresApproval
        }))
      });
      editingDescription = false;
    } catch (error) {
      console.error('Failed to update description:', error);
    }
  }

  async function saveMaterials() {
    if (!$sopStore.currentSOP) return;
    
    try {
      await sopStore.updateSOP(sopId, {
        name: localTitle,
        description: localDescription,
        materials: localMaterials,
        equipment: localEquipment,
        changeSummary: 'Updated materials list',
        steps: localSteps.map(s => ({
          stepNumber: s.stepNumber,
          title: s.title,
          instructions: s.instructions,
          estimatedTimeMinutes: s.estimatedTimeMinutes,
          imageUrl: s.imageUrl,
          videoUrl: s.videoUrl,
          requiresApproval: s.requiresApproval
        }))
      });
      editingMaterials = false;
    } catch (error) {
      console.error('Failed to update materials:', error);
    }
  }

  async function saveEquipment() {
    if (!$sopStore.currentSOP) return;
    
    try {
      await sopStore.updateSOP(sopId, {
        name: localTitle,
        description: localDescription,
        materials: localMaterials,
        equipment: localEquipment,
        changeSummary: 'Updated equipment list',
        steps: localSteps.map(s => ({
          stepNumber: s.stepNumber,
          title: s.title,
          instructions: s.instructions,
          estimatedTimeMinutes: s.estimatedTimeMinutes,
          imageUrl: s.imageUrl,
          videoUrl: s.videoUrl,
          requiresApproval: s.requiresApproval
        }))
      });
      editingEquipment = false;
    } catch (error) {
      console.error('Failed to update equipment:', error);
    }
  }

  async function saveStep(stepIndex: number) {
    if (!$sopStore.currentSOP) return;
    
    try {
      await sopStore.updateSOP(sopId, {
        name: localTitle,
        description: localDescription,
        materials: localMaterials,
        equipment: localEquipment,
        changeSummary: `Updated step ${localSteps[stepIndex].stepNumber}`,
        steps: localSteps.map(s => ({
          stepNumber: s.stepNumber,
          title: s.title,
          instructions: s.instructions,
          estimatedTimeMinutes: s.estimatedTimeMinutes,
          imageUrl: s.imageUrl,
          videoUrl: s.videoUrl,
          requiresApproval: s.requiresApproval
        }))
      });
      editingSteps.delete(stepIndex);
      editingSteps = editingSteps;
    } catch (error) {
      console.error('Failed to update step:', error);
    }
  }

  function addMaterial() {
    if (newMaterialInput.trim()) {
      localMaterials = [...localMaterials, newMaterialInput.trim()];
      newMaterialInput = '';
    }
  }

  function removeMaterial(index: number) {
    localMaterials = localMaterials.filter((_, i) => i !== index);
  }

  function addEquipment() {
    if (newEquipmentInput.trim()) {
      localEquipment = [...localEquipment, newEquipmentInput.trim()];
      newEquipmentInput = '';
    }
  }

  function removeEquipment(index: number) {
    localEquipment = localEquipment.filter((_, i) => i !== index);
  }

  async function handleDelete() {
    if (confirm('Are you sure you want to delete this SOP? This action cannot be undone.')) {
      try {
        await sopStore.deleteSOP(sopId);
        goto('/sops');
      } catch (error) {
        console.error('Failed to delete SOP:', error);
      }
    }
  }
</script>

<div class="h-full overflow-hidden">
  {#if $sopStore.loading}
    <div class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
    </div>
  {:else if $sopStore.error}
    <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg m-4">
      <p class="font-medium">Error loading SOP</p>
      <p class="text-sm">{$sopStore.error}</p>
    </div>
  {:else if $sopStore.currentSOP}
    <!-- Two Column Layout -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 h-full">
      <!-- Left Column: Main Content -->
      <div class="lg:col-span-2 flex flex-col overflow-hidden">
        <!-- Sticky Breadcrumb -->
        <div class="sticky top-0 z-10 bg-white dark:bg-gray-900 py-4 px-4 border-b border-gray-200 dark:border-gray-700">
          <nav class="flex items-center text-sm text-gray-600 dark:text-gray-400">
            <button
              on:click={() => goto('/sops')}
              class="hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
            >
              SOPs
            </button>
            <span class="mx-2">/</span>
            <span class="text-gray-900 dark:text-white font-medium truncate">
              {localTitle}
            </span>
          </nav>
        </div>

        <!-- Scrollable Content -->
        <div class="flex-1 overflow-y-auto px-4 py-6 space-y-6">
          <!-- Title -->
          <div class="pb-6 border-b border-gray-200 dark:border-gray-700">
            {#if editingTitle}
              <div class="space-y-3">
                <input
                  type="text"
                  bind:value={localTitle}
                  class="w-full text-3xl font-bold text-gray-900 dark:text-white bg-gray-50 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  on:keydown={(e) => {
                    if (e.key === 'Enter') saveTitle();
                    if (e.key === 'Escape') editingTitle = false;
                  }}
                />
                <div class="flex gap-2">
                  <button
                    on:click={saveTitle}
                    class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-sm"
                  >
                    Save
                  </button>
                  <button
                    on:click={() => {
                      editingTitle = false;
                      localTitle = $sopStore.currentSOP?.name || '';
                    }}
                    class="bg-gray-600 hover:bg-gray-700 text-white px-3 py-1 rounded text-sm"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            {:else}
              <h1
                class="text-3xl font-bold text-gray-900 dark:text-white cursor-pointer hover:text-blue-600"
                on:click={() => (editingTitle = true)}
                on:keydown={(e) => e.key === 'Enter' && (editingTitle = true)}
                role="button"
                tabindex="0"
              >
                {localTitle}
              </h1>
            {/if}
            
            {#if $sopStore.currentSOP.currentVersion}
              <p class="text-sm text-gray-600 dark:text-gray-400 mt-2">
                Version {$sopStore.currentSOP.currentVersion.versionNumber} • 
                Last updated: {formatDate($sopStore.currentSOP.updatedAt)}
              </p>
            {/if}
          </div>

          <!-- Description -->
          <div class="pb-6 border-b border-gray-200 dark:border-gray-700">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-3">Description</h2>
          
          {#if editingDescription}
            <div class="space-y-3">
              <textarea
                bind:value={localDescription}
                rows="4"
                class="w-full text-gray-700 dark:text-gray-300 bg-gray-50 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                on:keydown={(e) => {
                  if (e.key === 'Escape') editingDescription = false;
                }}
              ></textarea>
              <div class="flex gap-2">
                <button
                  on:click={saveDescription}
                  class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-sm"
                >
                  Save
                </button>
                <button
                  on:click={() => {
                    editingDescription = false;
                    localDescription = $sopStore.currentSOP?.currentVersion?.description || '';
                  }}
                  class="bg-gray-600 hover:bg-gray-700 text-white px-3 py-1 rounded text-sm"
                >
                  Cancel
                </button>
              </div>
            </div>
          {:else}
            <p
              class="text-gray-700 dark:text-gray-300 cursor-pointer hover:text-blue-600"
              on:click={() => (editingDescription = true)}
              on:keydown={(e) => e.key === 'Enter' && (editingDescription = true)}
              role="button"
              tabindex="0"
            >
              {localDescription || 'Click to add description'}
            </p>
          {/if}
          </div>

          <!-- Steps -->
          {#if localSteps && localSteps.length > 0}
            <div>
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-4">Steps</h2>
            
            <div class="space-y-2">
              {#each localSteps as step, index}
                <div class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
                  <!-- Collapsed View -->
                  <div
                    class="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-900"
                    on:click={() => toggleStep(step.id || index)}
                    on:keydown={(e) => e.key === 'Enter' && toggleStep(step.id || index)}
                    role="button"
                    tabindex="0"
                  >
                    <div class="flex items-center gap-3 flex-1">
                      <span class="inline-flex items-center justify-center w-8 h-8 bg-blue-600 text-white rounded-full font-bold text-sm flex-shrink-0">
                        {step.stepNumber}
                      </span>
                      <h3 class="text-base font-semibold text-gray-900 dark:text-white">
                        {step.title}
                      </h3>
                    </div>
                    
                    <div class="flex items-center gap-3">
                      {#if step.estimatedTimeMinutes}
                        <span class="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200 px-2 py-1 rounded text-xs">
                          {step.estimatedTimeMinutes} min
                        </span>
                      {/if}
                      <svg
                        class="w-5 h-5 text-gray-600 dark:text-gray-400 transform transition-transform {expandedSteps.has(step.id || index) ? 'rotate-180' : ''}"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </div>
                  </div>

                  <!-- Expanded View -->
                  {#if expandedSteps.has(step.id || index)}
                    <div class="border-t border-gray-200 dark:border-gray-700 p-4 bg-gray-50 dark:bg-gray-900">
                      {#if editingSteps.has(index)}
                        <!-- Edit Mode -->
                        <div class="space-y-4">
                          <div>
                            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Title</label>
                            <input
                              type="text"
                              bind:value={localSteps[index].title}
                              class="w-full bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-gray-900 dark:text-white"
                            />
                          </div>
                          
                          <div>
                            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Instructions</label>
                            <textarea
                              bind:value={localSteps[index].instructions}
                              rows="4"
                              class="w-full bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-gray-900 dark:text-white"
                            ></textarea>
                          </div>
                          
                          <div>
                            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Estimated Time (minutes)</label>
                            <input
                              type="number"
                              bind:value={localSteps[index].estimatedTimeMinutes}
                              class="w-full bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-gray-900 dark:text-white"
                            />
                          </div>
                          
                          <div class="flex items-center gap-2">
                            <input
                              type="checkbox"
                              bind:checked={localSteps[index].requiresApproval}
                              id="approval-{index}"
                              class="rounded"
                            />
                            <label for="approval-{index}" class="text-sm text-gray-700 dark:text-gray-300">
                              Requires Approval
                            </label>
                          </div>
                          
                          <div class="flex gap-2">
                            <button
                              on:click={() => saveStep(index)}
                              class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-sm"
                            >
                              Save
                            </button>
                            <button
                              on:click={() => {
                                editingSteps.delete(index);
                                editingSteps = editingSteps;
                                localSteps = $sopStore.currentSOP?.currentVersion?.steps || [];
                              }}
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
                            on:click={() => {
                              editingSteps.add(index);
                              editingSteps = editingSteps;
                            }}
                            class="text-blue-600 hover:text-blue-700 text-sm font-medium"
                          >
                            Edit Step
                          </button>
                        </div>
                      {/if}
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
            </div>
          {/if}
        </div>
      </div>

      <!-- Right Column: Sidebar -->
      <div class="overflow-y-auto px-4 py-6 space-y-6 border-l border-gray-200 dark:border-gray-700">
        <!-- Actions -->
        <div class="pb-6 border-b border-gray-200 dark:border-gray-700">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-3">Actions</h3>
          <div class="space-y-2">
            <button
              on:click={loadVersionHistory}
              class="w-full bg-blue-600 hover:bg-blue-700 text-white px-3 py-2 rounded-lg text-sm transition-colors"
            >
              {showVersionHistory ? 'Hide' : 'Show'} Version History
            </button>
            <button
              on:click={handleDelete}
              class="w-full bg-red-600 hover:bg-red-700 text-white px-3 py-2 rounded-lg text-sm transition-colors"
            >
              Delete SOP
            </button>
          </div>
        </div>

        <!-- Metadata -->
        <div class="pb-6 border-b border-gray-200 dark:border-gray-700">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-3">Details</h3>
          <div class="space-y-3 text-sm">
            <div>
              <span class="text-gray-600 dark:text-gray-400">Created:</span>
              <span class="text-gray-900 dark:text-white ml-2">{formatDate($sopStore.currentSOP.createdAt)}</span>
            </div>
            <div>
              <span class="text-gray-600 dark:text-gray-400">Updated:</span>
              <span class="text-gray-900 dark:text-white ml-2">{formatDate($sopStore.currentSOP.updatedAt)}</span>
            </div>
            {#if $sopStore.currentSOP.currentVersion}
              <div>
                <span class="text-gray-600 dark:text-gray-400">Version:</span>
                <span class="text-gray-900 dark:text-white ml-2">{$sopStore.currentSOP.currentVersion.versionNumber}</span>
              </div>
            {/if}
          </div>
        </div>

        <!-- Summary Stats -->
        {#if localSteps.length > 0}
          <div class="pb-6 border-b border-gray-200 dark:border-gray-700">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-3">Summary</h3>
            <div class="space-y-3">
              <div>
                <div class="text-xs text-gray-600 dark:text-gray-400 mb-1">Total Steps</div>
                <div class="text-xl font-bold text-gray-900 dark:text-white">
                  {localSteps.length}
                </div>
              </div>
              
              <div>
                <div class="text-xs text-gray-600 dark:text-gray-400 mb-1">Estimated Time</div>
                <div class="text-xl font-bold text-gray-900 dark:text-white">
                  {getTotalEstimatedTime(localSteps)} min
                </div>
              </div>
              
              <div>
                <div class="text-xs text-gray-600 dark:text-gray-400 mb-1">Approval Required</div>
                <div class="text-xl font-bold text-gray-900 dark:text-white">
                  {localSteps.filter(s => s.requiresApproval).length}
                </div>
              </div>
            </div>
          </div>
        {/if}

        <!-- Materials -->
        <div class="pb-6 border-b border-gray-200 dark:border-gray-700">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-3">Materials</h3>
          
          {#if editingMaterials}
            <div class="space-y-3">
              <ul class="space-y-2">
                {#each localMaterials as material, index}
                  <li class="flex items-center justify-between text-sm text-gray-700 dark:text-gray-300">
                    <span>• {material}</span>
                    <button
                      on:click={() => removeMaterial(index)}
                      class="text-red-600 hover:text-red-700 text-xs"
                    >
                      Remove
                    </button>
                  </li>
                {/each}
              </ul>
              
              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={newMaterialInput}
                  placeholder="Add material..."
                  class="flex-1 bg-gray-50 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded px-2 py-1 text-sm"
                  on:keydown={(e) => e.key === 'Enter' && addMaterial()}
                />
                <button
                  on:click={addMaterial}
                  class="bg-green-600 hover:bg-green-700 text-white px-2 py-1 rounded text-xs"
                >
                  Add
                </button>
              </div>
              
              <div class="flex gap-2">
                <button
                  on:click={saveMaterials}
                  class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-sm"
                >
                  Save
                </button>
                <button
                  on:click={() => {
                    editingMaterials = false;
                    localMaterials = $sopStore.currentSOP?.currentVersion?.materials || [];
                  }}
                  class="bg-gray-600 hover:bg-gray-700 text-white px-3 py-1 rounded text-sm"
                >
                  Cancel
                </button>
              </div>
            </div>
          {:else}
            <div>
              {#if localMaterials.length > 0}
                <ul class="space-y-1 text-sm text-gray-700 dark:text-gray-300 mb-3">
                  {#each localMaterials as material}
                    <li>• {material}</li>
                  {/each}
                </ul>
              {:else}
                <p class="text-sm text-gray-500 dark:text-gray-500 mb-3">No materials listed</p>
              {/if}
              
              <button
                on:click={() => (editingMaterials = true)}
                class="text-blue-600 hover:text-blue-700 text-sm font-medium"
              >
                Edit Materials
              </button>
            </div>
          {/if}
        </div>

        <!-- Equipment -->
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white mb-3">Equipment</h3>
          
          {#if editingEquipment}
            <div class="space-y-3">
              <ul class="space-y-2">
                {#each localEquipment as equipment, index}
                  <li class="flex items-center justify-between text-sm text-gray-700 dark:text-gray-300">
                    <span>• {equipment}</span>
                    <button
                      on:click={() => removeEquipment(index)}
                      class="text-red-600 hover:text-red-700 text-xs"
                    >
                      Remove
                    </button>
                  </li>
                {/each}
              </ul>
              
              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={newEquipmentInput}
                  placeholder="Add equipment..."
                  class="flex-1 bg-gray-50 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded px-2 py-1 text-sm"
                  on:keydown={(e) => e.key === 'Enter' && addEquipment()}
                />
                <button
                  on:click={addEquipment}
                  class="bg-green-600 hover:bg-green-700 text-white px-2 py-1 rounded text-xs"
                >
                  Add
                </button>
              </div>
              
              <div class="flex gap-2">
                <button
                  on:click={saveEquipment}
                  class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-sm"
                >
                  Save
                </button>
                <button
                  on:click={() => {
                    editingEquipment = false;
                    localEquipment = $sopStore.currentSOP?.currentVersion?.equipment || [];
                  }}
                  class="bg-gray-600 hover:bg-gray-700 text-white px-3 py-1 rounded text-sm"
                >
                  Cancel
                </button>
              </div>
            </div>
          {:else}
            <div>
              {#if localEquipment.length > 0}
                <ul class="space-y-1 text-sm text-gray-700 dark:text-gray-300 mb-3">
                  {#each localEquipment as equipment}
                    <li>• {equipment}</li>
                  {/each}
                </ul>
              {:else}
                <p class="text-sm text-gray-500 dark:text-gray-500 mb-3">No equipment listed</p>
              {/if}
              
              <button
                on:click={() => (editingEquipment = true)}
                class="text-blue-600 hover:text-blue-700 text-sm font-medium"
              >
                Edit Equipment
              </button>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>

<!-- Version History Modal/Overlay (Outside main grid) -->
{#if $sopStore.currentSOP && showVersionHistory}
  <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl border border-gray-200 dark:border-gray-700 max-w-4xl w-full max-h-[80vh] overflow-y-auto">
      <div class="sticky top-0 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 p-6 flex items-center justify-between">
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white">Version History</h2>
        <button
          on:click={() => showVersionHistory = false}
          class="text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      
      <div class="p-6">
        
        {#if $sopStore.loading}
          <div class="flex justify-center py-6">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        {:else if $sopStore.currentVersions.length === 0}
          <p class="text-gray-600 dark:text-gray-400">No version history available</p>
        {:else}
          <div class="space-y-4">
            {#each $sopStore.currentVersions as version}
              <div class="border border-gray-200 dark:border-gray-600 rounded-lg p-4 {version.isActive ? 'bg-blue-50 dark:bg-blue-900/20 border-blue-300 dark:border-blue-700' : ''}">
                <div class="flex justify-between items-start mb-2">
                  <div class="flex items-center gap-3">
                    <span class="text-lg font-bold text-gray-900 dark:text-white">
                      Version {version.versionNumber}
                    </span>
                    {#if version.isActive}
                      <span class="bg-blue-600 text-white px-2 py-1 rounded text-xs font-medium">
                        Current
                      </span>
                    {/if}
                  </div>
                  <span class="text-sm text-gray-600 dark:text-gray-400">
                    {formatDate(version.createdAt)}
                  </span>
                </div>
                
                {#if version.changeSummary}
                  <p class="text-sm text-gray-700 dark:text-gray-300 mb-2">
                    <span class="font-medium">Changes:</span> {version.changeSummary}
                  </p>
                {/if}
                
                {#if version.description}
                  <p class="text-sm text-gray-600 dark:text-gray-400">
                    {version.description}
                  </p>
                {/if}
                
                {#if version.steps}
                  <p class="text-xs text-gray-500 dark:text-gray-500 mt-2">
                    {version.steps.length} steps
                  </p>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
