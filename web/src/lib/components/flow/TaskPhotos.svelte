<script lang="ts">
	import { taskApi } from '$lib/api/task';
	import { resolveMediaUrl } from '$lib/api/client';
	import type { TaskMedia } from '$lib/types/task';
	import { Button } from '$lib/components/ui/button';
	import { ImagePlus, Loader2, Play, Images } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';
	import MediaViewerModal from './MediaViewerModal.svelte';

	interface Props {
		spaceId: string;
		taskId: string;
		/** Used as the viewer modal heading. */
		title?: string;
	}

	let { spaceId, taskId, title = 'Photos' }: Props = $props();

	let media = $state<TaskMedia[]>([]);
	let loading = $state(true);
	let uploading = $state(false);
	let fileInput = $state<HTMLInputElement | null>(null);

	// Viewer modal state.
	let viewerOpen = $state(false);
	let viewerStartIndex = $state(0);

	let viewerItems = $derived(
		media.map((m) => ({ id: m.id, url: m.url, isVideo: isVideo(m) })),
	);

	// Show at most the first three photos in the row; if there are more, the
	// third tile gets a "+N" overlay indicating the rest (all are still
	// reachable in the viewer).
	let visibleMedia = $derived(media.slice(0, 3));
	let overflowCount = $derived(Math.max(0, media.length - 3));

	async function load() {
		loading = true;
		try {
			const res = await taskApi.listTaskMedia(spaceId, taskId);
			media = res.items;
		} catch (e) {
			// Surface nothing on initial load failure; the section just stays empty.
			media = [];
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		// Re-load whenever the task changes.
		taskId;
		load();
	});

	function pickFile() {
		fileInput?.click();
	}

	async function onFileSelected(e: Event) {
		const input = e.target as HTMLInputElement;
		const files = input.files;
		if (!files || files.length === 0) return;

		uploading = true;
		try {
			for (const file of Array.from(files)) {
				const created = await taskApi.uploadTaskMedia(spaceId, taskId, file);
				media = [...media, created];
			}
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to upload photo');
		} finally {
			uploading = false;
			// Reset so selecting the same file again still fires change.
			input.value = '';
		}
	}

	function openViewer(index: number) {
		viewerStartIndex = index;
		viewerOpen = true;
	}

	async function deleteById(id: string) {
		const prev = media;
		media = media.filter((m) => m.id !== id);
		try {
			await taskApi.deleteTaskMedia(spaceId, taskId, id);
		} catch (err) {
			media = prev; // restore on failure
			toast.error(err instanceof Error ? err.message : 'Failed to delete photo');
			return;
		}
		if (media.length === 0) viewerOpen = false;
	}

	function isVideo(m: TaskMedia): boolean {
		return m.mimeType.startsWith('video/');
	}
</script>

<div class="space-y-2" data-testid="task-photos">
	<div class="flex items-center justify-between">
		<h4 class="text-sm font-medium text-foreground">Photos</h4>
		<Button
			variant="ghost"
			size="sm"
			class="h-7 gap-1 text-xs"
			onclick={pickFile}
			disabled={uploading}
			data-testid="task-photos-add"
		>
			{#if uploading}
				<Loader2 class="size-3.5 animate-spin" />
			{:else}
				<ImagePlus class="size-3.5" />
			{/if}
			Add photo
		</Button>
		<input
			bind:this={fileInput}
			type="file"
			accept="image/*,video/*"
			multiple
			class="hidden"
			onchange={onFileSelected}
			data-testid="task-photos-input"
		/>
	</div>

	{#if loading}
		<p class="text-xs text-muted-foreground">Loading photos…</p>
	{:else if media.length === 0}
		<p class="text-xs text-muted-foreground">No photos yet.</p>
	{:else}
		<div class="grid grid-cols-3 gap-2" data-testid="task-photos-grid">
			{#each visibleMedia as item, i (item.id)}
				<button
					type="button"
					class="group/photo relative aspect-square rounded-md border border-border overflow-hidden bg-muted"
					onclick={() => openViewer(i)}
					title={i === 2 && overflowCount > 0 ? `View all ${media.length} photos` : 'View photo'}
					data-testid="task-photo"
				>
					{#if isVideo(item)}
						<!-- svelte-ignore a11y_media_has_caption -->
						<video src={resolveMediaUrl(item.url)} class="w-full h-full object-cover" muted preload="metadata"></video>
						{#if !(i === 2 && overflowCount > 0)}
							<span class="absolute inset-0 flex items-center justify-center bg-black/20 pointer-events-none">
								<Play class="size-6 text-white/90" />
							</span>
						{/if}
					{:else}
						<img src={resolveMediaUrl(item.url)} alt={item.fileName} class="w-full h-full object-cover" />
					{/if}

					{#if i === 2 && overflowCount > 0}
						<span
							class="absolute inset-0 flex items-center justify-center gap-1 bg-black/60 text-white text-sm font-medium pointer-events-none"
							data-testid="task-photos-more"
						>
							<Images class="size-4" />
							+{overflowCount}
						</span>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

<!-- Full-size viewer (shared with sub-step photos). -->
<MediaViewerModal
	bind:open={viewerOpen}
	{title}
	items={viewerItems}
	startIndex={viewerStartIndex}
	ondelete={deleteById}
/>
