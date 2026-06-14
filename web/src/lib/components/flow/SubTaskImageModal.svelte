<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Carousel from '$lib/components/ui/carousel';
	import { resolveMediaUrl } from '$lib/api/client';
	import type { SubTaskImage } from '$lib/types/task';
	import { Trash2, ImageIcon } from '@lucide/svelte';

	interface Props {
		open: boolean;
		title: string;
		description?: string | null;
		images: SubTaskImage[];
		/** Called when the user deletes an image; parent owns the API call + state. */
		ondelete?: (imageId: string) => void;
	}

	let { open = $bindable(false), title, description = null, images, ondelete }: Props = $props();
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="max-w-3xl">
		<Dialog.Header>
			<Dialog.Title>{title}</Dialog.Title>
		</Dialog.Header>

		{#if images.length > 0}
			<Carousel.Root class="w-full" opts={{ loop: images.length > 1 }}>
				<Carousel.Content>
					{#each images as img (img.id)}
						<Carousel.Item class="flex items-center justify-center">
							<div class="relative flex w-full items-center justify-center bg-muted/30 rounded-md">
								<img
									src={resolveMediaUrl(img.imageUrl)}
									alt={title}
									class="max-h-[65vh] w-auto object-contain"
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
		{:else}
			<div class="flex flex-col items-center justify-center gap-2 py-12 text-muted-foreground">
				<ImageIcon class="size-8 opacity-40" />
				<p class="text-sm">No photos for this step yet.</p>
			</div>
		{/if}

		{#if description}
			<p class="text-sm text-muted-foreground whitespace-pre-wrap">{description}</p>
		{/if}
	</Dialog.Content>
</Dialog.Root>
