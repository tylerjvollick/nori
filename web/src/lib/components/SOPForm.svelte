<script lang="ts">
  import type { SOPStep } from '$lib/api/sop';
  import CollapsibleStep from '$lib/components/CollapsibleStep.svelte';
  import { Collapsible, CollapsibleTrigger, CollapsibleContent } from '$lib/components/ui/collapsible';
  import { ChevronDown } from 'lucide-svelte';
  
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
  
  // Collapsible section state
  let materialsOpen = $state(false);
  let equipmentOpen = $state(false);
  let stepsOpen = $state(true); // Open by default

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

<div class="space-y-6">
  {#if error}
    <div class="bg-destructive/10 border border-destructive/30 text-destructive px-4 py-3 rounded-lg">
      <p class="font-medium">Error</p>
      <p class="text-sm">{error}</p>
    </div>
  {/if}

  <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-6">
    <!-- Name -->
    <div>
      <label for="name" class="block text-sm font-medium text-foreground mb-2">
        Name <span class="text-destructive">*</span>
      </label>
      <input
        id="name"
        type="text"
        bind:value={name}
        placeholder="e.g., Equipment Maintenance Procedure"
        class="w-full px-4 py-2 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
        required
      />
    </div>

    <!-- Description -->
    <div>
      <label for="description" class="block text-sm font-medium text-foreground mb-2">
        Description
      </label>
      <textarea
        id="description"
        bind:value={description}
        placeholder="Brief overview of this SOP..."
        rows="3"
        class="w-full px-4 py-2 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
      ></textarea>
    </div>

    {#if isEditMode}
      <div>
        <label for="changeSummary" class="block text-sm font-medium text-foreground mb-2">
          Change Summary <span class="text-destructive">*</span>
        </label>
        <textarea
          id="changeSummary"
          bind:value={changeSummary}
          placeholder="Describe what changed in this version..."
          rows="2"
          class="w-full px-4 py-2 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
          required
        ></textarea>
        <p class="text-xs text-muted-foreground mt-1">
          This helps track what changed between versions
        </p>
      </div>
    {/if}

    <!-- Materials (Collapsible) -->
    <Collapsible bind:open={materialsOpen}>
      <div class="border border-border rounded-lg">
        <CollapsibleTrigger class="w-full flex items-center justify-between px-4 py-3 hover:bg-accent transition-colors">
          <div class="flex items-center gap-2">
            <ChevronDown class="w-4 h-4 text-muted-foreground transition-transform {materialsOpen ? 'rotate-180' : ''}" />
            <span class="font-medium text-foreground">Materials</span>
            {#if materials.length > 0}
              <span class="text-xs px-2 py-0.5 bg-primary/10 text-primary border border-primary/20 rounded-full">
                {materials.length}
              </span>
            {/if}
          </div>
        </CollapsibleTrigger>
        
        <CollapsibleContent>
          <div class="px-4 pb-4 pt-2 space-y-2 border-t border-border">
            {#if materials.length > 0}
              <ul class="space-y-2 mb-3">
                {#each materials as material, index}
                  <li class="flex items-center justify-between bg-background px-3 py-2 rounded border border-border">
                    <span class="text-sm text-foreground">• {material}</span>
                    <button
                      type="button"
                      onclick={() => removeMaterial(index)}
                      class="text-destructive hover:text-destructive/80 text-xs font-medium"
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
                class="flex-1 px-4 py-2 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
                onkeydown={(e) => e.key === 'Enter' && (e.preventDefault(), addMaterial())}
              />
              <button
                type="button"
                onclick={addMaterial}
                class="bg-primary hover:bg-primary/90 text-white px-4 py-2 rounded-lg text-sm transition-colors"
              >
                Add
              </button>
            </div>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>

    <!-- Equipment (Collapsible) -->
    <Collapsible bind:open={equipmentOpen}>
      <div class="border border-border rounded-lg">
        <CollapsibleTrigger class="w-full flex items-center justify-between px-4 py-3 hover:bg-accent transition-colors">
          <div class="flex items-center gap-2">
            <ChevronDown class="w-4 h-4 text-muted-foreground transition-transform {equipmentOpen ? 'rotate-180' : ''}" />
            <span class="font-medium text-foreground">Equipment</span>
            {#if equipment.length > 0}
              <span class="text-xs px-2 py-0.5 bg-primary/10 text-primary border border-primary/20 rounded-full">
                {equipment.length}
              </span>
            {/if}
          </div>
        </CollapsibleTrigger>
        
        <CollapsibleContent>
          <div class="px-4 pb-4 pt-2 space-y-2 border-t border-border">
            {#if equipment.length > 0}
              <ul class="space-y-2 mb-3">
                {#each equipment as item, index}
                  <li class="flex items-center justify-between bg-background px-3 py-2 rounded border border-border">
                    <span class="text-sm text-foreground">• {item}</span>
                    <button
                      type="button"
                      onclick={() => removeEquipment(index)}
                      class="text-destructive hover:text-destructive/80 text-xs font-medium"
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
                class="flex-1 px-4 py-2 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
                onkeydown={(e) => e.key === 'Enter' && (e.preventDefault(), addEquipment())}
              />
              <button
                type="button"
                onclick={addEquipment}
                class="bg-primary hover:bg-primary/90 text-white px-4 py-2 rounded-lg text-sm transition-colors"
              >
                Add
              </button>
            </div>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>

    <!-- Steps (Collapsible) -->
    <Collapsible bind:open={stepsOpen}>
      <div class="border border-border rounded-lg">
        <CollapsibleTrigger class="w-full flex items-center justify-between px-4 py-3 hover:bg-accent transition-colors">
          <div class="flex items-center gap-2">
            <ChevronDown class="w-4 h-4 text-muted-foreground transition-transform {stepsOpen ? 'rotate-180' : ''}" />
            <span class="font-medium text-foreground">Steps</span>
            <span class="text-xs px-2 py-0.5 bg-primary/10 text-primary border border-primary/20 rounded-full">
              {steps.length}
            </span>
          </div>
        </CollapsibleTrigger>
        
        <CollapsibleContent>
          <div class="px-4 pb-4 pt-2 space-y-3 border-t border-border">
            <div class="flex justify-end">
              <button
                type="button"
                onclick={(e) => { e.preventDefault(); addStep(true); }}
                class="bg-primary hover:bg-primary/90 text-white px-4 py-2 rounded-lg text-sm transition-colors"
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
        </CollapsibleContent>
      </div>
    </Collapsible>
  </form>
</div>
