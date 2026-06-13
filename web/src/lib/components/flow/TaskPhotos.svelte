<script lang="ts">
	import { taskApi } from '$lib/api/task';
	import { resolveMediaUrl } from '$lib/api/client';
	import type { TaskMedia } from '$lib/types/task';
	import { Button } from '$lib/components/ui/button';
	import { ImagePlus, Trash2, Loader2 } from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	interface Props {
		spaceId: string;
		taskId: string;
	}

	let { spaceId, taskId }: Props = $props();

	let media = $state<TaskMedia[]>([]);
	let loading = $state(true);
	let uploading = $state(false);
	let fileInput = $state<HTMLInputElement | null>(null);

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

	async function remove(item: TaskMedia) {
		const prev = media;
		media = media.filter((m) => m.id !== item.id);
		try {
			await taskApi.deleteTaskMedia(spaceId, taskId, item.id);
		} catch (err) {
			media = prev; // restore on failure
			toast.error(err instanceof Error ? err.message : 'Failed to delete photo');
		}
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
			{#each media as item (item.id)}
				<div class="group/photo relative aspect-square rounded-md border border-border overflow-hidden bg-muted">
					{#if isVideo(item)}
						<!-- svelte-ignore a11y_media_has_caption -->
						<video src={resolveMediaUrl(item.url)} class="w-full h-full object-cover" controls></video>
					{:else}
						<img src={resolveMediaUrl(item.url)} alt={item.fileName} class="w-full h-full object-cover" />
					{/if}
					<button
						class="absolute top-1 right-1 p-1 rounded bg-black/50 text-white/90 opacity-0 group-hover/photo:opacity-100 hover:text-red-400 transition-opacity"
						onclick={() => remove(item)}
						type="button"
						title="Delete photo"
						aria-label="Delete photo"
						data-testid="task-photo-delete"
					>
						<Trash2 class="size-3.5" />
					</button>
				</div>
			{/each}
		</div>
	{/if}
</div>
