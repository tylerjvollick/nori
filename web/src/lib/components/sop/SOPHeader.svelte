<script lang="ts">
  import { goto } from '$app/navigation';
  import { Button } from '$lib/components/ui/button';

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
    <Button
      onclick={() => goto('/sops')}
      variant="link"
      class="h-auto p-0"
    >
      SOPs
    </Button>
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
        <Button
          onclick={onsave}
          size="sm"
        >
          Save
        </Button>
        <Button
          onclick={oncanceledit}
          variant="secondary"
          size="sm"
        >
          Cancel
        </Button>
      </div>
    </div>
  {:else}
    <Button
      variant="ghost"
      onclick={onstartedit}
      class="text-3xl font-bold text-left w-full h-auto justify-start p-0"
      type="button"
    >
      {title}
    </Button>
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
