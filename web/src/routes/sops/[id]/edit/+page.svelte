<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { sopStore } from '$lib/stores/sop';
  import type { SOPStep } from '$lib/api/sop';

  let sopId: number;
  let originalName = '';
  let name = '';
  let description = '';
  let materials: string[] = [];
  let equipment: string[] = [];
  let newMaterialInput = '';
  let newEquipmentInput = '';
  let changeSummary = '';
  let steps: Omit<SOPStep, 'id'>[] = [];
  let error = '';
  let isSubmitting = false;
  let isLoading = true;

  $: sopId = parseInt($page.params.id || '0');

  onMount(async () => {
    try {
      const sop = await sopStore.loadSOP(sopId);
      if (sop && sop.currentVersion) {
        originalName = sop.name;
        name = sop.name;
        description = sop.currentVersion.description || '';
        materials = sop.currentVersion.materials || [];
        equipment = sop.currentVersion.equipment || [];
        
        // Load existing steps
        if (sop.currentVersion.steps && sop.currentVersion.steps.length > 0) {
          steps = sop.currentVersion.steps.map(step => ({
            stepNumber: step.stepNumber,
            title: step.title,
            instructions: step.instructions || '',
            estimatedTimeMinutes: step.estimatedTimeMinutes,
            imageUrl: step.imageUrl || '',
            videoUrl: step.videoUrl || '',
            requiresApproval: step.requiresApproval
          }));
        } else {
          // Default to one empty step if none exist
          steps = [{
            stepNumber: 1,
            title: '',
            instructions: '',
            estimatedTimeMinutes: undefined,
            imageUrl: '',
            videoUrl: '',
            requiresApproval: false
          }];
        }
      }
      isLoading = false;
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load SOP';
      isLoading = false;
    }
  });

  function addStep() {
    steps = [
      ...steps,
      {
        stepNumber: steps.length + 1,
        title: '',
        instructions: '',
        estimatedTimeMinutes: undefined,
        imageUrl: '',
        videoUrl: '',
        requiresApproval: false
      }
    ];
  }

  function removeStep(index: number) {
    steps = steps.filter((_, i) => i !== index);
    // Renumber remaining steps
    steps = steps.map((step, i) => ({
      ...step,
      stepNumber: i + 1
    }));
  }

  function moveStep(index: number, direction: 'up' | 'down') {
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= steps.length) return;

    const newSteps = [...steps];
    [newSteps[index], newSteps[newIndex]] = [newSteps[newIndex], newSteps[index]];
    
    // Renumber steps
    steps = newSteps.map((step, i) => ({
      ...step,
      stepNumber: i + 1
    }));
  }

  function addMaterial() {
    if (newMaterialInput.trim()) {
      materials = [...materials, newMaterialInput.trim()];
      newMaterialInput = '';
    }
  }

  function removeMaterial(index: number) {
    materials = materials.filter((_, i) => i !== index);
  }

  function addEquipment() {
    if (newEquipmentInput.trim()) {
      equipment = [...equipment, newEquipmentInput.trim()];
      newEquipmentInput = '';
    }
  }

  function removeEquipment(index: number) {
    equipment = equipment.filter((_, i) => i !== index);
  }

  async function handleSubmit() {
    error = '';
    
    // Validation
    if (!name.trim()) {
      error = 'SOP name is required';
      return;
    }
    
    if (!changeSummary.trim()) {
      error = 'Change summary is required when updating an SOP';
      return;
    }
    
    if (steps.length === 0) {
      error = 'At least one step is required';
      return;
    }

    const emptySteps = steps.filter(s => !s.title.trim());
    if (emptySteps.length > 0) {
      error = 'All steps must have a title';
      return;
    }

    isSubmitting = true;

    try {
      // Clean up steps data
      const cleanSteps = steps.map(step => ({
        stepNumber: step.stepNumber,
        title: step.title.trim(),
        instructions: step.instructions?.trim() || undefined,
        estimatedTimeMinutes: step.estimatedTimeMinutes || undefined,
        imageUrl: step.imageUrl?.trim() || undefined,
        videoUrl: step.videoUrl?.trim() || undefined,
        requiresApproval: step.requiresApproval
      }));

      await sopStore.updateSOP(sopId, {
        name: name.trim() !== originalName ? name.trim() : undefined,
        description: description.trim() || undefined,
        materials: materials.length > 0 ? materials : undefined,
        equipment: equipment.length > 0 ? equipment : undefined,
        changeSummary: changeSummary.trim(),
        steps: cleanSteps
      });

      goto(`/sops/${sopId}`);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to update SOP';
      isSubmitting = false;
    }
  }
