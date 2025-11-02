<script lang="ts">
  import type { SOPStep } from '$lib/api/sop';
  import CollapsibleStep from '$lib/components/CollapsibleStep.svelte';
  
  interface SOPFormProps {
    onSubmit: (data: {
      name: string;
      description?: string;
      materials?: string[];
      equipment?: string[];
      steps: Omit<SOPStep, 'id'>[];
      changeSummary?: string;
    }) => void;
    onCancel?: () => void;
    isSubmitting?: boolean;
    submitButtonText?: string;
    showCancelButton?: boolean;
    isEditMode?: boolean;
    initialData?: {
      name?: string;
      description?: string;
      materials?: string[];
      equipment?: string[];
      steps?: Omit<SOPStep, 'id'>[];
    };
  }
  
  let { 
    onSubmit, 
    onCancel, 
    isSubmitting = false,
    submitButtonText = 'Create SOP',
    showCancelButton = true,
    isEditMode = false,
    initialData
  }: SOPFormProps = $props();

  let name = $state(initialData?.name || '');
  let description = $state(initialData?.description || '');
  let materials: string[] = $state(initialData?.materials || []);
  let equipment: string[] = $state(initialData?.equipment || []);
  let newMaterialInput = $state('');
  let newEquipmentInput = $state('');
  let changeSummary = $state('');
  let steps: Omit<SOPStep, 'id'>[] = $state(
    initialData?.steps && initialData.steps.length > 0
      ? initialData.steps
      : [
          {
            order: 'a',
            title: '',
            instructions: '',
            estimatedTimeMinutes: undefined,
            imageUrl: '',
            videoUrl: '',
            requiresApproval: false
          }
        ]
  );
  let error = $state('');
  let expandedSteps = $state<Set<number>>(new Set());
  let stepRefs: CollapsibleStep[] = [];

  function addStep(focusOnNew = false) {
    const newIndex = steps.length;
    // Generate order string: 'a', 'b', 'c', ..., 'z', 'aa', 'ab', etc.
    const order = String.fromCharCode(97 + (newIndex % 26)) + (newIndex >= 26 ? Math.floor(newIndex / 26).toString() : '');
    steps = [
      ...steps,
      {
        order,
        title: '',
        instructions: '',
        estimatedTimeMinutes: undefined,
        imageUrl: '',
        videoUrl: '',
        requiresApproval: false
      }
    ];
    
    // Focus on the new step's title after it's rendered
    if (focusOnNew) {
      setTimeout(() => {
        stepRefs[newIndex]?.focusTitle();
      }, 100);
    }
  }

  function handleStepNext(currentIndex: number) {
    const nextIndex = currentIndex + 1;
    
    // If there's already a next step, focus on it
    if (nextIndex < steps.length) {
      setTimeout(() => {
        stepRefs[nextIndex]?.focusTitle();
      }, 100);
    } else {
      // Otherwise, create a new step
      addStep(true);
    }
  }
  
  function updateStep(index: number, updatedStep: Omit<SOPStep, 'id'>) {
    steps = steps.map((step, i) => i === index ? updatedStep : step);
  }
  
  function toggleStepExpanded(index: number) {
    if (expandedSteps.has(index)) {
      expandedSteps.delete(index);
    } else {
      expandedSteps.add(index);
    }
    expandedSteps = new Set(expandedSteps);
  }

  function removeStep(index: number) {
    steps = steps.filter((_, i) => i !== index);
    // Regenerate order strings for remaining steps
    steps = steps.map((step, i) => ({
      ...step,
      order: String.fromCharCode(97 + (i % 26)) + (i >= 26 ? Math.floor(i / 26).toString() : '')
    }));
  }

  function moveStep(index: number, direction: 'up' | 'down') {
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= steps.length) return;

    const newSteps = [...steps];
    [newSteps[index], newSteps[newIndex]] = [newSteps[newIndex], newSteps[index]];
    
    // Regenerate order strings after reordering
    steps = newSteps.map((step, i) => ({
      ...step,
      order: String.fromCharCode(97 + (i % 26)) + (i >= 26 ? Math.floor(i / 26).toString() : '')
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

  function handleSubmit() {
    error = '';
    
    // Validation
    if (!name.trim()) {
      error = 'SOP name is required';
      return;
    }
    
    if (isEditMode && !changeSummary.trim()) {
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

    // Clean up steps data
    const cleanSteps = steps.map(step => ({
      order: step.order,
      title: step.title.trim(),
      instructions: step.instructions?.trim() || undefined,
      estimatedTimeMinutes: step.estimatedTimeMinutes || undefined,
      imageUrl: step.imageUrl?.trim() || undefined,
      videoUrl: step.videoUrl?.trim() || undefined,
      requiresApproval: step.requiresApproval
    }));

    onSubmit({
      name: name.trim(),
      description: description.trim() || undefined,
      materials: materials.length > 0 ? materials : undefined,
      equipment: equipment.length > 0 ? equipment : undefined,
      steps: cleanSteps,
      changeSummary: isEditMode ? changeSummary.trim() : undefined
    });
  }
</script>

<div class="flex flex-col max-h-[85vh]">
  {#if error}
    <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-4">
      <p class="font-medium">Error</p>
      <p class="text-sm">{error}</p>
    </div>
  {/if}

  <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="flex flex-col flex-1 min-h-0">
    <!-- Two Column Layout -->
    <div class="grid grid-cols-1 lg:grid-cols-[1fr_350px] gap-6 flex-1 min-h-0 overflow-hidden">
      <!-- Left Column - Steps -->
      <div class="space-y-4 overflow-y-auto pr-2">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white">Steps</h2>
          <button
            type="button"
            onclick={(e) => { e.preventDefault(); addStep(true); }}
            class="bg-emerald-600 hover:bg-emerald-700 text-white px-4 py-2 rounded-lg text-sm transition-colors"
          >
            + Add Step
          </button>
        </div>

        <div class="space-y-3">
          {#each steps as step, index}
            <CollapsibleStep
              bind:this={stepRefs[index]}
              {step}
              {index}
              isExpanded={expandedSteps.has(index)}
              onUpdate={(updatedStep) => updateStep(index, updatedStep)}
              onNext={() => handleStepNext(index)}
              onRemove={() => removeStep(index)}
              onToggleExpand={() => toggleStepExpanded(index)}
              onMoveUp={() => moveStep(index, 'up')}
              onMoveDown={() => moveStep(index, 'down')}
              canMoveUp={index > 0}
              canMoveDown={index < steps.length - 1}
            />
          {/each}
        </div>
      </div>

      <!-- Right Column - SOP Metadata -->
      <div class="space-y-4 overflow-y-auto">
        <!-- SOP Details Card -->
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-4">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">SOP Details</h3>
          
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
                class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
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
            class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
          ></textarea>
        </div>

        {#if isEditMode}
          <div>
            <label for="changeSummary" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Change Summary <span class="text-red-500">*</span>
            </label>
            <textarea
              id="changeSummary"
              bind:value={changeSummary}
              placeholder="Describe what changed in this version..."
              rows="2"
              class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
              required
            ></textarea>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              This helps track what changed between versions
            </p>
          </div>
        {/if}

        <!-- Materials -->
        <div>
          <label for="materials-input" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
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
                      onclick={() => removeMaterial(index)}
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
                id="materials-input"
                type="text"
                bind:value={newMaterialInput}
                placeholder="Add material (e.g., Safety goggles)..."
                class="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                onkeydown={(e) => e.key === 'Enter' && (e.preventDefault(), addMaterial())}
              />
              <button
                type="button"
                onclick={addMaterial}
                class="bg-emerald-600 hover:bg-emerald-700 text-white px-4 py-2 rounded-lg text-sm transition-colors"
              >
                Add
              </button>
            </div>
          </div>
        </div>

        <!-- Equipment -->
        <div>
          <label for="equipment-input" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
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
                      onclick={() => removeEquipment(index)}
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
                id="equipment-input"
                type="text"
                bind:value={newEquipmentInput}
                placeholder="Add equipment (e.g., Oscilloscope)..."
                class="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                onkeydown={(e) => e.key === 'Enter' && (e.preventDefault(), addEquipment())}
              />
              <button
                type="button"
                onclick={addEquipment}
                class="bg-emerald-600 hover:bg-emerald-700 text-white px-4 py-2 rounded-lg text-sm transition-colors"
              >
                Add
              </button>
            </div>
          </div>
        </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Submit Buttons -->
    <div class="border-t border-gray-200 dark:border-gray-700 pt-6 mt-6">
      <div class="flex gap-4">
        {#if showCancelButton}
          <button
            type="button"
            onclick={onCancel}
            class="flex-1 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-900 dark:text-white px-6 py-3 rounded-lg font-medium transition-colors"
            disabled={isSubmitting}
          >
            Cancel
          </button>
        {/if}
        <button
          type="submit"
          class="flex-1 bg-emerald-600 hover:bg-emerald-700 text-white px-6 py-3 rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          disabled={isSubmitting}
        >
          {isSubmitting ? 'Creating...' : submitButtonText}
        </button>
      </div>
    </div>
  </form>
</div>
