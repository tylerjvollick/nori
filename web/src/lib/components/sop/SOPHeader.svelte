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
<div class="sticky top-0 z-10 bg-background py-4 px-4 border-b border-border">
  <nav class="flex items-center text-sm text-muted-foreground">
    <button
      onclick={() => goto('/sops')}
      class="hover:text-primary600 dark:text-primary400 dark:hover:text-primary400 transition-colors"
    >
      SOPs
    </button>
    <span class="mx-2">/</span>
    <span class="text-foreground font-medium truncate">
      {title}
    </span>
  </nav>
</div>

<!-- Title Section -->
<div class="px-4 pt-4 pb-6 border-b border-border">
  {#if editing}
    <div class="space-y-3">
      <input
        type="text"
        bind:value={title}
        class="w-full text-3xl font-bold text-foreground bg-background border border-border rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        onkeydown={handleKeydown}
      />
      <div class="flex gap-2">
        <button
          onclick={onsave}
          class="bg-primary hover:bg-primary/90 text-primary-foreground px-3 py-1 rounded text-sm"
        >
          Save
        </button>
        <button
          onclick={oncanceledit}
          class="bg-secondary/600 hover:bg-secondary/700 text-white px-3 py-1 rounded text-sm"
        >
          Cancel
        </button>
      </div>
    </div>
  {:else}
    <button
      class="text-3xl font-bold text-foreground hover:text-primary600 dark:text-primary400 text-left w-full"
      onclick={onstartedit}
      type="button"
    >
      {title}
    </button>
  {/if}
  
  {#if isDraftMode}
    <div class="flex items-center gap-2 mt-2">
      <span class="text-xs px-2 py-1 bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 border border-yellow-200 dark:border-yellow-800 rounded-full font-medium">
        DRAFT MODE
      </span>
      <p class="text-sm text-muted-foreground">
        Last updated: {formatDate(lastUpdated)}
      </p>
    </div>
  {:else if versionNumber}
    <p class="text-sm text-muted-foreground mt-2">
      Version {versionNumber} • 
      Last updated: {formatDate(lastUpdated)}
    </p>
  {/if}
</div>
