<script lang="ts">
	import { Dialog } from 'bits-ui';
	import * as Carousel from '$lib/components/ui/carousel';
	import type { CarouselAPI } from '$lib/components/ui/carousel/context';
	import { resolveMediaUrl } from '$lib/api/client';
	import type { SubTaskImage } from '$lib/types/task';
	import { Trash2, ImageIcon, X } from '@lucide/svelte';

	interface Props {
		open: boolean;
		title: string;
		description?: string | null;
		images: SubTaskImage[];
		/** Called when the user deletes an image; parent owns the API call + state. */
		ondelete?: (imageId: string) => void;
	}

	let { open = $bindable(false), title, description = null, images, ondelete }: Props = $props();

	// Embla measures slide sizes on init. Inside a portal/dialog that first
	// measurement can happen before layout settles, so re-measure once after the
	// dialog opens and whenever the image set changes (e.g. after a delete).
	// Slides use a fixed-height frame, so image loads do NOT require a reInit.
	let emblaApi = $state<CarouselAPI | undefined>(undefined);

	// Current slide (0-based) for the "n / total" position indicator.
	let selectedIndex = $state(0);

	$effect(() => {
		void open;
		void images.length;
		if (!open || !emblaApi) return;
		const id = requestAnimationFrame(() => emblaApi?.reInit());
		return () => cancelAnimationFrame(id);
	});

	$effect(() => {
		const api = emblaApi;
		if (!api) return;
		const update = () => (selectedIndex = api.selectedScrollSnap());
		update();
		api.on('select', update);
		api.on('reInit', update);
		return () => {
			api.off('select', update);
			api.off('reInit', update);
		};
	});
</script>

<Dialog.Root bind:open>
	<Dialog.Portal>
		<!-- Plain dark overlay — intentionally NO backdrop-blur: blurring the page
		     behind (which includes the live SvelteFlow graph) is GPU-expensive and
		     makes the modal feel sluggish to open. -->
		<Dialog.Overlay
			class="fixed inset-0 z-50 bg-black/70 data-open:animate-in data-closed:animate-out data-open:fade-in-0 data-closed:fade-out-0 duration-100"
		/>
		<Dialog.Content
			class="fixed left-1/2 top-1/2 z-50 w-full max-w-4xl -translate-x-1/2 -translate-y-1/2 rounded-md bg-popover text-popover-foreground p-4 shadow-lg outline-none data-open:animate-in data-closed:animate-out data-open:fade-in-0 data-closed:fade-out-0 duration-100"
		>
			<div class="flex items-start justify-between gap-2 mb-3">
				<Dialog.Title class="text-sm font-medium">{title}</Dialog.Title>
				<Dialog.Close
					class="text-muted-foreground hover:text-foreground transition-colors"
					aria-label="Close"
				>
					<X class="size-4" />
				</Dialog.Close>
			</div>

			{#if images.length > 0}
				<Carousel.Root
					class="w-full"
					opts={{ loop: images.length > 2, align: 'center' }}
					setApi={(api) => (emblaApi = api)}
				>
					<Carousel.Content>
						{#each images as img (img.id)}
							<Carousel.Item>
								<!-- Fixed-height frame keeps slide dimensions stable regardless of
								     the loaded image's aspect ratio, so embla never has to relayout. -->
								<div class="relative flex h-[65vh] items-center justify-center rounded-md bg-muted/30">
									<img
										src={resolveMediaUrl(img.imageUrl)}
										alt={title}
										class="max-h-full max-w-full object-contain"
										data-testid="subtask-modal-image"
									/>
									<button
										type="button"
										class="absolute top-2 right-2 p-1.5 rounded bg-black/50 text-white/90 hover:text-red-400 transition-colors"
										title="Delete image"
										aria-label="Delete image"
										onclick={() => ondelete?.(img.id)}
										data-testid="subtask-modal-delete"
									>
										<Trash2 class="size-4" />
									</button>
								</div>
							</Carousel.Item>
						{/each}
					</Carousel.Content>
					{#if images.length > 1}
						<Carousel.Previous />
						<Carousel.Next />
					{/if}
				</Carousel.Root>

				{#if images.length > 1}
					<div
						class="mt-2 text-center text-xs font-medium text-muted-foreground tabular-nums"
						data-testid="subtask-modal-counter"
					>
						{selectedIndex + 1} / {images.length}
					</div>
				{/if}
			{:else}
				<div class="flex flex-col items-center justify-center gap-2 py-12 text-muted-foreground">
					<ImageIcon class="size-8 opacity-40" />
					<p class="text-sm">No photos for this step yet.</p>
				</div>
			{/if}

			{#if description}
				<p class="text-sm text-muted-foreground whitespace-pre-wrap mt-3">{description}</p>
			{/if}
		</Dialog.Content>
	</Dialog.Portal>
</Dialog.Root>
