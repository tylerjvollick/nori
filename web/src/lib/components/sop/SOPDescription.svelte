<script lang="ts">
  interface Props {
    description: string;
    editing: boolean;
    onstartedit?: () => void;
    oncanceledit?: () => void;
    onsave?: () => void;
  }

  let {
    description = $bindable(),
    editing,
    onstartedit,
    oncanceledit,
    onsave
  }: Props = $props();

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && oncanceledit) {
      oncanceledit();
    }
  }
</script>

<div class="pb-6 border-b border-gray-200 dark:border-gray-700">
  <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-3">Description</h2>

  {#if editing}
    <div class="space-y-3">
      <textarea
        bind:value={description}
        rows="4"
        class="w-full text-gray-700 dark:text-gray-300 bg-gray-50 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        onkeydown={handleKeydown}
      ></textarea>
      <div class="flex gap-2">
        <button
          onclick={onsave}
          class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-sm"
        >
          Save
        </button>
        <button
          onclick={oncanceledit}
          class="bg-gray-600 hover:bg-gray-700 text-white px-3 py-1 rounded text-sm"
        >
          Cancel
        </button>
      </div>
    </div>
  {:else}
    <button
      class="text-gray-700 dark:text-gray-300 hover:text-blue-600 text-left w-full"
      onclick={onstartedit}
      type="button"
    >
      {description || 'Click to add description'}
    </button>
  {/if}
</div>
