<script lang="ts">
  import { slide } from 'svelte/transition';
  import { Button } from '$lib/components/ui/button';
  import type { SOPStep } from '$lib/api/sop';
  
  interface CollapsibleStepProps {
    step: Omit<SOPStep, 'id'>;
    index: number;
    isExpanded: boolean;
    onUpdate: (step: Omit<SOPStep, 'id'>) => void;
    onNext: () => void;
    onRemove: () => void;
    onToggleExpand: () => void;
    onMoveUp?: () => void;
    onMoveDown?: () => void;
    canMoveUp: boolean;
    canMoveDown: boolean;
  }
  
  let { 
    step,
    index,
    isExpanded = $bindable(false),
    onUpdate,
    onNext,
    onRemove,
    onToggleExpand,
    onMoveUp,
    onMoveDown,
    canMoveUp,
    canMoveDown
  }: CollapsibleStepProps = $props();
  
  let titleInput = $state<HTMLInputElement>();
  let descriptionTextarea = $state<HTMLTextAreaElement>();
  
  function handleTitleKeydown(e: KeyboardEvent) {
    // Enter (with or without Shift) in title moves to next step
    if (e.key === 'Enter') {
      e.preventDefault();
      onNext();
    } else if (e.key === 'Tab' && !isExpanded) {
      e.preventDefault();
      isExpanded = true;
      onToggleExpand();
      // Focus description after expand animation
      setTimeout(() => {
        descriptionTextarea?.focus();
      }, 100);
    }
  }
  
  function handleDescriptionKeydown(e: KeyboardEvent) {
    if (e.shiftKey && e.key === 'Enter') {
      e.preventDefault();
      // Collapse current step before moving to next
      if (isExpanded) {
        isExpanded = false;
        onToggleExpand();
      }
      onNext();
    }
  }
  
  function updateField(field: keyof Omit<SOPStep, 'id'>, value: any) {
    onUpdate({ ...step, [field]: value });
  }
  
  export function focusTitle() {
    setTimeout(() => {
      titleInput?.focus();
    }, 100);
  }
</script>

<div class="border border-border rounded-lg overflow-hidden bg-card">
  <!-- Collapsed Header View -->
  <div class="flex items-center gap-2 p-3 {isExpanded ? 'border-b border-border' : ''}">
    <!-- Expand/Collapse Button -->
    <Button
      type="button"
      onclick={onToggleExpand}
      variant="ghost"
      size="icon-sm"
      class="flex-shrink-0 transition-transform {isExpanded ? 'rotate-90' : ''}"
      aria-label={isExpanded ? 'Collapse step' : 'Expand step'}
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
      </svg>
    </Button>
    
    <!-- Step Number -->
    <span class="flex-shrink-0 text-sm font-medium text-muted-foreground w-8">
      {index + 1}
    </span>
    
    <!-- Title Input (always visible) -->
    <input
      bind:this={titleInput}
      type="text"
      value={step.title}
      oninput={(e) => updateField('title', e.currentTarget.value)}
      onkeydown={handleTitleKeydown}
      placeholder="Step title (Enter to add new step, Tab to expand)..."
      class="flex-1 px-2 py-1 bg-transparent border-none focus:outline-none focus:ring-2 focus:ring-ring rounded text-foreground placeholder-muted-foreground"
    />
    
    <!-- Action Buttons -->
    <div class="flex-shrink-0 flex items-center gap-1">
      {#if canMoveUp}
        <Button
          type="button"
          onclick={onMoveUp}
          variant="ghost"
          size="icon-sm"
          title="Move up"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
          </svg>
        </Button>
      {/if}
      
      {#if canMoveDown}
        <Button
          type="button"
          onclick={onMoveDown}
          variant="ghost"
          size="icon-sm"
          title="Move down"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </Button>
      {/if}
      
      <Button
        type="button"
        onclick={onRemove}
        variant="ghost"
        size="icon-sm"
        class="text-destructive hover:text-destructive/80 hover:bg-destructive/10"
        title="Remove step"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </Button>
    </div>
  </div>
  
  <!-- Expanded Content -->
  {#if isExpanded}
    <div class="p-4 space-y-4" transition:slide={{ duration: 200 }}>
      <!-- Instructions -->
      <div>
        <label for="step-instructions-{index}" class="block text-sm font-medium text-foreground mb-2">
          Instructions
        </label>
        <textarea
          bind:this={descriptionTextarea}
          id="step-instructions-{index}"
          value={step.instructions}
          oninput={(e) => updateField('instructions', e.currentTarget.value)}
          onkeydown={handleDescriptionKeydown}
          placeholder="Detailed instructions for this step... (Shift+Enter to add new step)"
          rows="3"
          class="w-full px-4 py-2 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
        ></textarea>
      </div>
      
      <!-- Time Estimate and Approval -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label for="step-time-{index}" class="block text-sm font-medium text-foreground mb-2">
            Estimated Time (minutes)
          </label>
          <input
            id="step-time-{index}"
            type="number"
            min="0"
            value={step.estimatedTimeMinutes || ''}
            oninput={(e) => updateField('estimatedTimeMinutes', e.currentTarget.value ? parseInt(e.currentTarget.value) : undefined)}
            placeholder="e.g., 15"
            class="w-full px-4 py-2 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
          />
        </div>
        
        <div>
          <label for="step-approval-{index}" class="flex items-center mt-8">
            <input
              id="step-approval-{index}"
              type="checkbox"
              checked={step.requiresApproval}
              onchange={(e) => updateField('requiresApproval', e.currentTarget.checked)}
              class="w-4 h-4 text-primary border-input rounded focus:ring-ring"
            />
            <span class="ml-2 text-sm font-medium text-foreground">
              Requires Approval
            </span>
          </label>
        </div>
      </div>
      
      <!-- Image and Video URLs -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label for="step-image-{index}" class="block text-sm font-medium text-foreground mb-2">
            Image URL
          </label>
          <input
            id="step-image-{index}"
            type="url"
            value={step.imageUrl || ''}
            oninput={(e) => updateField('imageUrl', e.currentTarget.value)}
            placeholder="https://example.com/image.jpg"
            class="w-full px-4 py-2 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
          />
        </div>
        
        <div>
          <label for="step-video-{index}" class="block text-sm font-medium text-foreground mb-2">
            Video URL
          </label>
          <input
            id="step-video-{index}"
            type="url"
            value={step.videoUrl || ''}
            oninput={(e) => updateField('videoUrl', e.currentTarget.value)}
            placeholder="https://example.com/video.mp4"
            class="w-full px-4 py-2 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-transparent"
          />
        </div>
      </div>
    </div>
  {/if}
</div>
