<script lang="ts">
  import { goto } from '$app/navigation';

  interface Props {
    title: string;
    editing: boolean;
    isDraftMode: boolean;
    versionNumber?: number;
    lastUpdated: string;
    ontitlechange?: (title: string) => void;
    onstartedit?: () => void;
    oncanceledit?: () => void;
    onsave?: () => void;
  }

  let {
    title = $bindable(),
    editing,
    isDraftMode,
    versionNumber,
    lastUpdated,
    ontitlechange,
    onstartedit,
    oncanceledit,
    onsave
  }: Props = $props();

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && onsave) {
      onsave();
    }
    if (e.key === 'Escape' && oncanceledit) {
      oncanceledit();
    }
  }
</script>

<!-- Breadcrumb -->
<div class="sticky top-0 z-10 bg-white dark:bg-gray-900 py-4 px-4 border-b border-gray-200 dark:border-gray-700">
  <nav class="flex items-center text-sm text-gray-600 dark:text-gray-400">
    <button
      onclick={() => goto('/sops')}
      class="hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
    >
      SOPs
    </button>
    <span class="mx-2">/</span>
    <span class="text-gray-900 dark:text-white font-medium truncate">
      {title}
    </span>
  </nav>
</div>

<!-- Title Section -->
<div class="px-4 pt-4 pb-6 border-b border-gray-200 dark:border-gray-700">
  {#if editing}
    <div class="space-y-3">
      <input
        type="text"
        bind:value={title}
        class="w-full text-3xl font-bold text-gray-900 dark:text-white bg-gray-50 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        onkeydown={handleKeydown}
      />
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
      class="text-3xl font-bold text-gray-900 dark:text-white hover:text-blue-600 text-left w-full"
      onclick={onstartedit}
      type="button"
    >
      {title}
    </button>
  {/if}
  
  {#if isDraftMode}
    <div class="flex items-center gap-2 mt-2">
      <span class="text-xs px-2 py-1 bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 rounded-full font-medium">
        DRAFT MODE
      </span>
      <p class="text-sm text-gray-600 dark:text-gray-400">
        Last updated: {formatDate(lastUpdated)}
      </p>
    </div>
  {:else if versionNumber}
    <p class="text-sm text-gray-600 dark:text-gray-400 mt-2">
      Version {versionNumber} • 
      Last updated: {formatDate(lastUpdated)}
    </p>
  {/if}
</div>
