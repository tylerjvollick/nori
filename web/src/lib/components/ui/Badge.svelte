<script lang="ts">
  import type { Snippet } from 'svelte';
  
  interface BadgeProps {
    variant?: 'default' | 'outline' | 'secondary';
    class?: string;
    children?: Snippet;
  }
  
  let { variant = 'default', class: className = '', children }: BadgeProps = $props();
  
  const variants = {
    default: 'bg-emerald-100 text-emerald-700 border-emerald-200',
    outline: 'border border-slate-200 text-slate-700',
    secondary: 'bg-slate-100 text-slate-700'
  };
</script>

<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border {variants[variant]} {className}">
  {#if children}
    {@render children()}
  {/if}
</span>
