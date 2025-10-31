<script lang="ts">
  import { slide } from 'svelte/transition';
  
  interface CollapsibleProps {
    open?: boolean;
    class?: string;
    children?: import('svelte').Snippet;
  }
  
  let { 
    open = $bindable(false),
    class: className = '',
    children
  }: CollapsibleProps = $props();
</script>

<div class="collapsible {className}">
  {@render children?.()}
</div>

<style>
  .collapsible {
    display: block;
  }
</style>
