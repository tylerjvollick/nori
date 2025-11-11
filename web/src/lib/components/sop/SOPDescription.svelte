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

<div class="pb-6 border-b border-border">
  <h2 class="text-xl font-semibold text-foreground mb-3">Description</h2>

  {#if editing}
    <div class="space-y-3">
      <textarea
        bind:value={description}
        rows="4"
        class="w-full text-foreground bg-background border border-border rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        onkeydown={handleKeydown}
      ></textarea>
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
      class="text-foreground hover:text-primary600 dark:text-primary400 text-left w-full"
      onclick={onstartedit}
      type="button"
    >
      {description || 'Click to add description'}
    </button>
  {/if}
</div>
