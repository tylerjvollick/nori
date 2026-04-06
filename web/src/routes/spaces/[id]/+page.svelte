<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { spaceStore } from '$lib/stores/space';
  import { Button } from '$lib/components/ui/button';
  import * as Dialog from '$lib/components/ui/dialog';

  let showSpaceSettings = $state(false);
  let newSpaceName = $state('');

  // Get space ID from route params
  $effect(() => {
    const spaceId = $page.params.id;
    if (spaceId) {
      // Load the specific space and record visit
      spaceStore.loadSpace(spaceId);
      spaceStore.recordVisit(spaceId);
    }
  });

  // Get current space
  const currentSpace = $derived($spaceStore.currentSpace);
  const loading = $derived($spaceStore.isLoading);
  const error = $derived($spaceStore.error);

  function openSpaceSettings() {
    if (currentSpace) {
      newSpaceName = currentSpace.name;
      showSpaceSettings = true;
    }
  }

  function handleCancelSettings() {
    showSpaceSettings = false;
    newSpaceName = '';
  }

  async function handleUpdateSpace() {
    if (currentSpace && newSpaceName.trim()) {
      await spaceStore.updateSpace(currentSpace.id, newSpaceName.trim());
      showSpaceSettings = false;
      newSpaceName = '';
    }
  }

  async function handleDeleteSpace() {
    if (currentSpace && confirm(`Are you sure you want to delete "${currentSpace.name}"? This action cannot be undone.`)) {
      const success = await spaceStore.deleteSpace(currentSpace.id);
      if (success) {
        goto('/');
      }
    }
  }
</script>

<div class="container mx-auto px-4 py-8">
  {#if loading}
    <div class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary600"></div>
    </div>
  {:else if error}
    <div class="bg-destructive/10 border border-destructive/200 text-destructive700 px-4 py-3 rounded-lg dark:bg-destructive/900/20 dark:border-destructive/800 dark:text-destructive400">
      <p class="font-medium">Error loading space</p>
      <p class="text-sm">{error}</p>
    </div>
  {:else if currentSpace}
    <!-- Breadcrumb Navigation -->
    <nav class="flex items-center space-x-2 text-sm text-muted-foreground mb-6">
      <a href="/" class="hover:text-primary">Home</a>
      <span>/</span>
      <span class="text-foreground font-medium">{currentSpace.name}</span>
    </nav>

    <!-- Header -->
    <div class="flex justify-between items-center mb-8">
      <div class="flex items-center gap-4">
        <div>
          <h1 class="text-3xl font-bold text-foreground">{currentSpace.name}</h1>
          <p class="text-muted-foreground mt-1">Manage this space</p>
        </div>
        <Button
          onclick={openSpaceSettings}
          variant="ghost"
          size="icon"
          title="Space settings"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </Button>
      </div>
    </div>

    <div class="text-center py-12">
      <p class="text-muted-foreground">Tasks are now managed through the Flow board.</p>
      <a href="/flow">
        <Button class="mt-4">Go to Flow Board</Button>
      </a>
    </div>
  {:else}
    <div class="text-center py-12">
      <p class="text-muted-foreground">Space not found</p>
      <a href="/">
        <Button class="mt-4">Go Home</Button>
      </a>
    </div>
  {/if}
</div>

<!-- Space Settings Dialog -->
<Dialog.Root bind:open={showSpaceSettings}>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>Space Settings</Dialog.Title>
    </Dialog.Header>

    {#if currentSpace}
      <div class="space-y-6">
        <!-- Rename Space -->
        <div>
          <label class="block text-sm font-medium text-foreground mb-2">
            Space Name
          </label>
          <input
            type="text"
            bind:value={newSpaceName}
            placeholder="Enter space name..."
            class="w-full px-3 py-2 border border-border rounded-lg bg-white dark:bg-secondary/800 text-foreground focus:ring-2 focus:ring-ring focus:border-ring"
          />
        </div>

        <!-- Space Info -->
        <div class="p-4 bg-secondary/50 dark:bg-secondary/800 rounded-lg">
          <p class="text-sm text-muted-foreground">
            <strong>Created:</strong> {new Date(currentSpace.createdAt).toLocaleDateString()}
          </p>
          {#if currentSpace.isDefault}
            <p class="text-sm text-primary mt-1">
              This is your default space
            </p>
          {/if}
        </div>

        <!-- Delete Space -->
        {#if !currentSpace.isDefault}
          <div class="pt-4 border-t border-border">
            <p class="text-sm text-muted-foreground mb-3">
              Deleting this space will remove all tasks associated with it. This action cannot be undone.
            </p>
            <Button variant="destructive" onclick={handleDeleteSpace} class="w-full">
              Delete Space
            </Button>
          </div>
        {:else}
          <div class="pt-4 border-t border-border">
            <p class="text-sm text-muted-foreground italic">
              Default spaces cannot be deleted
            </p>
          </div>
        {/if}
      </div>

      <Dialog.Footer>
        <Button variant="outline" onclick={handleCancelSettings}>
          Cancel
        </Button>
        <Button
          onclick={handleUpdateSpace}
          disabled={!newSpaceName.trim() || newSpaceName === currentSpace.name}
        >
          Save Changes
        </Button>
      </Dialog.Footer>
    {/if}
  </Dialog.Content>
</Dialog.Root>
