<script lang="ts">
  import { dndzone } from 'svelte-dnd-action';
  import { sopApi } from '$lib/api/sop';
  import type { SOPStepMedia } from '$lib/api/sop';
  import { Button } from '$lib/components/ui/button';
  import * as Carousel from '$lib/components/ui/carousel';
  import * as Dialog from '$lib/components/ui/dialog';
  import { Image, Trash2, Upload, X, Video } from 'lucide-svelte';

  interface Props {
    sopId: number;
    stepId: number;
    photos?: SOPStepMedia[];
    onPhotosChange?: (photos: SOPStepMedia[]) => void;
  }

  let { sopId, stepId, photos = [], onPhotosChange }: Props = $props();

  // Local state
  let localPhotos = $state<SOPStepMedia[]>([...photos]);
  let uploading = $state(false);
  let isDragging = $state(false);
  let fileInputRef = $state<HTMLInputElement>();
  let selectedPhotoIndex = $state<number | null>(null);
  let deletingPhotoId = $state<number | null>(null);
  let fullscreenDialogOpen = $state(false);
  let fullscreenCarouselApi = $state<any>();

  // Update local state when props change
  $effect(() => {
    localPhotos = [...photos];
  });

  // Notify parent of changes
  function notifyChange() {
    if (onPhotosChange) {
      onPhotosChange(localPhotos);
    }
  }

  // Check if media is a video
  function isVideo(mimeType: string): boolean {
    return mimeType.startsWith('video/');
  }

  function openFileDialog() {
    fileInputRef?.click();
  }

  async function handleFileSelect(e: Event) {
    const target = e.target as HTMLInputElement;
    const files = target.files;
    
    if (!files || files.length === 0) return;

    const file = files[0];
    
    // Validate file type
    if (!file.type.startsWith('image/') && !file.type.startsWith('video/')) {
      alert('Please select an image or video file');
      return;
    }

    // Validate file size (1GB max)
    const maxSize = 1024 * 1024 * 1024; // 1GB
    if (file.size > maxSize) {
      alert('File size must be less than 1GB');
      return;
    }

    try {
      uploading = true;
      const newMedia = await sopApi.uploadStepMedia(sopId, stepId, file);
      
      // Add to local state
      localPhotos = [...localPhotos, newMedia];
      notifyChange();
      
      // Reset file input
      if (target) target.value = '';
    } catch (error) {
      console.error('Failed to upload media:', error);
      alert('Failed to upload media. Please try again.');
    } finally {
      uploading = false;
    }
  }

  async function deletePhoto(photoId: number) {
    if (!confirm('Are you sure you want to delete this media?')) {
      return;
    }

    try {
      deletingPhotoId = photoId;
      await sopApi.deleteStepMedia(photoId);
      
      // Remove from local state
      localPhotos = localPhotos.filter(p => p.id !== photoId);
      notifyChange();
    } catch (error) {
      console.error('Failed to delete media:', error);
      alert('Failed to delete media. Please try again.');
    } finally {
      deletingPhotoId = null;
    }
  }

  function openLightbox(index: number) {
    selectedPhotoIndex = index;
    fullscreenDialogOpen = true;
    // Wait for dialog to open, then scroll to the selected photo
    setTimeout(() => {
      fullscreenCarouselApi?.scrollTo(index, true);
    }, 100);
  }

  function closeLightbox() {
    fullscreenDialogOpen = false;
    selectedPhotoIndex = null;
  }

  // DnD event handlers
  function handleDndConsider(e: CustomEvent) {
    localPhotos = e.detail.items;
    if (!isDragging) {
      isDragging = true;
    }
  }

  async function handleDndFinalize(e: CustomEvent) {
    const newLocalPhotos = e.detail.items;
    const info = e.detail.info;
    
    if (!info) {
      localPhotos = newLocalPhotos;
      isDragging = false;
      return;
    }
    
    const movedItemId = info.id;
    const newIndex = newLocalPhotos.findIndex((item: any) => item.id === movedItemId);
    
    if (newIndex === -1) {
      localPhotos = newLocalPhotos;
      isDragging = false;
      return;
    }
    
    isDragging = false;
    await handleReorder(movedItemId, newIndex, newLocalPhotos);
  }

  async function handleReorder(photoId: number, newIndex: number, newLocalPhotos: SOPStepMedia[]) {
    try {
      let beforeMediaId: number | undefined;
      let afterMediaId: number | undefined;
      
      if (newIndex > 0) {
        beforeMediaId = newLocalPhotos[newIndex - 1].id;
      }
      
      if (newIndex < newLocalPhotos.length - 1) {
        afterMediaId = newLocalPhotos[newIndex + 1].id;
      }
      
      await sopApi.reorderStepMedia(photoId, {
        beforeMediaId,
        afterMediaId
      });
      
      localPhotos = newLocalPhotos;
      notifyChange();
    } catch (error) {
      console.error('Failed to reorder media:', error);
      localPhotos = [...photos];
      notifyChange();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && fullscreenDialogOpen) {
      closeLightbox();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="space-y-3">
  <!-- Photo Carousel -->
  {#if localPhotos && localPhotos.length > 0}
    <Carousel.Root class="w-full">
      <Carousel.Content>
        {#each localPhotos as photo, index (photo.id)}
          <Carousel.Item class="md:basis-1/2 lg:basis-1/3">
            <div class="relative group aspect-square rounded-lg overflow-hidden border border-border bg-muted">
              <!-- Media (Photo or Video) -->
              <button
                onclick={() => openLightbox(index)}
                class="w-full h-full"
                type="button"
                aria-label={isVideo(photo.mimeType) ? "View video" : "View photo"}
              >
                {#if isVideo(photo.mimeType)}
                  <video
                    src={sopApi.getMediaUrl(photo.uuid)}
                    class="w-full h-full object-cover"
                    muted
                    loop
                    preload="metadata"
                  >
                    <track kind="captions" />
                  </video>
                  <!-- Video overlay indicator -->
                  <div class="absolute inset-0 flex items-center justify-center pointer-events-none">
                    <div class="bg-black/50 rounded-full p-3">
                      <Video class="w-8 h-8 text-white" />
                    </div>
                  </div>
                {:else}
                  <img
                    src={sopApi.getMediaUrl(photo.uuid)}
                    alt={photo.fileName}
                    class="w-full h-full object-cover transition-transform group-hover:scale-105"
                  />
                {/if}
              </button>

              <!-- Overlay on hover -->
              <div class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
                <!-- View button -->
                <button
                  onclick={() => openLightbox(index)}
                  type="button"
                  class="p-2 bg-white/90 rounded-full hover:bg-white transition-colors"
                  aria-label="View photo"
                >
                  <Image class="w-4 h-4 text-gray-900" />
                </button>

                <!-- Delete button -->
                <button
                  onclick={() => deletePhoto(photo.id)}
                  disabled={deletingPhotoId === photo.id}
                  type="button"
                  class="p-2 bg-red-500/90 rounded-full hover:bg-red-600 transition-colors disabled:opacity-50"
                  aria-label="Delete photo"
                >
                  <Trash2 class="w-4 h-4 text-white" />
                </button>
              </div>
            </div>
          </Carousel.Item>
        {/each}
      </Carousel.Content>
      <Carousel.Previous class="left-2" />
      <Carousel.Next class="right-2" />
    </Carousel.Root>
  {:else}
    <div class="text-center py-8 border-2 border-dashed border-border rounded-lg">
      <Image class="w-12 h-12 mx-auto text-muted-foreground mb-2" />
      <p class="text-sm text-muted-foreground">No media yet. Upload your first photo or video.</p>
    </div>
  {/if}

  <!-- Upload Button (moved to bottom) -->
  <div>
    <input
      bind:this={fileInputRef}
      type="file"
      accept="image/*,video/*"
      onchange={handleFileSelect}
      class="hidden"
      aria-label="Upload photo or video"
    />
    <Button
      onclick={openFileDialog}
      disabled={uploading}
      size="sm"
      variant="outline"
      class="w-full sm:w-auto"
    >
      {#if uploading}
        <svg class="animate-spin h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        Uploading...
      {:else}
        <Upload class="w-4 h-4 mr-2" />
        Upload Media
      {/if}
    </Button>
  </div>
</div>

<!-- Fullscreen Dialog with Single-Photo Carousel -->
<Dialog.Root bind:open={fullscreenDialogOpen}>
  <Dialog.Content class="sm:max-w-[90vw]  max-h-[90vh] max-w-[95vw] max-h-[95vh] p-0 border-0 bg-black">
    <div class="relative w-full h-full">
      <!-- Close button -->
      <button
        onclick={closeLightbox}
        type="button"
        class="absolute top-4 right-4 z-50 p-2 bg-black/50 hover:bg-black/70 rounded-full text-white transition-colors"
        aria-label="Close"
      >
        <X class="w-6 h-6" />
      </button>

      <!-- Single Photo Carousel -->
      {#if localPhotos.length > 0}
        <Carousel.Root class="w-full h-[90vh]" setApi={(api) => { fullscreenCarouselApi = api; }}>
          <Carousel.Content class="h-full">
            {#each localPhotos as photo (photo.id)}
              <Carousel.Item class="h-full">
                <div class="flex items-center justify-center h-full p-4">
                  {#if isVideo(photo.mimeType)}
                    <video
                      src={sopApi.getMediaUrl(photo.uuid)}
                      controls
                      class="max-w-full max-h-full object-contain"
                    >
                      <track kind="captions" />
                    </video>
                  {:else}
                    <img
                      src={sopApi.getMediaUrl(photo.uuid)}
                      alt={photo.fileName}
                      class="max-w-full max-h-full object-contain"
                    />
                  {/if}
                </div>
              </Carousel.Item>
            {/each}
          </Carousel.Content>
          <Carousel.Previous class="left-4 bg-black/50 hover:bg-black/70 text-white" />
          <Carousel.Next class="right-4 bg-black/50 hover:bg-black/70 text-white" />
        </Carousel.Root>

        <!-- Photo Info -->
        {#if selectedPhotoIndex !== null && localPhotos[selectedPhotoIndex]}
          <div class="absolute bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black/80 to-transparent">
            <h3 class="font-medium text-white">{localPhotos[selectedPhotoIndex].fileName}</h3>
            <p class="text-sm text-white/70 mt-1">
              {(localPhotos[selectedPhotoIndex].fileSize / 1024).toFixed(1)} KB • 
              {new Date(localPhotos[selectedPhotoIndex].createdAt).toLocaleDateString()}
            </p>
          </div>
        {/if}
      {/if}
    </div>
  </Dialog.Content>
</Dialog.Root>

<style>
  /* Smooth transitions for drag and drop */
  :global(.photo-grid-item) {
    transition: transform 200ms ease;
  }
</style>
