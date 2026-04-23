<script lang="ts">
  import { onMount } from 'svelte';
  import { sopStore } from '$lib/stores/sop';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import * as Tabs from '$lib/components/ui/tabs';
  import * as Card from '$lib/components/ui/card';
  import * as Alert from '$lib/components/ui/alert';
  import * as Dialog from '$lib/components/ui/dialog';
  import { Badge } from '$lib/components/ui/badge';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import { CircleAlert } from '@lucide/svelte';
  import CreateSOPModal from '$lib/components/CreateSOPModal.svelte';
  import type { SOPTemplate } from '$lib/api/sop';
  
  let searchQuery = $state('');
  let activeTab = $state<string>('published');
  let showCreateModal = $state(false);
  let deleteDialogOpen = $state(false);
  let draftToDelete = $state<{ id: string; name: string } | null>(null);

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

  function openDeleteDialog(draftId: string, draftName: string) {
    draftToDelete = { id: draftId, name: draftName };
    deleteDialogOpen = true;
  }

  async function confirmDelete() {
    if (draftToDelete) {
      await sopStore.deleteDraft(draftToDelete.id);
      deleteDialogOpen = false;
      draftToDelete = null;
    }
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

  <!-- Tabs + Search -->
  <Tabs.Root bind:value={activeTab}>
    <Tabs.List>
      <Tabs.Trigger value="published">
        Published SOPs ({$sopStore.sops.length})
      </Tabs.Trigger>
      <Tabs.Trigger value="drafts">
        My Drafts ({$sopStore.drafts.length})
      </Tabs.Trigger>
    </Tabs.List>

    <!-- Search Bar -->
    <div class="mb-6">
      <Input
        type="text"
        bind:value={searchQuery}
        placeholder="Search SOPs by name or description..."
        class="h-10"
      />
    </div>

    <!-- Published SOPs Tab -->
    <Tabs.Content value="published">
      {#if $sopStore.loading}
        <!-- Skeleton card placeholders -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {#each Array(6) as _}
            <Card.Root>
              <Card.Header>
                <div class="flex items-start justify-between">
                  <Skeleton class="h-6 w-3/4" />
                  <Skeleton class="h-5 w-12" />
                </div>
              </Card.Header>
              <Card.Content class="px-0 py-0">
                <div class="px-4">
                  <Skeleton class="h-4 w-full mb-2" />
                  <Skeleton class="h-4 w-2/3" />
                </div>
              </Card.Content>
              <Card.Footer class="px-4 py-3">
                <div class="flex items-center justify-between w-full">
                  <Skeleton class="h-3 w-16" />
                  <Skeleton class="h-3 w-24" />
                </div>
              </Card.Footer>
            </Card.Root>
          {/each}
        </div>
      {:else if $sopStore.error}
        <Alert.Root variant="destructive">
          <CircleAlert />
          <Alert.Title>Error loading SOPs</Alert.Title>
          <Alert.Description>{$sopStore.error}</Alert.Description>
        </Alert.Root>
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
              class="block group"
            >
              <Card.Root class="h-full transition-all hover:shadow-lg hover:ring-primary/50">
                <Card.Header>
                  <div class="flex items-start justify-between">
                    <Card.Title class="text-xl group-hover:text-primary transition-colors">
                      {sop.name}
                    </Card.Title>
                    <div class="flex gap-2 shrink-0 ml-2">
                      {#if sop.activeDraftId}
                        <Badge variant="outline" class="bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 border-yellow-200 dark:border-yellow-800">
                          Draft
                        </Badge>
                      {/if}
                      <Badge variant="secondary">
                        v{sop.currentVersion?.versionNumber || '1'}
                      </Badge>
                    </div>
                  </div>
                </Card.Header>
                <Card.Content class="px-0 py-0">
                  <div class="px-4">
                    {#if sop.currentVersion?.description}
                      <p class="text-sm text-muted-foreground line-clamp-3">
                        {sop.currentVersion.description}
                      </p>
                    {:else}
                      <p class="text-sm text-muted-foreground/70 italic">
                        No description
                      </p>
                    {/if}
                  </div>
                </Card.Content>
                <Card.Footer class="px-4 py-3">
                  <div class="flex items-center justify-between w-full text-xs text-muted-foreground border-t border-border pt-3">
                    <span>
                      {sop.currentVersion?.steps?.length || 0} steps
                    </span>
                    <span>
                      Updated {formatDate(sop.updatedAt)}
                    </span>
                  </div>
                </Card.Footer>
              </Card.Root>
            </a>
          {/each}
        </div>

        <div class="mt-8 text-center text-sm text-muted-foreground">
          Showing {filteredSOPs.length} of {$sopStore.sops.length} SOP{$sopStore.sops.length === 1 ? '' : 's'}
        </div>
      {/if}
    </Tabs.Content>

    <!-- Drafts Tab -->
    <Tabs.Content value="drafts">
      {#if $sopStore.loading}
        <!-- Skeleton card placeholders -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {#each Array(3) as _}
            <Card.Root>
              <Card.Header>
                <div class="flex items-start justify-between">
                  <Skeleton class="h-6 w-3/4" />
                  <Skeleton class="h-5 w-20" />
                </div>
              </Card.Header>
              <Card.Content class="px-0 py-0">
                <div class="px-4">
                  <Skeleton class="h-4 w-full mb-2" />
                  <Skeleton class="h-4 w-1/2" />
                </div>
              </Card.Content>
              <Card.Footer class="px-4 py-3">
                <div class="flex gap-2 w-full">
                  <Skeleton class="h-9 flex-1" />
                  <Skeleton class="h-9 w-16" />
                </div>
              </Card.Footer>
            </Card.Root>
          {/each}
        </div>
      {:else if $sopStore.error}
        <Alert.Root variant="destructive">
          <CircleAlert />
          <Alert.Title>Error loading drafts</Alert.Title>
          <Alert.Description>{$sopStore.error}</Alert.Description>
        </Alert.Root>
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
            <Card.Root class="transition-all hover:shadow-lg">
              <Card.Header>
                <div class="flex items-start justify-between">
                  <Card.Title class="text-xl">
                    {draft.sopTemplateName}
                  </Card.Title>
                  <Badge variant="outline" class="bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 border-yellow-200 dark:border-yellow-800 shrink-0 ml-2">
                    Draft v{draft.versionNumber}
                  </Badge>
                </div>
              </Card.Header>
              <Card.Content class="px-0 py-0">
                <div class="px-4">
                  {#if draft.changeSummary}
                    <p class="text-sm text-muted-foreground line-clamp-3">
                      {draft.changeSummary}
                    </p>
                  {:else}
                    <p class="text-sm text-muted-foreground/70 italic">
                      No change summary
                    </p>
                  {/if}
                </div>
              </Card.Content>
              <Card.Footer class="px-4 py-3">
                <div class="w-full">
                  <div class="flex items-center justify-between text-xs text-muted-foreground border-t border-border pt-3 mb-3">
                    <span>
                      Version {draft.versionNumber}
                    </span>
                    <span>
                      Updated {formatDate(draft.updatedAt)}
                    </span>
                  </div>
                  <div class="flex gap-2">
                    <Button href="/sops/{draft.sopTemplateId}?draftId={draft.id}" class="flex-1">
                      Continue Editing
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onclick={() => openDeleteDialog(draft.id, draft.sopTemplateName)}
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              </Card.Footer>
            </Card.Root>
          {/each}
        </div>

        <div class="mt-8 text-center text-sm text-muted-foreground">
          Showing {filteredDrafts.length} of {$sopStore.drafts.length} draft{$sopStore.drafts.length === 1 ? '' : 's'}
        </div>
      {/if}
    </Tabs.Content>
  </Tabs.Root>
</div>

<!-- Delete Confirmation Dialog -->
<Dialog.Root bind:open={deleteDialogOpen} onOpenChange={(open) => { if (!open) draftToDelete = null; }}>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>Delete Draft</Dialog.Title>
      <Dialog.Description>
        Are you sure you want to delete the draft for "{draftToDelete?.name}"? This action cannot be undone.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer>
      <Button variant="outline" onclick={() => { deleteDialogOpen = false; }}>
        Cancel
      </Button>
      <Button variant="destructive" onclick={confirmDelete}>
        Delete
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<!-- Create SOP Modal -->
<CreateSOPModal isOpen={showCreateModal} onClose={() => showCreateModal = false} />
