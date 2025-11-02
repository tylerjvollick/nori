<script lang="ts">
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

<div class="border border-blue-500 dark:border-blue-600 rounded-lg overflow-hidden bg-blue-50 dark:bg-blue-900/20">
  <div class="flex items-center gap-3 p-4">
    <span class="inline-flex items-center justify-center w-8 h-8 bg-blue-600 text-white rounded-full font-bold text-sm flex-shrink-0">
      {stepCount + 1}
    </span>
    <input
      bind:this={inputRef}
      bind:value={newStepTitle}
      onkeydown={onkeydown}
      type="text"
      placeholder="Enter step title and press Enter or Tab..."
      class="flex-1 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
    />
    <button
      onclick={oncancel}
      type="button"
      aria-label="Cancel"
      class="text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
      </svg>
    </button>
  </div>
</div>
