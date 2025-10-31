<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { sopStore } from '$lib/stores/sop';
  
  let confirmDeleteId: number | null = null;

  onMount(() => {
    sopStore.loadAllSOPs();
  });

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  }

  async function handleDelete(id: number) {
    try {
      await sopStore.deleteSOP(id);
      confirmDeleteId = null;
    } catch (error) {
      console.error('Failed to delete SOP:', error);
    }
  }
</script>

<div class="container mx-auto px-4 py-8">
  <div class="flex justify-between items-center mb-8">
    <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Standard Operating Procedures</h1>
    <button
      on:click={() => goto('/sops/create')}
      class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition-colors"
    >
      Create New SOP
    </button>
  </div>

  {#if $sopStore.loading}
    <div class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
    </div>
  {:else if $sopStore.error}
    <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
      <p class="font-medium">Error loading SOPs</p>
      <p class="text-sm">{$sopStore.error}</p>
    </div>
  {:else if !$sopStore.sops || $sopStore.sops.length === 0}
    <div class="text-center py-12">
      <p class="text-gray-500 dark:text-gray-400 mb-4">No SOPs created yet</p>
      <button
        on:click={() => goto('/sops/create')}
        class="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-lg transition-colors"
      >
        Create Your First SOP
      </button>
    </div>
  {:else}
    <div class="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
      {#each $sopStore.sops as sop}
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 hover:shadow-lg transition-shadow">
          <div class="p-6">
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
              {sop.name}
            </h2>
            
            {#if sop.currentVersion}
              <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Version {sop.currentVersion.versionNumber} • {sop.currentVersion.steps?.length || 0} steps
              </p>
              
              {#if sop.currentVersion.description}
                <p class="text-sm text-gray-700 dark:text-gray-300 mb-4 line-clamp-2">
                  {sop.currentVersion.description}
                </p>
              {/if}
            {/if}

            <div class="text-xs text-gray-500 dark:text-gray-500 mb-4">
              Last updated: {formatDate(sop.updatedAt)}
            </div>

            <div class="flex gap-2">
              <button
                on:click={() => goto(`/sops/${sop.id}`)}
                class="flex-1 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm transition-colors"
              >
                View
              </button>
              <button
                on:click={() => goto(`/sops/${sop.id}/edit`)}
                class="flex-1 bg-gray-600 hover:bg-gray-700 text-white px-4 py-2 rounded-lg text-sm transition-colors"
              >
                Edit
              </button>
              <button
                on:click={() => confirmDeleteId = sop.id}
                class="bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-lg text-sm transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Delete Confirmation Modal -->
{#if confirmDeleteId !== null}
  <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
    <div class="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
        Confirm Delete
      </h3>
      <p class="text-gray-700 dark:text-gray-300 mb-6">
        Are you sure you want to delete this SOP? This action cannot be undone and will delete all versions.
      </p>
      <div class="flex gap-4">
        <button
          on:click={() => confirmDeleteId = null}
          class="flex-1 bg-gray-200 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-900 dark:text-white px-4 py-2 rounded-lg transition-colors"
        >
          Cancel
        </button>
        <button
          on:click={() => confirmDeleteId && handleDelete(confirmDeleteId)}
          class="flex-1 bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded-lg transition-colors"
        >
          Delete
        </button>
      </div>
    </div>
  </div>
{/if}
