<script lang="ts">
	import { dndzone, type DndEvent } from 'svelte-dnd-action';
	import { flip } from 'svelte/animate';
	import type { SubTaskResponse } from '$lib/types/task';
	import { taskApi } from '$lib/api/task';
	import { resolveMediaUrl } from '$lib/api/client';
	import SubTaskImageModal from './SubTaskImageModal.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { toast } from 'svelte-sonner';
	import {
		List,
		LayoutGrid,
		GripVertical,
		Plus,
		Trash2,
		ChevronDown,
		ChevronRight,
		ImageIcon,
		ImagePlus,
		Images,
		Pencil,
		Check,
		X,
	} from '@lucide/svelte';

	interface Props {
		spaceId: string;
		taskId: string;
	}

	let { spaceId, taskId }: Props = $props();

	type ViewMode = 'list' | 'grid';

	// --- State ---
	let viewMode = $state<ViewMode>('list');
	let items = $state<(SubTaskResponse & { id: string })[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Inline add
	let addingTitle = $state('');
	let addingActive = $state(false);
	let addingLoading = $state(false);

	// Per-item expanded (list view)
	let expanded = $state<Set<string>>(new Set());

	// Per-item edit state
	interface EditState { title: string; description: string }
	let editing = $state<Map<string, EditState>>(new Map());

	// --- Load ---
	async function loadSubTasks() {
		loading = true;
		error = null;
		try {
			const res = await taskApi.listSubTasks(spaceId, taskId);
			items = res.items as (SubTaskResponse & { id: string })[];
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load sub-tasks';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		loadSubTasks();
	});

	// --- Add sub-task ---
	async function submitAdd() {
		const title = addingTitle.trim();
		if (!title) return;
		addingLoading = true;
		try {
			const created = await taskApi.createSubTask(spaceId, taskId, { title });
			items = [...items, created as SubTaskResponse & { id: string }];
			addingTitle = '';
			addingActive = false;
		} catch {
			// keep form open on error
		} finally {
			addingLoading = false;
		}
	}

	function cancelAdd() {
		addingTitle = '';
		addingActive = false;
	}

	let addInputEl: HTMLInputElement | null = $state(null);

	async function submitAndContinue() {
		await submitAdd();
		// Re-open the add input immediately for rapid entry
		addingActive = true;
		// Need a tick for the input to mount before focusing
		requestAnimationFrame(() => addInputEl?.focus());
	}

	function onAddKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && e.metaKey) {
			// Cmd+Enter: submit and immediately re-open for next sub-task
			e.preventDefault();
			submitAndContinue();
		} else if (e.key === 'Enter') {
			e.preventDefault();
			submitAdd();
		} else if (e.key === 'Escape') {
			cancelAdd();
		}
	}

	// --- Delete ---
	async function deleteItem(id: string) {
		try {
			await taskApi.deleteSubTask(spaceId, taskId, id);
			items = items.filter((i) => i.id !== id);
			expanded.delete(id);
			editing.delete(id);
		} catch {
			// silent — could surface toast
		}
	}

	// --- Inline edit ---
	function startEdit(item: SubTaskResponse) {
		editing.set(item.id, { title: item.title, description: item.description ?? '' });
		editing = new Map(editing); // trigger reactivity
	}

	function cancelEdit(id: string) {
		editing.delete(id);
		editing = new Map(editing);
	}

	async function commitEdit(id: string) {
		const state = editing.get(id);
		if (!state) return;
		try {
			const updated = await taskApi.updateSubTask(spaceId, taskId, id, {
				title: state.title.trim() || undefined,
				description: state.description.trim() || undefined,
			});
			items = items.map((i) => (i.id === id ? (updated as SubTaskResponse & { id: string }) : i));
			editing.delete(id);
			editing = new Map(editing);
		} catch {
			// keep edit open
		}
	}

	function onEditTitleKeydown(e: KeyboardEvent, id: string) {
		if (e.key === 'Enter') { e.preventDefault(); commitEdit(id); }
		if (e.key === 'Escape') cancelEdit(id);
	}

	// --- Expand toggle (list view) ---
	function toggleExpand(id: string) {
		if (expanded.has(id)) {
			expanded.delete(id);
		} else {
			expanded.add(id);
		}
		expanded = new Set(expanded);
	}

	// --- Drag-to-reorder ---
	let dragging = $state(false);

	function handleDndConsider(e: CustomEvent<DndEvent<SubTaskResponse & { id: string }>>) {
		items = e.detail.items;
		dragging = true;
	}

	async function handleDndFinalize(e: CustomEvent<DndEvent<SubTaskResponse & { id: string }>>) {
		items = e.detail.items;
		dragging = false;
		try {
			await taskApi.reorderSubTasks(spaceId, taskId, { ids: items.map((i) => i.id) });
		} catch {
			// Reload to re-sync order on failure
			await loadSubTasks();
		}
	}

	const flipDurationMs = 150;

	// --- Image attachment ---
	let imageInput = $state<HTMLInputElement | null>(null);
	let uploadTargetId = $state<string | null>(null);
	let uploadingId = $state<string | null>(null);

	function pickImage(subtaskId: string) {
		uploadTargetId = subtaskId;
		imageInput?.click();
	}

	async function onImageSelected(e: Event) {
		const input = e.target as HTMLInputElement;
		const files = input.files ? Array.from(input.files) : [];
		const targetId = uploadTargetId;
		input.value = '';
		uploadTargetId = null;
		if (files.length === 0 || !targetId) return;

		uploadingId = targetId;
		try {
			for (const file of files) {
				const img = await taskApi.uploadSubTaskImage(spaceId, taskId, targetId, file);
				items = items.map((i) =>
					i.id === targetId ? { ...i, images: [...(i.images ?? []), img] } : i,
				);
			}
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to attach image');
		} finally {
			uploadingId = null;
		}
	}

	// --- Full-size image viewer modal ---
	let modalSubTaskId = $state<string | null>(null);
	let modalOpen = $state(false);

	let modalSubTask = $derived(
		modalSubTaskId ? (items.find((i) => i.id === modalSubTaskId) ?? null) : null,
	);

	function openModal(subtaskId: string) {
		modalSubTaskId = subtaskId;
		modalOpen = true;
	}

	async function deleteImage(subtaskId: string, imageId: string) {
		const prev = items;
		// Optimistically remove from local state.
		items = items.map((i) =>
			i.id === subtaskId ? { ...i, images: (i.images ?? []).filter((img) => img.id !== imageId) } : i,
		);
		try {
			await taskApi.deleteSubTaskImage(spaceId, taskId, subtaskId, imageId);
		} catch (err) {
			items = prev; // restore on failure
			toast.error(err instanceof Error ? err.message : 'Failed to delete image');
			return;
		}
		// Close the modal once the last image is gone.
		if ((modalSubTask?.images?.length ?? 0) === 0) {
			modalOpen = false;
		}
	}
