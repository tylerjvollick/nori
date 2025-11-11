<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { sopStore } from '$lib/stores/sop';
  import SOPForm from '$lib/components/SOPForm.svelte';
  import type { SOPStep } from '$lib/api/sop';
  import { Button } from '$lib/components/ui/button';

  let originalName = '';
  let isSubmitting = $state(false);
  let isLoading = $state(true);
  let initialData = $state<{
    name?: string;
    description?: string;
    materials?: string[];
    equipment?: string[];
    steps?: Omit<SOPStep, 'id'>[];
  } | undefined>(undefined);

  let sopId = $derived(parseInt($page.params.id || '0'));

  onMount(async () => {
    try {
      const sop = await sopStore.loadSOP(sopId);
      if (sop && sop.currentVersion) {
        originalName = sop.name;
        
        // Prepare initial data for the form
        const steps = sop.currentVersion.steps && sop.currentVersion.steps.length > 0
          ? sop.currentVersion.steps.map(step => ({
              order: step.order,
              title: step.title,
              instructions: step.instructions || '',
              estimatedTimeMinutes: step.estimatedTimeMinutes,
              imageUrl: step.imageUrl || '',
              videoUrl: step.videoUrl || '',
              requiresApproval: step.requiresApproval
            }))
          : undefined;

        initialData = {
          name: sop.name,
          description: sop.currentVersion.description || '',
          materials: sop.currentVersion.materials || [],
          equipment: sop.currentVersion.equipment || [],
          steps
        };
      }
      isLoading = false;
    } catch (err) {
      console.error('Failed to load SOP:', err);
      isLoading = false;
    }
  });

  async function handleSubmit(data: {
    name: string;
    description?: string;
    materials?: string[];
    equipment?: string[];
    steps: Omit<SOPStep, 'id'>[];
    changeSummary?: string;
  }) {
    isSubmitting = true;

    try {
      await sopStore.updateSOP(sopId, {
        name: data.name !== originalName ? data.name : undefined,
        description: data.description,
        materials: data.materials,
        equipment: data.equipment,
        changeSummary: data.changeSummary!,
        steps: data.steps
      });

      goto(`/sops/${sopId}`);
    } catch (err) {
      console.error('Failed to update SOP:', err);
      isSubmitting = false;
    }
  }

  function handleCancel() {
    goto(`/sops/${sopId}`);
  }
</script>

<div class="container mx-auto px-4 py-8 max-w-4xl">
  {#if isLoading}
    <div class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary600"></div>
    </div>
  {:else}
    <div class="mb-8">
      <Button
        onclick={() => goto(`/sops/${sopId}`)}
        variant="link"
        class="mb-4 h-auto p-0"
      >
        ← Back to SOP
      </Button>
      <h1 class="text-3xl font-bold text-foreground">Edit SOP</h1>
      <p class="text-sm text-muted-foreground mt-2">
        Updating this SOP will create a new version
      </p>
    </div>

    <SOPForm 
      onSubmit={handleSubmit}
      onCancel={handleCancel}
      isSubmitting={isSubmitting}
      submitButtonText="Save Changes (Create New Version)"
      showCancelButton={true}
      isEditMode={true}
      initialData={initialData}
    />
  {/if}
</div>
