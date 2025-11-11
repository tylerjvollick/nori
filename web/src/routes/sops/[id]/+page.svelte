<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { sopStore } from '$lib/stores/sop';
  import type { SOPStep } from '$lib/api/sop';
  import Button from '$lib/components/ui/Button.svelte';
  import SOPHeader from '$lib/components/sop/SOPHeader.svelte';
  import SOPDescription from '$lib/components/sop/SOPDescription.svelte';
  import SOPStepList from '$lib/components/sop/SOPStepList.svelte';

  let showVersionHistory = $state(false);
  let editingTitle = $state(false);
  let editingDescription = $state(false);
  let editingMaterials = $state(false);
  let editingEquipment = $state(false);
  let isPublishing = $state(false);
  let showMoreActionsMenu = $state(false);

  // Local editable state
  let localTitle = $state('');
  let localDescription = $state('');
  let localMaterials = $state<string[]>([]);
  let localEquipment = $state<string[]>([]);
  let localSteps = $state<SOPStep[]>([]);
  let newMaterialInput = $state('');
  let newEquipmentInput = $state('');

  // Derived values using $derived
  let sopId = $derived(parseInt($page.params.id || '0'));
  let isDraftMode = $derived($sopStore.currentSOP?.currentVersion?.status === 'draft');

  // Update local state when SOP changes
  $effect(() => {
    if ($sopStore.currentSOP) {
      localTitle = $sopStore.currentSOP.name;
      localDescription = $sopStore.currentSOP.currentVersion?.description || '';
      localMaterials = $sopStore.currentSOP.currentVersion?.materials || [];
      localEquipment = $sopStore.currentSOP.currentVersion?.equipment || [];
      localSteps = $sopStore.currentSOP.currentVersion?.steps || [];
    }
  });

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

  function getTotalEstimatedTime(steps: SOPStep[]): number {
    return steps.reduce((total, step) => total + (step.estimatedTimeMinutes || 0), 0);
  }


  async function publish() {
    // Only allow publishing if in draft mode
    if (!isDraftMode || !$sopStore.currentSOP?.activeDraftId) {
      return;
    }

    // Prevent double-clicks
    if (isPublishing) {
      return;
    }

    if (confirm('Are you sure you want to publish this draft? This will create a new version of the SOP.')) {
      try {
        isPublishing = true;
        await sopStore.publishDraft($sopStore.currentSOP.activeDraftId, 'Published draft changes');
        // Reload the SOP to get the published version
        await sopStore.loadSOP(sopId);
        isPublishing = false;
      } catch (error) {
        console.error('Failed to publish draft:', error);
        isPublishing = false;
      }
    }
  }

  async function discardChanges() {
    if (!isDraftMode || !$sopStore.currentSOP?.activeDraftId) return;
    
    if (confirm('Are you sure you want to discard this draft? All changes will be lost and you will return to the published version.')) {
      try {
        await sopStore.deleteDraft($sopStore.currentSOP.activeDraftId);
        // Reload the SOP to get the published version
        await sopStore.loadSOP(sopId);
      } catch (error) {
        console.error('Failed to discard draft:', error);
      }
    }
  }

  async function saveTitle() {
    if (!$sopStore.currentSOP) return;
    
    try {
      // Ensure we have a draft
      await sopStore.ensureDraft(sopId);
      
      const draftData = {
        description: localDescription,
        materials: localMaterials,
        equipment: localEquipment,
        changeSummary: 'Updated SOP title',
        steps: localSteps.map(s => ({
          order: s.order,
          title: s.title,
          instructions: s.instructions,
          estimatedTimeMinutes: s.estimatedTimeMinutes,
          imageUrl: s.imageUrl,
          videoUrl: s.videoUrl,
          requiresApproval: s.requiresApproval
        }))
      };

      if ($sopStore.currentSOP.activeDraftId) {
        await sopStore.updateDraft($sopStore.currentSOP.activeDraftId, draftData);
      }
      
      editingTitle = false;
    } catch (error) {
      console.error('Failed to update title:', error);
    }
  }

  async function saveDescription() {
    if (!$sopStore.currentSOP) return;
    
    try {
      // Ensure we have a draft
      await sopStore.ensureDraft(sopId);
      
      const draftData = {
        description: localDescription,
        materials: localMaterials,
        equipment: localEquipment,
        changeSummary: 'Updated SOP description',
        steps: localSteps.map(s => ({
          order: s.order,
          title: s.title,
          instructions: s.instructions,
          estimatedTimeMinutes: s.estimatedTimeMinutes,
          imageUrl: s.imageUrl,
          videoUrl: s.videoUrl,
          requiresApproval: s.requiresApproval
        }))
      };

      if ($sopStore.currentSOP.activeDraftId) {
        await sopStore.updateDraft($sopStore.currentSOP.activeDraftId, draftData);
      }
      
      editingDescription = false;
    } catch (error) {
      console.error('Failed to update description:', error);
    }
  }

  async function saveMaterials() {
    if (!$sopStore.currentSOP) return;
    
    try {
      // Ensure we have a draft
      await sopStore.ensureDraft(sopId);
      
      const draftData = {
        description: localDescription,
        materials: localMaterials,
        equipment: localEquipment,
        changeSummary: 'Updated materials list',
        steps: localSteps.map(s => ({
          order: s.order,
          title: s.title,
          instructions: s.instructions,
          estimatedTimeMinutes: s.estimatedTimeMinutes,
          imageUrl: s.imageUrl,
          videoUrl: s.videoUrl,
          requiresApproval: s.requiresApproval
        }))
      };

      if ($sopStore.currentSOP.activeDraftId) {
        await sopStore.updateDraft($sopStore.currentSOP.activeDraftId, draftData);
      }
      
      editingMaterials = false;
    } catch (error) {
      console.error('Failed to update materials:', error);
    }
  }

  async function saveEquipment() {
    if (!$sopStore.currentSOP) return;
    
    try {
      // Ensure we have a draft
      await sopStore.ensureDraft(sopId);
      
      const draftData = {
        description: localDescription,
        materials: localMaterials,
        equipment: localEquipment,
        changeSummary: 'Updated equipment list',
        steps: localSteps.map(s => ({
          order: s.order,
          title: s.title,
          instructions: s.instructions,
          estimatedTimeMinutes: s.estimatedTimeMinutes,
          imageUrl: s.imageUrl,
          videoUrl: s.videoUrl,
          requiresApproval: s.requiresApproval
        }))
      };

      if ($sopStore.currentSOP.activeDraftId) {
        await sopStore.updateDraft($sopStore.currentSOP.activeDraftId, draftData);
      }
      
      editingEquipment = false;
    } catch (error) {
      console.error('Failed to update equipment:', error);
    }
  }


  function addMaterial() {
    if (newMaterialInput.trim()) {
      localMaterials = [...localMaterials, newMaterialInput.trim()];
      newMaterialInput = '';
    }
  }

  function removeMaterial(index: number) {
    localMaterials = localMaterials.filter((_, i) => i !== index);
  }

  function addEquipment() {
    if (newEquipmentInput.trim()) {
      localEquipment = [...localEquipment, newEquipmentInput.trim()];
      newEquipmentInput = '';
    }
  }

  function removeEquipment(index: number) {
    localEquipment = localEquipment.filter((_, i) => i !== index);
  }

  async function handleDelete() {
    if (confirm('Are you sure you want to delete this SOP? This action cannot be undone.')) {
      try {
        await sopStore.deleteSOP(sopId);
        goto('/sops');
      } catch (error) {
        console.error('Failed to delete SOP:', error);
      }
    }
  }


