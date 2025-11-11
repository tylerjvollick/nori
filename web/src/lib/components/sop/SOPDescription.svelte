<script lang="ts">
  import { Button } from '$lib/components/ui/button';

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
      class="text-left w-full h-auto justify-start p-0"
      type="button"
    >
      {description || 'Click to add description'}
    </Button>
  {/if}
</div>
