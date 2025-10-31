<script lang="ts">
  import type { Snippet } from 'svelte';
  
  interface ButtonProps {
    variant?: 'default' | 'outline' | 'ghost' | 'destructive';
    size?: 'default' | 'sm' | 'lg';
    class?: string;
    onclick?: () => void;
    disabled?: boolean;
    children?: Snippet;
  }
  
  let { 
    variant = 'default', 
    size = 'default', 
    class: className = '', 
    onclick,
    disabled = false,
    children 
  }: ButtonProps = $props();
  
  const variants = {
    default: 'bg-emerald-600 hover:bg-emerald-700 text-white',
    outline: 'border border-slate-300 hover:bg-slate-100 text-slate-700',
    ghost: 'hover:bg-slate-100 text-slate-700',
    destructive: 'bg-red-600 hover:bg-red-700 text-white'
  };
  
  const sizes = {
    default: 'px-4 py-2',
    sm: 'px-3 py-1.5 text-sm',
    lg: 'px-6 py-3 text-lg'
  };
</script>

<button
  class="inline-flex items-center justify-center rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed {variants[variant]} {sizes[size]} {className}"
  {onclick}
  {disabled}
>
  {#if children}
    {@render children()}
  {/if}
</button>
