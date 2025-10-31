<script lang="ts">
  import { goto } from '$app/navigation';
  import { sopStore } from '$lib/stores/sop';
  import SOPForm from '$lib/components/SOPForm.svelte';

  let isSubmitting = $state(false);

  async function handleSubmit(data: {
    name: string;
    description?: string;
    materials?: string[];
    equipment?: string[];
    steps: any[];
  }) {
    isSubmitting = true;

    try {
      await sopStore.createSOP({
        ...data,
        changeSummary: 'Initial version'
      });

      goto('/sops');
    } catch (err) {
      console.error('Failed to create SOP:', err);
      isSubmitting = false;
    }
  }

  function handleCancel() {
    goto('/sops');
  }
</script>

<div class="container mx-auto px-4 py-8 max-w-4xl">
  <div class="mb-8">
    <button
      onclick={() => goto('/sops')}
      class="text-emerald-600 hover:text-emerald-700 mb-4 inline-flex items-center"
    >
      ← Back to SOPs
    </button>
    <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Create New SOP</h1>
  </div>

  <SOPForm 
    onSubmit={handleSubmit}
    onCancel={handleCancel}
    isSubmitting={isSubmitting}
    submitButtonText="Create SOP"
    showCancelButton={true}
  />
</div>
