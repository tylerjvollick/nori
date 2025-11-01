<script lang="ts">
  import { onMount } from 'svelte';
  import { sopStore } from '$lib/stores/sop';
  import Button from '$lib/components/ui/Button.svelte';
  import type { SOPTemplate } from '$lib/api/sop';
  
  let searchQuery = $state('');
  let activeTab = $state<'published' | 'drafts'>('published');

  onMount(() => {
    sopStore.loadAllSOPs();
    sopStore.loadUserDrafts();
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

  // Filter drafts based on search query
  const filteredDrafts = $derived(
    searchQuery.trim()
      ? $sopStore.drafts.filter(draft => 
          draft.sopTemplateName.toLowerCase().includes(searchQuery.toLowerCase()) ||
          draft.changeSummary?.toLowerCase().includes(searchQuery.toLowerCase())
        )
      : $sopStore.drafts
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

  <!-- Tabs -->
  <div class="flex border-b border-slate-200 dark:border-slate-700 mb-6">
    <button
      class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'published' ? 'text-emerald-600 dark:text-emerald-400 border-b-2 border-emerald-600 dark:border-emerald-400' : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'}"
      on:click={() => activeTab = 'published'}
    >
      Published SOPs ({$sopStore.sops.length})
    </button>
    <button
      class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'drafts' ? 'text-emerald-600 dark:text-emerald-400 border-b-2 border-emerald-600 dark:border-emerald-400' : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'}"
      on:click={() => activeTab = 'drafts'}
    >
      My Drafts ({$sopStore.drafts.length})
    </button>
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

  {#if activeTab === 'published'}
    <!-- Published SOPs Tab -->
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
            href={sop.activeDraftId ? `/sops/${sop.id}?draftId=${sop.activeDraftId}` : `/sops/${sop.id}`}
            class="block p-6 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg hover:shadow-lg hover:border-emerald-500 dark:hover:border-emerald-500 transition-all"
          >
            <div class="flex items-start justify-between mb-3">
              <h3 class="text-xl font-semibold text-slate-900 dark:text-white">
                {sop.name}
              </h3>
              <div class="flex gap-2">
                {#if sop.activeDraftId}
                  <span class="text-xs px-2 py-1 bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 rounded-full">
                    Draft
                  </span>
                {/if}
                <span class="text-xs px-2 py-1 bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400 rounded-full">
                  v{sop.currentVersion?.versionNumber || '1'}
                </span>
              </div>
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
  {:else}
    <!-- Drafts Tab -->
    {#if $sopStore.loading}
      <div class="flex justify-center items-center py-12">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-600"></div>
      </div>
    {:else if $sopStore.error}
      <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg dark:bg-red-900/20 dark:border-red-800 dark:text-red-400">
        <p class="font-medium">Error loading drafts</p>
        <p class="text-sm">{$sopStore.error}</p>
      </div>
    {:else if !$sopStore.drafts || $sopStore.drafts.length === 0}
      <div class="text-center py-12">
        <p class="text-slate-500 dark:text-slate-400 mb-4">No draft SOPs yet. Start editing an SOP and save it as a draft.</p>
      </div>
    {:else if filteredDrafts.length === 0}
      <div class="text-center py-12">
        <p class="text-slate-500 dark:text-slate-400">No drafts match your search.</p>
      </div>
    {:else}
      <!-- Drafts Grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {#each filteredDrafts as draft}
          <div class="block p-6 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg hover:shadow-lg transition-all">
            <div class="flex items-start justify-between mb-3">
              <h3 class="text-xl font-semibold text-slate-900 dark:text-white">
                {draft.sopTemplateName}
              </h3>
              <span class="text-xs px-2 py-1 bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 rounded-full">
                Draft v{draft.versionNumber}
              </span>
            </div>
            
            {#if draft.changeSummary}
              <p class="text-sm text-slate-600 dark:text-slate-400 mb-4 line-clamp-3">
                {draft.changeSummary}
              </p>
            {:else}
              <p class="text-sm text-slate-400 dark:text-slate-500 italic mb-4">
                No change summary
              </p>
            {/if}

            <div class="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400 pt-4 border-t border-slate-100 dark:border-slate-700 mb-3">
              <span>
                Version {draft.versionNumber}
              </span>
              <span>
                Updated {formatDate(draft.updatedAt)}
              </span>
            </div>

            <div class="flex gap-2">
              <a 
                href="/sops/{draft.sopTemplateId}?draftId={draft.id}"
                class="flex-1 bg-emerald-600 hover:bg-emerald-700 text-white text-center px-3 py-2 rounded text-sm transition-colors"
              >
                Continue Editing
              </a>
              <button
                on:click={async () => {
                  if (confirm('Are you sure you want to delete this draft?')) {
                    await sopStore.deleteDraft(draft.id);
                  }
                }}
                class="bg-red-600 hover:bg-red-700 text-white px-3 py-2 rounded text-sm transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        {/each}
      </div>

      <div class="mt-8 text-center text-sm text-slate-500 dark:text-slate-400">
        Showing {filteredDrafts.length} of {$sopStore.drafts.length} draft{$sopStore.drafts.length === 1 ? '' : 's'}
      </div>
    {/if}
  {/if}
</div>