</script>

<div class="h-full overflow-hidden">
  {#if $sopStore.loading}
    <div class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
    </div>
  {:else if $sopStore.error}
    <div class="bg-destructive/10 border border-destructive/30 text-destructive px-4 py-3 rounded-lg m-4">
      <p class="font-medium">Error loading SOP</p>
      <p class="text-sm">{$sopStore.error}</p>
    </div>
  {:else if $sopStore.currentSOP}
    <!-- Two Column Layout -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6 h-full">
      <!-- Left Column: Main Content -->
      <div class="lg:col-span-2 flex flex-col overflow-hidden">
        <!-- Header with Breadcrumb and Title -->
        <SOPHeader
          bind:title={localTitle}
          editing={editingTitle}
          {isDraftMode}
          versionNumber={$sopStore.currentSOP.currentVersion?.versionNumber}
          lastUpdated={$sopStore.currentSOP.currentVersion?.updatedAt || $sopStore.currentSOP.updatedAt}
          onstartedit={() => editingTitle = true}
          oncanceledit={() => {
            editingTitle = false;
            localTitle = $sopStore.currentSOP?.name || '';
          }}
          onsave={saveTitle}
        />

        <!-- Scrollable Content -->
        <div class="flex-1 overflow-y-auto px-4 py-6 space-y-6">
          <!-- Description -->
          <SOPDescription
            bind:description={localDescription}
            editing={editingDescription}
            onstartedit={() => editingDescription = true}
            oncanceledit={() => {
              editingDescription = false;
              localDescription = $sopStore.currentSOP?.currentVersion?.description || '';
            }}
            onsave={saveDescription}
          />

          <!-- Steps -->
          <SOPStepList 
            steps={localSteps}
            {sopId}
            {isDraftMode}
            onStepsChange={(updatedSteps) => {
              localSteps = updatedSteps;
            }}
            onEnsureDraft={async () => {
              await sopStore.ensureDraft(sopId);
            }}
          />
        </div>
      </div>

      <!-- Right Column: Sidebar -->
      <div class="overflow-y-auto px-4 py-6 space-y-6 border-l border-border">
        <!-- Actions -->
        <div class="pb-6 border-b border-border">
          <div class="flex items-center justify-between gap-2">
            <!-- Left: Publish and Discard Buttons -->
            <div class="flex items-center gap-2">
              <Button
                onclick={publish}
                disabled={!isDraftMode || isPublishing}
                size="sm"
              >
                {isPublishing ? 'Publishing...' : 'Publish'}
              </Button>
              
              {#if isDraftMode}
                <Button
                  onclick={discardChanges}
                  variant="ghost"
                  size="sm"
                >
                  Discard Changes
                </Button>
              {/if}
            </div>
            
            <!-- Right: Three-dot menu -->
            <div class="relative">
              <button
                onclick={() => showMoreActionsMenu = !showMoreActionsMenu}
                class="p-2 rounded-lg hover:bg-accent text-muted-foreground transition-colors"
                aria-label="More actions"
              >
                <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
                </svg>
              </button>
              
              <!-- Dropdown menu -->
              {#if showMoreActionsMenu}
                <div class="absolute right-0 mt-2 w-48 bg-card rounded-lg shadow-lg border border-border py-1 z-10">
                  <button
                    onclick={() => {
                      handleDelete();
                      showMoreActionsMenu = false;
                    }}
                    class="w-full text-left px-4 py-2 text-sm text-destructive hover:bg-accent transition-colors"
                  >
                    Delete SOP
                  </button>
                </div>
              {/if}
            </div>
          </div>
        </div>

        <!-- Metadata -->
        <div class="pb-6 border-b border-border">
          <h3 class="text-sm font-semibold text-foreground mb-3">Details</h3>
          <div class="space-y-3 text-sm">
            <div>
              <span class="text-muted-foreground">Created:</span>
              <span class="text-foreground ml-2">{formatDate($sopStore.currentSOP.createdAt)}</span>
            </div>
            <div>
              <span class="text-muted-foreground">Updated:</span>
              <span class="text-foreground ml-2">{formatDate($sopStore.currentSOP.updatedAt)}</span>
            </div>
            {#if $sopStore.currentSOP.currentVersion}
              <div class="flex items-center justify-between">
                <div>
                  <span class="text-muted-foreground">Version:</span>
                  <span class="text-foreground ml-2">{$sopStore.currentSOP.currentVersion.versionNumber}</span>
                </div>
                <button
                  onclick={loadVersionHistory}
                  class="p-1 rounded hover:bg-accent text-muted-foreground transition-colors"
                  aria-label="Show version history"
                  title="Show version history"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </button>
              </div>
            {/if}
          </div>
        </div>

        <!-- Summary Stats -->
        {#if localSteps.length > 0}
          <div class="pb-6 border-b border-border">
            <h3 class="text-sm font-semibold text-foreground mb-3">Summary</h3>
            <div class="space-y-3">
              <div>
                <div class="text-xs text-muted-foreground mb-1">Total Steps</div>
                <div class="text-xl font-bold text-foreground">
                  {localSteps.length}
                </div>
              </div>
              
              <div>
                <div class="text-xs text-muted-foreground mb-1">Estimated Time</div>
                <div class="text-xl font-bold text-foreground">
                  {getTotalEstimatedTime(localSteps)} min
                </div>
              </div>
              
              <div>
                <div class="text-xs text-muted-foreground mb-1">Approval Required</div>
                <div class="text-xl font-bold text-foreground">
                  {localSteps.filter(s => s.requiresApproval).length}
                </div>
              </div>
            </div>
          </div>
        {/if}

        <!-- Materials -->
        <div class="pb-6 border-b border-border">
          <h3 class="text-sm font-semibold text-foreground mb-3">Materials</h3>
          
          {#if editingMaterials}
            <div class="space-y-3">
              <ul class="space-y-2">
                {#each localMaterials as material, index}
                  <li class="flex items-center justify-between text-sm text-foreground">
                    <span>• {material}</span>
                    <button
                      onclick={() => removeMaterial(index)}
                      class="text-destructive hover:text-destructive/80 text-xs"
                    >
                      Remove
                    </button>
                  </li>
                {/each}
              </ul>
              
              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={newMaterialInput}
                  placeholder="Add material..."
                  class="flex-1 bg-background border border-border rounded px-2 py-1 text-sm"
                  onkeydown={(e) => e.key === 'Enter' && addMaterial()}
                />
                <button
                  onclick={addMaterial}
                  class="bg-primary hover:bg-primary/90 text-white px-2 py-1 rounded text-xs"
                >
                  Add
                </button>
              </div>
              
              <div class="flex gap-2">
                <button
                  onclick={saveMaterials}
                  class="bg-primary hover:bg-primary/90 text-white px-3 py-1 rounded text-sm"
                >
                  Save
                </button>
                <button
                  onclick={() => {
                    editingMaterials = false;
                    localMaterials = $sopStore.currentSOP?.currentVersion?.materials || [];
                  }}
                  class="bg-secondary hover:bg-secondary/90 text-white px-3 py-1 rounded text-sm"
                >
                  Cancel
                </button>
              </div>
            </div>
          {:else}
            <div>
              {#if localMaterials.length > 0}
                <ul class="space-y-1 text-sm text-foreground mb-3">
                  {#each localMaterials as material}
                    <li>• {material}</li>
                  {/each}
                </ul>
              {:else}
                <p class="text-sm text-muted-foreground mb-3">No materials listed</p>
              {/if}
              
              <button
                onclick={() => (editingMaterials = true)}
                class="text-primary hover:text-primary/80 text-sm font-medium"
              >
                Edit Materials
              </button>
            </div>
          {/if}
        </div>

        <!-- Equipment -->
        <div>
          <h3 class="text-sm font-semibold text-foreground mb-3">Equipment</h3>
          
          {#if editingEquipment}
            <div class="space-y-3">
              <ul class="space-y-2">
                {#each localEquipment as equipment, index}
                  <li class="flex items-center justify-between text-sm text-foreground">
                    <span>• {equipment}</span>
                    <button
                      onclick={() => removeEquipment(index)}
                      class="text-destructive hover:text-destructive/80 text-xs"
                    >
                      Remove
                    </button>
                  </li>
                {/each}
              </ul>
              
              <div class="flex gap-2">
                <input
                  type="text"
                  bind:value={newEquipmentInput}
                  placeholder="Add equipment..."
                  class="flex-1 bg-background border border-border rounded px-2 py-1 text-sm"
                  onkeydown={(e) => e.key === 'Enter' && addEquipment()}
                />
                <button
                  onclick={addEquipment}
                  class="bg-primary hover:bg-primary/90 text-white px-2 py-1 rounded text-xs"
                >
                  Add
                </button>
              </div>
              
              <div class="flex gap-2">
                <button
                  onclick={saveEquipment}
                  class="bg-primary hover:bg-primary/90 text-white px-3 py-1 rounded text-sm"
                >
                  Save
                </button>
                <button
                  onclick={() => {
                    editingEquipment = false;
                    localEquipment = $sopStore.currentSOP?.currentVersion?.equipment || [];
                  }}
                  class="bg-secondary hover:bg-secondary/90 text-white px-3 py-1 rounded text-sm"
                >
                  Cancel
                </button>
              </div>
            </div>
          {:else}
            <div>
              {#if localEquipment.length > 0}
                <ul class="space-y-1 text-sm text-foreground mb-3">
                  {#each localEquipment as equipment}
                    <li>• {equipment}</li>
                  {/each}
                </ul>
              {:else}
                <p class="text-sm text-muted-foreground mb-3">No equipment listed</p>
              {/if}
              
              <button
                onclick={() => (editingEquipment = true)}
                class="text-primary hover:text-primary/80 text-sm font-medium"
              >
                Edit Equipment
              </button>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}
</div>

<!-- Version History Modal/Overlay (Outside main grid) -->
{#if $sopStore.currentSOP && showVersionHistory}
  <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
    <div class="bg-card rounded-lg shadow-xl border border-border max-w-4xl w-full max-h-[80vh] overflow-y-auto">
      <div class="sticky top-0 bg-card border-b border-border p-6 flex items-center justify-between">
        <h2 class="text-xl font-semibold text-foreground">Version History</h2>
        <button
          onclick={() => showVersionHistory = false}
          class="text-muted-foreground hover:text-foreground"
        >
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      
      <div class="p-6">
        
        {#if $sopStore.loading}
          <div class="flex justify-center py-6">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        {:else if $sopStore.currentVersions.length === 0}
          <p class="text-muted-foreground">No version history available</p>
        {:else}
          <div class="space-y-4">
            {#each $sopStore.currentVersions as version}
              <div class="border border-border rounded-lg p-4 {version.isActive ? 'bg-primary/5 border-primary/30' : ''}">
                <div class="flex justify-between items-start mb-2">
                  <div class="flex items-center gap-3">
                    <span class="text-lg font-bold text-foreground">
                      Version {version.versionNumber}
                    </span>
                    {#if version.isActive}
                      <span class="bg-primary text-primary-foreground px-2 py-1 rounded text-xs font-medium">
                        Current
                      </span>
                    {/if}
                  </div>
                  <span class="text-sm text-muted-foreground">
                    {formatDate(version.createdAt)}
                  </span>
                </div>
                
                {#if version.changeSummary}
                  <p class="text-sm text-foreground mb-2">
                    <span class="font-medium">Changes:</span> {version.changeSummary}
                  </p>
                {/if}
                
                {#if version.description}
                  <p class="text-sm text-muted-foreground">
                    {version.description}
                  </p>
                {/if}
                
                {#if version.steps}
                  <p class="text-xs text-muted-foreground mt-2">
                    {version.steps.length} steps
                  </p>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}