</script>

<div class="space-y-3">
	<!-- Hidden file input shared by all sub-task rows for image attachment. -->
	<input
		bind:this={imageInput}
		type="file"
		accept="image/*"
		multiple
		class="hidden"
		onchange={onImageSelected}
		data-testid="subtask-image-input"
	/>
	<!-- Header with view toggle -->
	<div class="flex items-center justify-between">
		<h4 class="text-sm font-medium text-foreground">Sub-tasks</h4>
		<div class="flex items-center gap-1">
			<Button
				variant={viewMode === 'list' ? 'secondary' : 'ghost'}
				size="icon"
				class="size-7"
				onclick={() => (viewMode = 'list')}
				title="List view"
			>
				<List class="size-3.5" />
			</Button>
			<Button
				variant={viewMode === 'grid' ? 'secondary' : 'ghost'}
				size="icon"
				class="size-7"
				onclick={() => (viewMode = 'grid')}
				title="Grid view"
			>
				<LayoutGrid class="size-3.5" />
			</Button>
		</div>
	</div>

	{#if loading}
		<div class="text-xs text-muted-foreground py-2">Loading…</div>
	{:else if error}
		<div class="text-xs text-red-500 py-2">{error}</div>
	{:else}

		<!-- ===== LIST VIEW ===== -->
		{#if viewMode === 'list'}
			{#if items.length > 0}
				<div
					use:dndzone={{ items, flipDurationMs, type: 'subtask-list', dropTargetStyle: {} }}
					onconsider={handleDndConsider}
					onfinalize={handleDndFinalize}
					class="space-y-1"
				>
					{#each items as item (item.id)}
						<div
							animate:flip={{ duration: flipDurationMs }}
							class="group/item rounded-md border border-border bg-card hover:border-border/80 transition-colors"
						>
							<!-- Row -->
							<div class="flex items-center gap-1.5 px-2 py-1.5">
								<!-- Drag handle -->
								<div class="cursor-grab text-muted-foreground/40 hover:text-muted-foreground shrink-0">
									<GripVertical class="size-3.5" />
								</div>

								<!-- Photo thumbnail (click to open the full-size viewer). When a
								     sub-task has more than one image, peek the extras behind the
								     first to signal there are multiple. -->
								<button
									type="button"
									class="relative size-14 shrink-0"
									onclick={() => openModal(item.id)}
									title={item.images && item.images.length > 0 ? 'View photos' : 'No photos yet'}
									data-testid="subtask-thumb"
								>
									{#if item.images && item.images.length > 0}
										{#if item.images.length > 2}
											<div class="absolute inset-0 rounded border border-border bg-muted overflow-hidden rotate-[6deg] translate-x-1">
												<img src={resolveMediaUrl(item.images[2].imageUrl)} alt="" class="size-full object-cover" />
											</div>
										{/if}
										{#if item.images.length > 1}
											<div class="absolute inset-0 rounded border border-border bg-muted overflow-hidden -rotate-[5deg] -translate-x-1">
												<img src={resolveMediaUrl(item.images[1].imageUrl)} alt="" class="size-full object-cover" />
											</div>
										{/if}
										<div class="absolute inset-0 rounded border border-border bg-muted overflow-hidden">
											<img src={resolveMediaUrl(item.images[0].imageUrl)} alt="" class="size-full object-cover" />
										</div>
										{#if item.images.length > 1}
											<span class="absolute -top-1 -right-1 z-10 flex items-center gap-0.5 rounded-full bg-foreground/80 px-1.5 py-0.5 text-[10px] font-medium text-background" data-testid="subtask-image-count">
												{item.images.length}
											</span>
										{/if}
									{:else}
										<div class="size-full rounded border border-border bg-muted flex items-center justify-center">
											<ImageIcon class="size-5 text-muted-foreground/30" />
										</div>
									{/if}
								</button>

								<!-- Expand toggle (only if description exists) -->
								<button
									class="shrink-0 text-muted-foreground/40 hover:text-muted-foreground"
									onclick={() => toggleExpand(item.id)}
									type="button"
								>
									{#if item.description}
										{#if expanded.has(item.id)}
											<ChevronDown class="size-3.5" />
										{:else}
											<ChevronRight class="size-3.5" />
										{/if}
									{:else}
										<span class="w-3.5 h-3.5 inline-block"></span>
									{/if}
								</button>

								<!-- Title (edit or display) -->
								{#if editing.has(item.id)}
									<div class="flex flex-1 items-center gap-1">
										<Input
											value={editing.get(item.id)!.title}
											oninput={(e) => {
												const s = editing.get(item.id);
												if (s) editing.set(item.id, { ...s, title: (e.target as HTMLInputElement).value });
											}}
											onkeydown={(e) => onEditTitleKeydown(e, item.id)}
											class="h-6 text-xs flex-1"
											autofocus
										/>
										<button
											class="text-green-500 hover:text-green-600"
											onclick={() => commitEdit(item.id)}
											type="button"
										>
											<Check class="size-3.5" />
										</button>
										<button
											class="text-muted-foreground hover:text-foreground"
											onclick={() => cancelEdit(item.id)}
											type="button"
										>
											<X class="size-3.5" />
										</button>
									</div>
								{:else}
									<span class="text-sm flex-1 leading-tight">{item.title}</span>
									<!-- Action buttons (hover) -->
									<div class="flex items-center gap-0.5 opacity-0 group-hover/item:opacity-100 transition-opacity shrink-0">
										<button
											class="text-muted-foreground/60 hover:text-foreground p-0.5 disabled:opacity-50"
											onclick={() => pickImage(item.id)}
											type="button"
											title="Attach image"
											disabled={uploadingId === item.id}
											data-testid="subtask-image-add"
										>
											<ImagePlus class="size-3" />
										</button>
										<button
											class="text-muted-foreground/60 hover:text-foreground p-0.5"
											onclick={() => startEdit(item)}
											type="button"
											title="Edit"
										>
											<Pencil class="size-3" />
										</button>
										<button
											class="text-muted-foreground/60 hover:text-red-500 p-0.5"
											onclick={() => deleteItem(item.id)}
											type="button"
											title="Delete"
										>
											<Trash2 class="size-3" />
										</button>
									</div>
								{/if}
							</div>

							<!-- Expanded description + edit -->
							{#if expanded.has(item.id) && item.description}
								<div class="px-8 pb-2">
									{#if editing.has(item.id)}
										<Textarea
											value={editing.get(item.id)!.description}
											oninput={(e) => {
												const s = editing.get(item.id);
												if (s) editing.set(item.id, { ...s, description: (e.target as HTMLTextAreaElement).value });
											}}
											class="text-xs min-h-[60px] resize-none"
											placeholder="Description…"
										/>
									{:else}
										<p class="text-xs text-muted-foreground whitespace-pre-wrap">{item.description}</p>
									{/if}
								</div>
							{/if}
						</div>
					{/each}
				</div>
			{:else}
				<p class="text-xs text-muted-foreground py-1">No sub-tasks yet.</p>
			{/if}
		{/if}

		<!-- ===== GRID VIEW ===== -->
		{#if viewMode === 'grid'}
			{#if items.length > 0}
				<div
					use:dndzone={{ items, flipDurationMs, type: 'subtask-grid', dropTargetStyle: {} }}
					onconsider={handleDndConsider}
					onfinalize={handleDndFinalize}
					class="grid grid-cols-3 gap-2"
				>
					{#each items as item (item.id)}
						<div
							animate:flip={{ duration: flipDurationMs }}
							class="group/card rounded-md border border-border bg-card overflow-hidden flex flex-col cursor-grab active:cursor-grabbing"
						>
							<!-- Photo area (click opens the full-size viewer) -->
							<div class="aspect-square bg-muted relative">
								<button
									type="button"
									class="absolute inset-0 flex items-center justify-center"
									onclick={() => openModal(item.id)}
									title={item.images && item.images.length > 0 ? 'View photos' : 'No photos yet'}
									data-testid="subtask-card"
								>
									{#if item.images && item.images.length > 0}
										<img
											src={resolveMediaUrl(item.images[0].imageUrl)}
											alt={item.title}
											class="w-full h-full object-cover"
										/>
									{:else}
										<ImageIcon class="size-6 text-muted-foreground/30" />
									{/if}
								</button>

								<!-- Multi-photo indicator -->
								{#if item.images && item.images.length > 1}
									<span
										class="absolute top-1 left-1 z-10 flex items-center gap-0.5 rounded-full bg-black/55 px-1.5 py-0.5 text-[10px] font-medium text-white pointer-events-none"
										data-testid="subtask-card-multi"
									>
										<Images class="size-3" />
										{item.images.length}
									</span>
								{/if}

								<!-- Hover overlay actions (sibling of the trigger; container is
								     pointer-events-none so clicks fall through to the viewer). -->
								<div class="absolute inset-0 bg-black/40 flex items-center justify-center gap-2 opacity-0 group-hover/card:opacity-100 transition-opacity pointer-events-none">
									<button
										class="text-white/80 hover:text-red-400 p-1 bg-black/30 rounded pointer-events-auto"
										onclick={() => deleteItem(item.id)}
										type="button"
										title="Delete"
									>
										<Trash2 class="size-3.5" />
									</button>
									<button
										class="text-white/80 hover:text-white p-1 bg-black/30 rounded pointer-events-auto"
										onclick={() => startEdit(item)}
										type="button"
										title="Edit"
									>
										<Pencil class="size-3.5" />
									</button>
								</div>
							</div>

							<!-- Caption -->
							<div class="p-1.5 flex-1 min-w-0">
								{#if editing.has(item.id)}
									<Input
										value={editing.get(item.id)!.title}
										oninput={(e) => {
											const s = editing.get(item.id);
											if (s) editing.set(item.id, { ...s, title: (e.target as HTMLInputElement).value });
										}}
										onkeydown={(e) => onEditTitleKeydown(e, item.id)}
										class="h-5 text-[11px] mb-1"
										autofocus
									/>
									{#if item.description || editing.get(item.id)?.description}
										<Input
											value={editing.get(item.id)!.description}
											oninput={(e) => {
												const s = editing.get(item.id);
												if (s) editing.set(item.id, { ...s, description: (e.target as HTMLInputElement).value });
											}}
											class="h-5 text-[10px]"
											placeholder="Description…"
										/>
									{/if}
									<div class="flex gap-1 mt-1">
										<button class="text-green-500 hover:text-green-600" onclick={() => commitEdit(item.id)} type="button">
											<Check class="size-3" />
										</button>
										<button class="text-muted-foreground hover:text-foreground" onclick={() => cancelEdit(item.id)} type="button">
											<X class="size-3" />
										</button>
									</div>
								{:else}
									<p class="text-[11px] font-medium leading-tight truncate">{item.title}</p>
									{#if item.description}
										<p class="text-[10px] text-muted-foreground leading-tight line-clamp-2 mt-0.5">{item.description}</p>
									{/if}
								{/if}
							</div>
						</div>
					{/each}
				</div>
			{:else}
				<p class="text-xs text-muted-foreground py-1">No sub-tasks yet.</p>
			{/if}
		{/if}

		<!-- ===== ADD SUB-TASK ===== -->
		{#if addingActive}
			<div class="flex items-center gap-1.5 mt-1">
				<Input
					bind:value={addingTitle}
					bind:ref={addInputEl}
					placeholder="Sub-task title…"
					class="h-7 text-xs flex-1"
					onkeydown={onAddKeydown}
					autofocus
					disabled={addingLoading}
				/>
				<Button
					size="sm"
					class="h-7 px-2 text-xs"
					onclick={submitAdd}
					disabled={addingLoading || !addingTitle.trim()}
				>
					Add
				</Button>
				<Button
					variant="ghost"
					size="sm"
					class="h-7 px-2 text-xs"
					onclick={cancelAdd}
					disabled={addingLoading}
				>
					Cancel
				</Button>
			</div>
		{:else}
			<button
				class="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors mt-1 w-full py-1"
				onclick={() => (addingActive = true)}
				type="button"
			>
				<Plus class="size-3.5" />
				<span>Add sub-task</span>
			</button>
		{/if}
	{/if}
</div>

<!-- Full-size image viewer for the selected sub-step. -->
{#if modalSubTask}
	<SubTaskImageModal
		bind:open={modalOpen}
		title={modalSubTask.title}
		description={modalSubTask.description}
		images={modalSubTask.images ?? []}
		ondelete={(imageId) => deleteImage(modalSubTask!.id, imageId)}
	/>
{/if}
