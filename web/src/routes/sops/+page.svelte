<script lang="ts">
  import { onMount } from 'svelte';
  import { sopStore } from '$lib/stores/sop';
  import Button from '$lib/components/ui/Button.svelte';
  import type { SOPTemplate } from '$lib/api/sop';
  
  let searchQuery = $state('');

  onMount(() => {
    sopStore.loadAllSOPs();
  });

  // Filter SOPs based on search query
  const filteredSOPs = $derived(
    searchQuery.trim()
      ? $sopStore.sops.filter(sop => 
          sop.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          sop.currentVersion?.description?.toLowerCase().includes(searchQuery.toLowerCase())
        )
      : $sopStore.sops
  );

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  }
</script>

<div class="container mx-auto px-4 py-8">
  <!-- Header -->
  <div class="flex justify-between items-center mb-8">
    <div>
      <h1 class="text-3xl font-bold text-slate-900 dark:text-white">SOP Templates</h1>
      <p class="text-slate-500 dark:text-slate-400 mt-1">Browse and manage your standard operating procedures</p>
    </div>
    <a href="/sops/create">
      <Button>Create New SOP</Button>
    </a>
  </div>

  <!-- Search Bar -->
  <div class="mb-6">
    <input
      type="text"
      bind:value={searchQuery}
      placeholder="Search SOPs by name or description..."
      class="w-full px-4 py-3 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
    />
  </div>

  {#if $sopStore.loading}
    <div class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-600"></div>
    </div>
  {:else if $sopStore.error}
    <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg dark:bg-red-900/20 dark:border-red-800 dark:text-red-400">
      <p class="font-medium">Error loading SOPs</p>
      <p class="text-sm">{$sopStore.error}</p>
    </div>
  {:else if !$sopStore.sops || $sopStore.sops.length === 0}
    <div class="text-center py-12">
      <p class="text-slate-500 dark:text-slate-400 mb-4">No SOPs created yet. Create your first SOP template to get started.</p>
      <a href="/sops/create">
        <Button>Create Your First SOP</Button>
      </a>
    </div>
  {:else if filteredSOPs.length === 0}
    <div class="text-center py-12">
      <p class="text-slate-500 dark:text-slate-400">No SOPs match your search.</p>
    </div>
  {:else}
    <!-- SOP Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {#each filteredSOPs as sop}
        <a 
          href="/sops/{sop.id}"
          class="block p-6 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg hover:shadow-lg hover:border-emerald-500 dark:hover:border-emerald-500 transition-all"
        >
          <div class="flex items-start justify-between mb-3">
            <h3 class="text-xl font-semibold text-slate-900 dark:text-white">
              {sop.name}
            </h3>
            <span class="text-xs px-2 py-1 bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400 rounded-full">
              v{sop.currentVersion?.versionNumber || '1'}
            </span>
          </div>
          
          {#if sop.currentVersion?.description}
            <p class="text-sm text-slate-600 dark:text-slate-400 mb-4 line-clamp-3">
              {sop.currentVersion.description}
            </p>
          {:else}
            <p class="text-sm text-slate-400 dark:text-slate-500 italic mb-4">
              No description
            </p>
          {/if}

          <div class="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400 pt-4 border-t border-slate-100 dark:border-slate-700">
            <span>
              {sop.currentVersion?.steps?.length || 0} steps
            </span>
            <span>
              Updated {formatDate(sop.updatedAt)}
            </span>
          </div>
        </a>
      {/each}
    </div>

    <div class="mt-8 text-center text-sm text-slate-500 dark:text-slate-400">
      Showing {filteredSOPs.length} of {$sopStore.sops.length} SOP{$sopStore.sops.length === 1 ? '' : 's'}
    </div>
  {/if}
</div>
