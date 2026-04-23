<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { sopApi } from '$lib/api/sop';
  import type { SOPStepMedia } from '$lib/api/sop';
  import { Button } from '$lib/components/ui/button';
  import * as Carousel from '$lib/components/ui/carousel';
  import * as Dialog from '$lib/components/ui/dialog';
  import { Image, Trash2, Upload, X, Video } from '@lucide/svelte';
  import Uppy from '@uppy/core';
  import Tus from '@uppy/tus';

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
  let uploadProgress = $state(0);
  let fileInputRef = $state<HTMLInputElement>();
  let selectedPhotoIndex = $state<number | null>(null);
  let deletingPhotoId = $state<number | null>(null);
  let fullscreenDialogOpen = $state(false);
  let fullscreenCarouselApi = $state<any>();
  let uppy: Uppy | null = null;

  // Update local state when props change
  $effect(() => {
    localPhotos = [...photos];
  });

  onMount(() => {
    // Initialize Uppy with tus plugin
    uppy = new Uppy({
      restrictions: {
        maxFileSize: 1024 * 1024 * 1024, // 1GB
        allowedFileTypes: ['image/*', 'video/*']
      },
      autoProceed: true
    });

    // Configure tus plugin
    uppy.use(Tus, {
      endpoint: `${import.meta.env.VITE_API_URL || 'http://localhost:8081'}/api/tus/`,
      chunkSize: 5 * 1024 * 1024, // 5MB chunks
      retryDelays: [0, 1000, 3000, 5000],
      removeFingerprintOnSuccess: true,
      withCredentials: false, // Don't send cookies (not needed for TUS uploads)
      headers: {
        // CORS headers are handled by browser
      }
    });

    // Track upload progress
    uppy.on('upload-progress', (file, progress) => {
      if (file && progress.bytesTotal) {
        uploadProgress = Math.round((progress.bytesUploaded / progress.bytesTotal) * 100);
      }
    });

    // Handle upload start
    uppy.on('upload', () => {
      uploading = true;
      uploadProgress = 0;
    });

    // Handle upload success
    uppy.on('upload-success', async (file, response) => {
      console.log('Upload completed:', file?.name);
    });

    // Handle upload error
    uppy.on('upload-error', (file, error) => {
      console.error('Upload failed:', error);
      alert(`Upload failed: ${error.message}`);
      uploading = false;
    });

    // Handle all uploads complete
    uppy.on('complete', async (result) => {
      // Refresh media list after all uploads complete
      await refreshMediaList();
      uploading = false;
      uploadProgress = 0;
      if (fileInputRef) fileInputRef.value = '';
    });
  });

  onDestroy(() => {
    if (uppy) {
      uppy.cancelAll();
      uppy = null;
    }
  });

  async function refreshMediaList() {
    try {
      const mediaItems = await sopApi.getStepMedia(sopId, stepId);
      localPhotos = mediaItems;
      notifyChange();
    } catch (error) {
      console.error('Failed to refresh media list:', error);
    }
  }

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

    // Process all selected files
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      
      // Validate file type
      if (!file.type.startsWith('image/') && !file.type.startsWith('video/')) {
        alert(`File "${file.name}" is not a valid image or video file. Skipping.`);
        continue;
      }

      // Add metadata for server processing
      uppy?.addFile({
        name: file.name,
        type: file.type,
        data: file,
        meta: {
          stepId: stepId.toString(),
          filename: file.name,
          filetype: file.type
        }
      });
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
  <div class="space-y-2">
    <input
      bind:this={fileInputRef}
      type="file"
      accept="image/*,video/*"
      multiple
      onchange={handleFileSelect}
      class="hidden"
      aria-label="Upload photos or videos"
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
        Uploading... {uploadProgress}%
      {:else}
        <Upload class="w-4 h-4 mr-2" />
        Upload Media
      {/if}
    </Button>
    
    {#if uploading}
      <!-- Progress bar -->
      <div class="w-full bg-muted rounded-full h-2 overflow-hidden">
        <div 
          class="bg-primary h-full transition-all duration-300" 
          style="width: {uploadProgress}%"
        ></div>
      </div>
    {/if}
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


