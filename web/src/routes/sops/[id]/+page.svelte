<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { sopStore } from '$lib/stores/sop';

  let sopId: number;
  let showVersionHistory = false;

  $: sopId = parseInt($page.params.id || '0');

  onMount(async () => {
    await sopStore.loadSOP(sopId);
  });

  async function loadVersionHistory() {
    if (!showVersionHistory) {
      await sopStore.loadSOPVersions(sopId);
    }
    showVersionHistory = !showVersionHistory;
  }

  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function getTotalEstimatedTime(steps: any[]): number {
    return steps.reduce((total, step) => total + (step.estimatedTimeMinutes || 0), 0);
  }
</script>

<div class="container mx-auto px-4 py-8 max-w-6xl">
  {#if $sopStore.loading}
    <div class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
    </div>
  {:else if $sopStore.error}
    <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg">
      <p class="font-medium">Error loading SOP</p>
      <p class="text-sm">{$sopStore.error}</p>
    </div>
  {:else if $sopStore.currentSOP}
    <div class="mb-8">
      <button
        on:click={() => goto('/sops')}
        class="text-blue-600 hover:text-blue-700 mb-4 inline-flex items-center"
      >
        ← Back to SOPs
      </button>
      
      <div class="flex justify-between items-start">
        <div>
          <h1 class="text-3xl font-bold text-gray-900 dark:text-white mb-2">
            {$sopStore.currentSOP.name}
          </h1>
          {#if $sopStore.currentSOP.currentVersion}
            <p class="text-sm text-gray-600 dark:text-gray-400">
              Version {$sopStore.currentSOP.currentVersion.versionNumber} • 
              Last updated: {formatDate($sopStore.currentSOP.updatedAt)}
            </p>
          {/if}
        </div>
        
        <div class="flex gap-2">
          <button
            on:click={() => goto(`/sops/${sopId}/edit`)}
            class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition-colors"
          >
            Edit SOP
          </button>
          <button
            on:click={loadVersionHistory}
            class="bg-gray-600 hover:bg-gray-700 text-white px-4 py-2 rounded-lg transition-colors"
          >
            {showVersionHistory ? 'Hide' : 'Show'} Version History
          </button>
        </div>
      </div>
    </div>

    {#if $sopStore.currentSOP.currentVersion}
      <!-- Description -->
      {#if $sopStore.currentSOP.currentVersion.description}
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-6 mb-6">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-3">Description</h2>
          <p class="text-gray-700 dark:text-gray-300">{$sopStore.currentSOP.currentVersion.description}</p>
        </div>
      {/if}

      <!-- Summary Stats -->
      {#if $sopStore.currentSOP.currentVersion.steps}
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
          <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-4">
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Total Steps</div>
            <div class="text-2xl font-bold text-gray-900 dark:text-white">
              {$sopStore.currentSOP.currentVersion.steps.length}
            </div>
          </div>
          
          <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-4">
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Estimated Time</div>
            <div class="text-2xl font-bold text-gray-900 dark:text-white">
              {getTotalEstimatedTime($sopStore.currentSOP.currentVersion.steps)} min
            </div>
          </div>
          
          <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-4">
            <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">Approval Required</div>
            <div class="text-2xl font-bold text-gray-900 dark:text-white">
              {$sopStore.currentSOP.currentVersion.steps.filter(s => s.requiresApproval).length}
            </div>
          </div>
        </div>
      {/if}

      <!-- Steps -->
      {#if $sopStore.currentSOP.currentVersion.steps && $sopStore.currentSOP.currentVersion.steps.length > 0}
        <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-6 mb-6">
          <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-6">Steps</h2>
          
          <div class="space-y-6">
            {#each $sopStore.currentSOP.currentVersion.steps as step, index}
              <div class="border-l-4 border-blue-500 pl-6 pr-4 py-4 bg-gray-50 dark:bg-gray-900 rounded-r-lg">
                <div class="flex justify-between items-start mb-3">
                  <div>
                    <div class="flex items-center gap-3">
                      <span class="inline-flex items-center justify-center w-8 h-8 bg-blue-600 text-white rounded-full font-bold text-sm">
                        {step.stepNumber}
                      </span>
                      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                        {step.title}
                      </h3>
                    </div>
                  </div>
                  
                  <div class="flex gap-2 text-xs">
                    {#if step.estimatedTimeMinutes}
                      <span class="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200 px-2 py-1 rounded">
                        {step.estimatedTimeMinutes} min
                      </span>
                    {/if}
                    {#if step.requiresApproval}
                      <span class="bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200 px-2 py-1 rounded">
                        Approval Required
                      </span>
                    {/if}
                  </div>
                </div>

                {#if step.instructions}
                  <p class="text-gray-700 dark:text-gray-300 mb-4 whitespace-pre-wrap">
                    {step.instructions}
                  </p>
                {/if}

                {#if step.imageUrl || step.videoUrl}
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {#if step.imageUrl}
                      <div>
                        <img 
                          src={step.imageUrl} 
                          alt="Step {step.stepNumber} illustration"
                          class="rounded-lg border border-gray-300 dark:border-gray-600 max-w-full h-auto"
                        />
                      </div>
                    {/if}
                    {#if step.videoUrl}
                      <div>
                        <video 
                          src={step.videoUrl} 
                          controls
                          class="rounded-lg border border-gray-300 dark:border-gray-600 max-w-full h-auto"
                        >
                          <track kind="captions" />
                        </video>
                      </div>
                    {/if}
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        </div>
      {/if}
    {/if}

    <!-- Version History -->
    {#if showVersionHistory}
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-6">
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-6">Version History</h2>
        
        {#if $sopStore.loading}
          <div class="flex justify-center py-6">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        {:else if $sopStore.currentVersions.length === 0}
          <p class="text-gray-600 dark:text-gray-400">No version history available</p>
        {:else}
          <div class="space-y-4">
            {#each $sopStore.currentVersions as version}
              <div class="border border-gray-200 dark:border-gray-600 rounded-lg p-4 {version.isActive ? 'bg-blue-50 dark:bg-blue-900/20 border-blue-300 dark:border-blue-700' : ''}">
                <div class="flex justify-between items-start mb-2">
                  <div class="flex items-center gap-3">
                    <span class="text-lg font-bold text-gray-900 dark:text-white">
                      Version {version.versionNumber}
                    </span>
                    {#if version.isActive}
                      <span class="bg-blue-600 text-white px-2 py-1 rounded text-xs font-medium">
                        Current
                      </span>
                    {/if}
                  </div>
                  <span class="text-sm text-gray-600 dark:text-gray-400">
                    {formatDate(version.createdAt)}
                  </span>
                </div>
                
                {#if version.changeSummary}
                  <p class="text-sm text-gray-700 dark:text-gray-300 mb-2">
                    <span class="font-medium">Changes:</span> {version.changeSummary}
                  </p>
                {/if}
                
                {#if version.description}
                  <p class="text-sm text-gray-600 dark:text-gray-400">
                    {version.description}
                  </p>
                {/if}
                
                {#if version.steps}
                  <p class="text-xs text-gray-500 dark:text-gray-500 mt-2">
                    {version.steps.length} steps
                  </p>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {/if}
</div>