</script>

<div class="container mx-auto px-4 py-8 max-w-4xl">
  {#if isLoading}
    <div class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
    </div>
  {:else}
    <div class="mb-8">
      <button
        on:click={() => goto(`/sops/${sopId}`)}
        class="text-blue-600 hover:text-blue-700 mb-4 inline-flex items-center"
      >
        ← Back to SOP
      </button>
      <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Edit SOP</h1>
      <p class="text-sm text-gray-600 dark:text-gray-400 mt-2">
        Updating this SOP will create a new version
      </p>
    </div>

    {#if error}
      <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-6">
        <p class="font-medium">Error</p>
        <p class="text-sm">{error}</p>
      </div>
    {/if}

    <form on:submit|preventDefault={handleSubmit} class="space-y-6">
      <!-- SOP Details -->
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-6">
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-4">SOP Details</h2>
        
        <div class="space-y-4">
          <div>
            <label for="name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Name <span class="text-red-500">*</span>
            </label>
            <input
              id="name"
              type="text"
              bind:value={name}
              placeholder="e.g., Equipment Maintenance Procedure"
              class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              required
            />
          </div>

          <div>
            <label for="description" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Description
            </label>
            <textarea
              id="description"
              bind:value={description}
              placeholder="Brief overview of this SOP..."
              rows="3"
              class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            ></textarea>
          </div>

          <!-- Materials -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Materials
            </label>
            <div class="space-y-2">
              {#if materials.length > 0}
                <ul class="space-y-2">
                  {#each materials as material, index}
                    <li class="flex items-center justify-between bg-gray-50 dark:bg-gray-900 px-3 py-2 rounded border border-gray-200 dark:border-gray-600">
                      <span class="text-sm text-gray-700 dark:text-gray-300">• {material}</span>
                      <button
                        type="button"
                        on:click={() => removeMaterial(index)}
                        class="text-red-600 hover:text-red-700 text-xs font-medium"
                      >
                        Remove
                      </button>
                    </li>
                  {/each}
                </ul>
              {/if}
              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={newMaterialInput}
                  placeholder="Add material (e.g., Safety goggles)..."
                  class="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  on:keydown={(e) => e.key === 'Enter' && (e.preventDefault(), addMaterial())}
                />
                <button
                  type="button"
                  on:click={addMaterial}
                  class="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg text-sm transition-colors"
                >
                  Add
                </button>
              </div>
            </div>
          </div>

          <!-- Equipment -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Equipment
            </label>
            <div class="space-y-2">
              {#if equipment.length > 0}
                <ul class="space-y-2">
                  {#each equipment as item, index}
                    <li class="flex items-center justify-between bg-gray-50 dark:bg-gray-900 px-3 py-2 rounded border border-gray-200 dark:border-gray-600">
                      <span class="text-sm text-gray-700 dark:text-gray-300">• {item}</span>
                      <button
                        type="button"
                        on:click={() => removeEquipment(index)}
                        class="text-red-600 hover:text-red-700 text-xs font-medium"
                      >
                        Remove
                      </button>
                    </li>
                  {/each}
                </ul>
              {/if}
              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={newEquipmentInput}
                  placeholder="Add equipment (e.g., Oscilloscope)..."
                  class="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  on:keydown={(e) => e.key === 'Enter' && (e.preventDefault(), addEquipment())}
                />
                <button
                  type="button"
                  on:click={addEquipment}
                  class="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg text-sm transition-colors"
                >
                  Add
                </button>
              </div>
            </div>
          </div>

          <div>
            <label for="changeSummary" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Change Summary <span class="text-red-500">*</span>
            </label>
            <textarea
              id="changeSummary"
              bind:value={changeSummary}
              placeholder="Describe what changed in this version..."
              rows="2"
              class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              required
            ></textarea>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              This helps track what changed between versions
            </p>
          </div>
        </div>
      </div>

      <!-- Steps -->
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-6">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">Steps</h2>
          <button
            type="button"
            on:click={addStep}
            class="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-lg text-sm transition-colors"
          >
            + Add Step
          </button>
        </div>

        <div class="space-y-6">
          {#each steps as step, index}
            <div class="border border-gray-200 dark:border-gray-600 rounded-lg p-4">
              <div class="flex justify-between items-start mb-4">
                <h3 class="text-lg font-medium text-gray-900 dark:text-white">
                  Step {step.stepNumber}
                </h3>
                <div class="flex gap-2">
                  <button
                    type="button"
                    on:click={() => moveStep(index, 'up')}
                    disabled={index === 0}
                    class="px-2 py-1 text-sm bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    ↑
                  </button>
                  <button
                    type="button"
                    on:click={() => moveStep(index, 'down')}
                    disabled={index === steps.length - 1}
                    class="px-2 py-1 text-sm bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    ↓
                  </button>
                  <button
                    type="button"
                    on:click={() => removeStep(index)}
                    disabled={steps.length === 1}
                    class="px-3 py-1 text-sm bg-red-600 hover:bg-red-700 text-white rounded disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Remove
                  </button>
                </div>
              </div>

              <div class="space-y-4">
                <div>
                  <label for="step-title-{index}" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Title <span class="text-red-500">*</span>
                  </label>
                  <input
                    id="step-title-{index}"
                    type="text"
                    bind:value={step.title}
                    placeholder="e.g., Inspect equipment for damage"
                    class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    required
                  />
                </div>

                <div>
                  <label for="step-instructions-{index}" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Instructions
                  </label>
                  <textarea
                    id="step-instructions-{index}"
                    bind:value={step.instructions}
                    placeholder="Detailed instructions for this step..."
                    rows="3"
                    class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  ></textarea>
                </div>

                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label for="step-time-{index}" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Estimated Time (minutes)
                    </label>
                    <input
                      id="step-time-{index}"
                      type="number"
                      min="0"
                      bind:value={step.estimatedTimeMinutes}
                      placeholder="e.g., 15"
                      class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label for="step-approval-{index}" class="flex items-center mt-8">
                      <input
                        id="step-approval-{index}"
                        type="checkbox"
                        bind:checked={step.requiresApproval}
                        class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                      />
                      <span class="ml-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                        Requires Approval
                      </span>
                    </label>
                  </div>
                </div>

                <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div>
                    <label for="step-image-{index}" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Image URL
                    </label>
                    <input
                      id="step-image-{index}"
                      type="url"
                      bind:value={step.imageUrl}
                      placeholder="https://example.com/image.jpg"
                      class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                  </div>

                  <div>
                    <label for="step-video-{index}" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Video URL
                    </label>
                    <input
                      id="step-video-{index}"
                      type="url"
                      bind:value={step.videoUrl}
                      placeholder="https://example.com/video.mp4"
                      class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                  </div>
                </div>
              </div>
            </div>
          {/each}
        </div>
      </div>

      <!-- Submit -->
      <div class="flex gap-4">
        <button
          type="button"
          on:click={() => goto(`/sops/${sopId}`)}
          class="flex-1 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-900 dark:text-white px-6 py-3 rounded-lg font-medium transition-colors"
          disabled={isSubmitting}
        >
          Cancel
        </button>
        <button
          type="submit"
          class="flex-1 bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          disabled={isSubmitting}
        >
          {isSubmitting ? 'Saving...' : 'Save Changes (Create New Version)'}
        </button>
      </div>
    </form>
  {/if}
</div>
