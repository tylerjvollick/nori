<script lang="ts">
  import { onMount } from 'svelte';
  import { sopStore } from '$lib/stores/sop';
  import Button from '$lib/components/ui/Button.svelte';
  import CreateSOPModal from '$lib/components/CreateSOPModal.svelte';
  import type { SOPTemplate } from '$lib/api/sop';
  
  let searchQuery = $state('');
  let activeTab = $state<'published' | 'drafts'>('published');
  let showCreateModal = $state(false);

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
      <h1 class="text-3xl font-bold text-foreground">SOP Templates</h1>
      <p class="text-muted-foreground mt-1">Browse and manage your standard operating procedures</p>
    </div>
    <Button onclick={() => showCreateModal = true}>Create New SOP</Button>
  </div>

  <!-- Tabs -->
  <div class="flex border-b border-border mb-6">
    <button
      class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'published' ? 'text-primary border-b-2 border-primary' : 'text-muted-foreground hover:text-foreground'}"
      on:click={() => activeTab = 'published'}
    >
      Published SOPs ({$sopStore.sops.length})
    </button>
    <button
      class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'drafts' ? 'text-primary border-b-2 border-primary' : 'text-muted-foreground hover:text-foreground'}"
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
      class="w-full px-4 py-3 border border-border rounded-lg bg-card text-foreground focus:ring-2 focus:ring-ring focus:border-ring"
    />
  </div>

  {#if activeTab === 'published'}
    <!-- Published SOPs Tab -->
    {#if $sopStore.loading}
      <div class="flex justify-center items-center py-12">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    {:else if $sopStore.error}
      <div class="bg-destructive/10 border border-destructive/30 text-destructive px-4 py-3 rounded-lg">
        <p class="font-medium">Error loading SOPs</p>
        <p class="text-sm">{$sopStore.error}</p>
      </div>
    {:else if !$sopStore.sops || $sopStore.sops.length === 0}
      <div class="text-center py-12">
        <p class="text-muted-foreground mb-4">No SOPs created yet. Create your first SOP template to get started.</p>
        <Button onclick={() => showCreateModal = true}>Create Your First SOP</Button>
      </div>
    {:else if filteredSOPs.length === 0}
      <div class="text-center py-12">
        <p class="text-muted-foreground">No SOPs match your search.</p>
      </div>
    {:else}
      <!-- SOP Grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {#each filteredSOPs as sop}
          <a 
            href={sop.activeDraftId ? `/sops/${sop.id}?draftId=${sop.activeDraftId}` : `/sops/${sop.id}`}
            class="block p-6 bg-card border border-border rounded-lg hover:shadow-lg hover:border-primary transition-all"
          >
            <div class="flex items-start justify-between mb-3">
              <h3 class="text-xl font-semibold text-foreground">
                {sop.name}
              </h3>
              <div class="flex gap-2">
                {#if sop.activeDraftId}
                  <span class="text-xs px-2 py-1 bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 rounded-full border border-yellow-200 dark:border-yellow-800">
                    Draft
                  </span>
                {/if}
                <span class="text-xs px-2 py-1 bg-primary/10 text-primary rounded-full border border-primary/20">
                  v{sop.currentVersion?.versionNumber || '1'}
                </span>
              </div>
            </div>
            
            {#if sop.currentVersion?.description}
              <p class="text-sm text-muted-foreground mb-4 line-clamp-3">
                {sop.currentVersion.description}
              </p>
            {:else}
              <p class="text-sm text-muted-foreground/70 italic mb-4">
                No description
              </p>
            {/if}

            <div class="flex items-center justify-between text-xs text-muted-foreground pt-4 border-t border-border">
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

      <div class="mt-8 text-center text-sm text-muted-foreground">
        Showing {filteredSOPs.length} of {$sopStore.sops.length} SOP{$sopStore.sops.length === 1 ? '' : 's'}
      </div>
    {/if}
  {:else}
    <!-- Drafts Tab -->
    {#if $sopStore.loading}
      <div class="flex justify-center items-center py-12">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
      </div>
    {:else if $sopStore.error}
      <div class="bg-destructive/10 border border-destructive/30 text-destructive px-4 py-3 rounded-lg">
        <p class="font-medium">Error loading drafts</p>
        <p class="text-sm">{$sopStore.error}</p>
      </div>
    {:else if !$sopStore.drafts || $sopStore.drafts.length === 0}
      <div class="text-center py-12">
        <p class="text-muted-foreground mb-4">No draft SOPs yet. Start editing an SOP and save it as a draft.</p>
      </div>
    {:else if filteredDrafts.length === 0}
      <div class="text-center py-12">
        <p class="text-muted-foreground">No drafts match your search.</p>
      </div>
    {:else}
      <!-- Drafts Grid -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {#each filteredDrafts as draft}
          <div class="block p-6 bg-card border border-border rounded-lg hover:shadow-lg transition-all">
            <div class="flex items-start justify-between mb-3">
              <h3 class="text-xl font-semibold text-foreground">
                {draft.sopTemplateName}
              </h3>
              <span class="text-xs px-2 py-1 bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 border border-yellow-200 dark:border-yellow-800 rounded-full">
                Draft v{draft.versionNumber}
              </span>
            </div>
            
            {#if draft.changeSummary}
              <p class="text-sm text-muted-foreground mb-4 line-clamp-3">
                {draft.changeSummary}
              </p>
            {:else}
              <p class="text-sm text-muted-foreground/70 italic mb-4">
                No change summary
              </p>
            {/if}

            <div class="flex items-center justify-between text-xs text-muted-foreground pt-4 border-t border-border mb-3">
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
                class="flex-1 bg-primary hover:bg-primary/90 text-white text-center px-3 py-2 rounded text-sm transition-colors"
              >
                Continue Editing
              </a>
              <button
                on:click={async () => {
                  if (confirm('Are you sure you want to delete this draft?')) {
                    await sopStore.deleteDraft(draft.id);
                  }
                }}
                class="bg-destructive hover:bg-destructive/90 text-destructive-foreground px-3 py-2 rounded text-sm transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        {/each}
      </div>

      <div class="mt-8 text-center text-sm text-muted-foreground">
        Showing {filteredDrafts.length} of {$sopStore.drafts.length} draft{$sopStore.drafts.length === 1 ? '' : 's'}
      </div>
    {/if}
  {/if}
</div>

<!-- Create SOP Modal -->
<CreateSOPModal isOpen={showCreateModal} onClose={() => showCreateModal = false} />
