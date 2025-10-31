<script lang="ts">
  import type { Snippet } from 'svelte';
  
  interface DialogProps {
    open?: boolean;
    onClose?: () => void;
    children?: Snippet;
  }
  
  let { open = $bindable(false), onClose, children }: DialogProps = $props();
  
  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      open = false;
      onClose?.();
    }
  }
  
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      open = false;
      onClose?.();
    }
  }
</script>

{#if open}
  <div
    class="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50"
    onclick={handleBackdropClick}
    onkeydown={handleKeydown}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div class="bg-white dark:bg-gray-800 rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto shadow-xl">
      {#if children}
        {@render children()}
      {/if}
    </div>
  </div>
{/if}
