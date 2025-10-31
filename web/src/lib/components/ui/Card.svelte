<script lang="ts">
  import type { Snippet } from 'svelte';
  
  interface CardProps {
    class?: string;
    children?: Snippet;
  }
  
  let { class: className = '', children }: CardProps = $props();
</script>

<div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm {className}">
  {#if children}
    {@render children()}
  {/if}
</div>
