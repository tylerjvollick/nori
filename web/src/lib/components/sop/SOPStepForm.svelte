<script lang="ts">
  import { Button } from '$lib/components/ui/button';

  interface Props {
    stepCount: number;
    onsave: () => void;
    oncancel: () => void;
    onkeydown: (e: KeyboardEvent) => void;
    newStepTitle: string;
    inputRef?: HTMLInputElement;
  }

  let {
    stepCount,
    onsave,
    oncancel,
    onkeydown,
    newStepTitle = $bindable(),
    inputRef = $bindable()
  }: Props = $props();
</script>

<div class="border border-primary/30 rounded-lg overflow-hidden bg-primary/5">
  <div class="flex items-center gap-3 p-4">
    <span class="inline-flex items-center justify-center w-8 h-8 bg-primary text-primary-foreground rounded-full font-bold text-sm flex-shrink-0">
      {stepCount + 1}
    </span>
    <input
      bind:this={inputRef}
      bind:value={newStepTitle}
      onkeydown={onkeydown}
      type="text"
      placeholder="Enter step title and press Enter or Tab..."
      class="flex-1 bg-card border border-border rounded-lg px-3 py-2 text-foreground focus:ring-2 focus:ring-blue-500 focus:border-transparent"
    />
    <Button
      onclick={oncancel}
      type="button"
      variant="ghost"
      size="icon"
      aria-label="Cancel"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
      </svg>
    </Button>
  </div>
</div>
