<script lang="ts">
  import { sopStore } from '$lib/stores/sop';
  import { goto } from '$app/navigation';
  import SOPForm from './SOPForm.svelte';
  import { X } from 'lucide-svelte';

  interface CreateSOPModalProps {
    isOpen: boolean;
    onClose: () => void;
  }

  let { isOpen, onClose }: CreateSOPModalProps = $props();

  let isSubmitting = $state(false);
  let error = $state('');

  async function handleSubmit(data: Parameters<typeof sopStore.createSOP>[0]) {
    isSubmitting = true;
    error = '';

    try {
      await sopStore.createSOP(data);
      // Reset and close
      isSubmitting = false;
      onClose();
      // Navigate to the SOP list or detail page
      goto('/sops');
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to create SOP';
      isSubmitting = false;
    }
  }

  function handleCancel() {
    error = '';
    onClose();
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !isSubmitting) {
      handleCancel();
    }
  }
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if isOpen}
  <div 
    class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4"
    role="dialog"
    aria-modal="true"
    onclick={handleCancel}
  >
    <div 
      class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-5xl w-full max-h-[90vh] overflow-hidden flex flex-col"
      onclick={(e) => e.stopPropagation()}
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700 bg-gradient-to-r from-emerald-50 to-teal-50 dark:from-emerald-900/20 dark:to-teal-900/20">
        <div>
          <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Create New SOP</h2>
          <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
            Define a standard operating procedure template
          </p>
        </div>
        <button
          onclick={handleCancel}
          disabled={isSubmitting}
          class="p-2 rounded-lg text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-white/50 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label="Close"
        >
          <X class="w-6 h-6" />
        </button>
      </div>

      <!-- Content -->
      <div class="flex-1 overflow-y-auto p-6">
        {#if error}
          <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-4">
            <p class="font-medium">Error</p>
            <p class="text-sm">{error}</p>
          </div>
        {/if}

        <SOPForm 
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          isSubmitting={isSubmitting}
          submitButtonText="Create SOP"
          showCancelButton={true}
        />
      </div>
    </div>
  </div>
{/if}
