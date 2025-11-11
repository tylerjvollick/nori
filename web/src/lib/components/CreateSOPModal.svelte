<script lang="ts">
  import { sopStore } from '$lib/stores/sop';
  import { goto } from '$app/navigation';
  import SOPForm from './SOPForm.svelte';
  import * as Dialog from '$lib/components/ui/dialog';

  interface CreateSOPModalProps {
    isOpen: boolean;
    onClose: () => void;
  }

  let { isOpen, onClose }: CreateSOPModalProps = $props();

  let isSubmitting = $state(false);
  let error = $state('');
  let formRef: any = $state(null);

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

  function triggerSubmit() {
    // Trigger form submission by finding the form element and calling submit
    const form = document.querySelector('form');
    if (form) {
      form.requestSubmit();
    }
  }

  function handleOpenChange(open: boolean) {
    if (!open && !isSubmitting) {
      handleCancel();
    }
  }
</script>

<Dialog.Root open={isOpen} onOpenChange={handleOpenChange}>
  <Dialog.Content class="max-w-4xl max-h-[90vh] flex flex-col p-0 gap-0">
    <!-- Header -->
    <Dialog.Header class="px-6 py-4 border-b border-border bg-accent flex-shrink-0 sm:text-left">
      <Dialog.Title class="text-2xl font-bold text-foreground">
        Create New SOP
      </Dialog.Title>
      <Dialog.Description class="text-sm text-muted-foreground mt-1">
        Define a standard operating procedure template
      </Dialog.Description>
    </Dialog.Header>

    <!-- Scrollable Content -->
    <div class="flex-1 overflow-y-auto p-6 min-h-0">
      {#if error}
        <div class="bg-destructive/10 border border-destructive/20 text-destructive px-4 py-3 rounded-lg mb-4">
          <p class="font-medium">Error</p>
          <p class="text-sm">{error}</p>
        </div>
      {/if}

      <SOPForm 
        bind:this={formRef}
        onSubmit={handleSubmit}
        onCancel={handleCancel}
        isSubmitting={isSubmitting}
        submitButtonText="Create SOP"
        showCancelButton={false}
      />
    </div>

    <!-- Sticky Footer -->
    <Dialog.Footer class="border-t border-border px-6 py-4 bg-card flex-shrink-0 sm:justify-start">
      <div class="flex gap-4 w-full">
        <button
          type="button"
          onclick={handleCancel}
          class="flex-1 bg-secondary hover:bg-secondary/80 text-secondary-foreground px-6 py-3 rounded-lg font-medium transition-colors"
          disabled={isSubmitting}
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={triggerSubmit}
          class="flex-1 bg-primary hover:bg-primary/90 text-primary-foreground px-6 py-3 rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          disabled={isSubmitting}
        >
          {isSubmitting ? 'Creating...' : 'Create SOP'}
        </button>
      </div>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
