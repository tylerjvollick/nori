<script lang="ts">
	import type { TaskTreeResponse, TaskDepsResponse } from '$lib/api/task';
	import { taskApi } from '$lib/api/task';
	import { spaceStore } from '$lib/stores/space';
	import type { TaskResponse, TaskDep } from '$lib/types/task';
	import type { StationResponse } from '$lib/types/station';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Separator } from '$lib/components/ui/separator';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import {
		ChevronRight,
		ChevronDown,
		Circle,
		CircleDot,
		CircleCheck,
		CirclePause,
		CircleX,
		CircleMinus,
		Plus,
		Trash2,
		Link as LinkIcon,
		Unlink,
		Clock,
		X,
		Check,
	} from '@lucide/svelte';
	import { formatDuration } from '$lib/utils/time';

	// ---- Types ----

	type EditorContext = 'recipe' | 'job';

	interface Props {
		/** The full task tree (root + children). */
		tree: TaskTreeResponse;
		/** Available stations for the dropdown. */
		stations: StationResponse[];
		/** Editor context: determines which fields are shown. */
		context: EditorContext;
		/** Station map for display (id -> name). */
		stationMap: Map<string, string>;
		/** Dependency data per task. */
		deps: Map<string, TaskDepsResponse>;
		/** Space members for assignee dropdown (job context). */
		members?: { userId: string; user: { firstName?: string; lastName?: string; email: string } }[];
		/** Called after any mutation so the parent can refresh the tree. */
		onmutate?: () => void;
		/** Called when a task is selected. */
		onselect?: (task: TaskResponse) => void;
		/** Currently selected task ID. */
		selectedTaskId?: string | null;
	}

	let {
		tree,
		stations,
		context,
		stationMap,
		deps,
		members = [],
		onmutate,
		onselect,
		selectedTaskId = null,
	}: Props = $props();

	let spaceId = $derived($spaceStore.currentSpace?.id ?? '');

	// ---- State ----

	/** Set of collapsed node IDs. */
	let collapsed = $state<Set<string>>(new Set());

	/** Task being inline-edited (ID -> field -> value). */
	let editingTaskId = $state<string | null>(null);
	let editingField = $state<string | null>(null);
	let editValue = $state('');

	/** Add child dialog state. */
	let addChildParentId = $state<string | null>(null);
	let addChildTitle = $state('');
	let addChildOpen = $state(false);

	/** Delete confirmation state. */
	let deleteTaskId = $state<string | null>(null);
	let deleteTaskTitle = $state('');
	let deleteHasChildren = $state(false);
	let deleteOpen = $state(false);

	/** Dep management state. */
	let depTaskId = $state<string | null>(null);
	let depOpen = $state(false);

	/** Loading/saving state. */
	let isSaving = $state(false);

	// ---- Flatten tree for display ----

	interface FlatRow {
		task: TaskTreeResponse;
		depth: number;
		hasChildren: boolean;
		isLastChild: boolean;
		parentPath: string[];
	}

	function flattenTree(node: TaskTreeResponse, depth: number, parentPath: string[]): FlatRow[] {
		const rows: FlatRow[] = [];
		const children = [...(node.children ?? [])].sort((a, b) => a.displayOrder - b.displayOrder);

		if (depth > 0) {
			rows.push({
				task: node,
				depth,
				hasChildren: children.length > 0,
				isLastChild: false, // set by parent
				parentPath,
			});
		}

		if (!collapsed.has(node.id)) {
			for (let i = 0; i < children.length; i++) {
				const childRows = flattenTree(children[i], depth + 1, [...parentPath, node.id]);
				if (childRows.length > 0) {
					childRows[0].isLastChild = i === children.length - 1;
				}
				rows.push(...childRows);
			}
		}

		return rows;
	}

	let flatRows = $derived.by((): FlatRow[] => {
		if (!tree) return [];
		// For the root, show children directly (root is context header)
		const children = [...(tree.children ?? [])].sort((a, b) => a.displayOrder - b.displayOrder);
		const rows: FlatRow[] = [];
		for (let i = 0; i < children.length; i++) {
			const childRows = flattenTree(children[i], 1, [tree.id]);
			if (childRows.length > 0) {
				childRows[0].isLastChild = i === children.length - 1;
			}
			rows.push(...childRows);
		}
		return rows;
	});

	// ---- Collapse/expand ----

	function toggleCollapse(taskId: string): void {
		const next = new Set(collapsed);
		if (next.has(taskId)) {
			next.delete(taskId);
		} else {
			next.add(taskId);
		}
		collapsed = next;
	}

	function childCount(node: TaskTreeResponse): number {
		let count = 0;
		function walk(n: TaskTreeResponse): void {
			for (const c of n.children ?? []) {
				count++;
				walk(c);
			}
		}
		walk(node);
		return count;
	}

	// ---- Status helpers ----

	type StatusConfig = { label: string; colorClass: string; bgClass: string };

	function getStatusConfig(status: string): StatusConfig {
		switch (status) {
			case 'active': return { label: 'Active', colorClass: 'text-blue-500', bgClass: 'bg-blue-500/10' };
			case 'done': return { label: 'Done', colorClass: 'text-green-500', bgClass: 'bg-green-500/10' };
			case 'paused': return { label: 'Paused', colorClass: 'text-yellow-500', bgClass: 'bg-yellow-500/10' };
			case 'skipped': return { label: 'Skipped', colorClass: 'text-muted-foreground', bgClass: 'bg-muted' };
			case 'cancelled': return { label: 'Cancelled', colorClass: 'text-red-500', bgClass: 'bg-red-500/10' };
			default: return { label: 'Open', colorClass: 'text-muted-foreground', bgClass: 'bg-muted' };
		}
	}

	function getStationName(stationId: string | null | undefined): string | null {
		if (!stationId) return null;
		return stationMap.get(stationId) ?? null;
	}

	// ---- Inline editing ----

	function startEditing(taskId: string, field: string, currentValue: string): void {
		editingTaskId = taskId;
		editingField = field;
		editValue = currentValue;
	}

	async function saveEdit(): Promise<void> {
		if (!editingTaskId || !editingField) return;
		isSaving = true;
		try {
			const update: Record<string, unknown> = {};
			if (editingField === 'title') {
				update.title = editValue;
			} else if (editingField === 'description') {
				update.description = editValue;
			}
			await taskApi.updateTask(spaceId, editingTaskId, update as { title?: string; description?: string });
			cancelEdit();
			onmutate?.();
		} catch (err) {
			console.error('Failed to save:', err);
		} finally {
			isSaving = false;
		}
	}

	function cancelEdit(): void {
		editingTaskId = null;
		editingField = null;
		editValue = '';
	}

	function handleEditKeydown(e: KeyboardEvent): void {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			saveEdit();
		} else if (e.key === 'Escape') {
			cancelEdit();
		}
	}

	// ---- Station assignment ----

	async function updateStation(taskId: string, stationId: string | undefined): Promise<void> {
		isSaving = true;
		try {
			await taskApi.updateTask(spaceId, taskId, { stationId: stationId || undefined } as { stationId?: string });
			onmutate?.();
		} catch (err) {
			console.error('Failed to update station:', err);
		} finally {
			isSaving = false;
		}
	}

	// ---- Add child task ----

	function openAddChild(parentId: string): void {
		addChildParentId = parentId;
		addChildTitle = '';
		addChildOpen = true;
	}

	async function confirmAddChild(): Promise<void> {
		if (!addChildParentId || !addChildTitle.trim()) return;
		isSaving = true;
		try {
			await taskApi.addChildTask(spaceId, addChildParentId, {
				title: addChildTitle.trim(),
				type: context === 'recipe' ? 'task' : 'task',
			});
			addChildOpen = false;
			addChildTitle = '';
			addChildParentId = null;
			onmutate?.();
		} catch (err) {
			console.error('Failed to add child:', err);
		} finally {
			isSaving = false;
		}
	}

	// ---- Delete task ----

	function openDelete(task: TaskTreeResponse): void {
		deleteTaskId = task.id;
		deleteTaskTitle = task.title;
		deleteHasChildren = (task.children?.length ?? 0) > 0;
		deleteOpen = true;
	}

	async function confirmDelete(): Promise<void> {
		if (!deleteTaskId) return;
		isSaving = true;
		try {
			await taskApi.deleteTask(spaceId, deleteTaskId);
			deleteOpen = false;
			deleteTaskId = null;
			onmutate?.();
		} catch (err) {
			console.error('Failed to delete:', err);
		} finally {
			isSaving = false;
		}
	}

	// ---- Sibling helpers ----

	function getSiblings(task: TaskTreeResponse): TaskTreeResponse[] {
		function findParent(node: TaskTreeResponse): TaskTreeResponse | null {
			for (const child of node.children ?? []) {
				if (child.id === task.id) return node;
				const found = findParent(child);
				if (found) return found;
			}
			return null;
		}
		const parent = findParent(tree);
		if (!parent) return [];
		return [...(parent.children ?? [])].sort((a, b) => a.displayOrder - b.displayOrder);
	}

	// ---- Dependency management ----

	function openDepManager(taskId: string): void {
		depTaskId = taskId;
		depOpen = true;
	}

	let depSiblings = $derived.by((): TaskResponse[] => {
		if (!depTaskId) return [];
		const task = findTaskInTree(tree, depTaskId);
		if (!task) return [];
		const siblings = getSiblings(task);
		return siblings.filter(s => s.id !== depTaskId);
	});

	let depBlockers = $derived.by((): TaskDep[] => {
		if (!depTaskId) return [];
		const taskDeps = deps.get(depTaskId);
		if (!taskDeps) return [];
		return taskDeps.blockers;
	});

	let depBlockerIds = $derived(new Set(depBlockers.map(d => d.toTaskId)));

	async function addDependency(blockerId: string): Promise<void> {
		if (!depTaskId) return;
		isSaving = true;
		try {
			// blockerId blocks depTaskId: POST /tasks/{blockerId}/deps with targetTaskId=depTaskId
			await taskApi.addDep(spaceId, blockerId, depTaskId);
			onmutate?.();
		} catch (err) {
			console.error('Failed to add dependency:', err);
		} finally {
			isSaving = false;
		}
	}

	async function removeDependency(dep: TaskDep): Promise<void> {
		if (!depTaskId) return;
		isSaving = true;
		try {
			await taskApi.removeDep(spaceId, dep.toTaskId, dep.id);
			onmutate?.();
		} catch (err) {
			console.error('Failed to remove dependency:', err);
		} finally {
			isSaving = false;
		}
	}

	// ---- Helpers ----

	function findTaskInTree(node: TaskTreeResponse, id: string): TaskTreeResponse | null {
		if (node.id === id) return node;
		for (const child of node.children ?? []) {
			const found = findTaskInTree(child, id);
			if (found) return found;
		}
		return null;
	}

	function getBlockerCount(taskId: string): number {
		const taskDeps = deps.get(taskId);
		if (!taskDeps) return 0;
		return taskDeps.blockers.length;
	}

	function getMemberName(userId: string): string {
		const member = members?.find(m => m.userId === userId);
		if (!member) return userId.slice(0, 8);
		const u = member.user;
		if (u.firstName && u.lastName) return `${u.firstName} ${u.lastName}`;
		if (u.firstName) return u.firstName;
		return u.email;
	}
</script>

{#snippet statusIcon(status: string, sizeClass: string)}
	{@const cfg = getStatusConfig(status)}
	{#if status === 'active'}
		<CircleDot class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'done'}
		<CircleCheck class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'paused'}
		<CirclePause class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'cancelled'}
		<CircleX class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'skipped'}
		<CircleMinus class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else}
		<Circle class="{sizeClass} {cfg.colorClass} shrink-0" />
	{/if}
{/snippet}

<!-- Root header -->
<div class="space-y-1">
	<div class="flex items-center justify-between px-2 py-2">
		<div class="flex items-center gap-2">
			{@render statusIcon(tree.status, 'w-5 h-5')}
			<span class="text-sm font-semibold text-foreground">{tree.title}</span>
			<Badge variant="outline" class="text-[10px] py-0 capitalize">{context}</Badge>
		</div>
		<Tooltip.Provider>
			<Tooltip.Root>
				<Tooltip.Trigger>
					<Button
						variant="ghost"
						size="sm"
						class="h-7 w-7 p-0"
						onclick={() => openAddChild(tree.id)}
					>
						<Plus class="w-4 h-4" />
					</Button>
				</Tooltip.Trigger>
				<Tooltip.Content>Add step</Tooltip.Content>
			</Tooltip.Root>
		</Tooltip.Provider>
	</div>
	<Separator />
</div>

<!-- Tree rows -->
<div class="space-y-0">
	{#each flatRows as row (row.task.id)}
		{@const task = row.task}
		{@const isSelected = task.id === selectedTaskId}
		{@const isEditing = editingTaskId === task.id}
		{@const station = getStationName(task.stationId)}
		{@const blockerCount = getBlockerCount(task.id)}
		{@const isCollapsed = collapsed.has(task.id)}
		{@const numChildren = childCount(task)}

		<div
			class="group flex items-center gap-1 py-1.5 px-2 rounded-md transition-colors
				{isSelected ? 'bg-accent ring-1 ring-inset ring-primary/30' : 'hover:bg-accent/50'}"
			onclick={() => onselect?.(task)}
			onkeydown={(e) => { if (e.key === 'Enter') onselect?.(task); }}
			role="button"
			tabindex="0"
		>
			<!-- Tree indentation -->
			{#if row.depth > 1}
				<span class="inline-flex shrink-0" style="width: {(row.depth - 1) * 16}px">
					{#each Array(row.depth - 2) as _}
						<span class="inline-block w-4 border-l border-border/50"></span>
					{/each}
					{#if row.depth > 1}
						<span class="inline-flex w-4 items-center text-muted-foreground/40">&#x2514;</span>
					{/if}
				</span>
			{/if}

			<!-- Collapse toggle -->
			{#if row.hasChildren}
				<button
					class="p-0.5 rounded hover:bg-accent transition-colors shrink-0"
					onclick={(e) => { e.stopPropagation(); toggleCollapse(task.id); }}
					title={isCollapsed ? `Expand (${numChildren})` : `Collapse (${numChildren})`}
				>
					{#if isCollapsed}
						<ChevronRight class="w-3.5 h-3.5 text-muted-foreground" />
					{:else}
						<ChevronDown class="w-3.5 h-3.5 text-muted-foreground" />
					{/if}
				</button>
			{:else}
				<div class="w-4.5 shrink-0"></div>
			{/if}

			<!-- Status icon -->
			{@render statusIcon(task.status, 'w-4 h-4')}

			<!-- Title (inline editable) -->
			<div class="flex-1 min-w-0 flex items-center gap-1.5">
				{#if isEditing && editingField === 'title'}
					<form
						class="flex items-center gap-1 flex-1"
						onsubmit={(e) => { e.preventDefault(); saveEdit(); }}
					>
						<Input
							class="h-6 text-sm py-0 px-1"
							bind:value={editValue}
							onkeydown={handleEditKeydown}
							autofocus
						/>
						<Button variant="ghost" size="sm" class="h-6 w-6 p-0" type="submit" disabled={isSaving}>
							<Check class="w-3 h-3" />
						</Button>
						<Button variant="ghost" size="sm" class="h-6 w-6 p-0" type="button" onclick={cancelEdit}>
							<X class="w-3 h-3" />
						</Button>
					</form>
				{:else}
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<span
						class="text-sm text-foreground truncate cursor-text"
						ondblclick={(e) => { e.stopPropagation(); startEditing(task.id, 'title', task.title); }}
					>
						{task.title}
					</span>
					{#if task.type === 'gate'}
						<Badge variant="outline" class="text-[10px] py-0">Gate</Badge>
					{:else if task.type === 'milestone'}
						<Badge variant="outline" class="text-[10px] py-0">Milestone</Badge>
					{/if}
				{/if}
			</div>

			<!-- Badges and metadata -->
			<div class="flex items-center gap-1.5 shrink-0">
				<!-- Station badge -->
				{#if station}
					<Badge variant="secondary" class="text-[10px] py-0">{station}</Badge>
				{/if}

				<!-- Dependency indicators -->
				{#if blockerCount > 0}
					<Tooltip.Provider>
						<Tooltip.Root>
							<Tooltip.Trigger>
								<Badge variant="outline" class="text-[10px] py-0 text-orange-500 border-orange-500/30">
									<LinkIcon class="w-2.5 h-2.5 mr-0.5" />
									{blockerCount}
								</Badge>
							</Tooltip.Trigger>
							<Tooltip.Content>Blocked by {blockerCount} task{blockerCount > 1 ? 's' : ''}</Tooltip.Content>
						</Tooltip.Root>
					</Tooltip.Provider>
				{/if}

				<!-- Recipe context: estimated time, batch size -->
				{#if context === 'recipe'}
					{#if task.estimatedTimeSeconds}
						<span class="text-[10px] text-muted-foreground flex items-center gap-0.5">
							<Clock class="w-2.5 h-2.5" />
							{formatDuration(task.estimatedTimeSeconds)}
						</span>
					{/if}
				{/if}

				<!-- Job context: actual time, status, assignee -->
				{#if context === 'job'}
					{#if task.actualTimeSeconds > 0}
						<span class="text-[10px] text-muted-foreground flex items-center gap-0.5">
							<Clock class="w-2.5 h-2.5" />
							{formatDuration(task.actualTimeSeconds)}
						</span>
					{/if}
					{#if task.assignedToId}
						<span class="text-[10px] text-muted-foreground truncate max-w-[60px]">
							{getMemberName(task.assignedToId)}
						</span>
					{/if}
				{/if}

				<!-- Collapsed child count -->
				{#if isCollapsed && numChildren > 0}
					<span class="text-[10px] text-muted-foreground/60">+{numChildren}</span>
				{/if}

				<!-- Action buttons (visible on hover) -->
				<div class="hidden group-hover:flex items-center gap-0.5">
					<Tooltip.Provider>
						<Tooltip.Root>
							<Tooltip.Trigger>
								<Button
									variant="ghost"
									size="sm"
									class="h-5 w-5 p-0"
									onclick={(e) => { e.stopPropagation(); openAddChild(task.id); }}
								>
									<Plus class="w-3 h-3" />
								</Button>
							</Tooltip.Trigger>
							<Tooltip.Content>Add child step</Tooltip.Content>
						</Tooltip.Root>
					</Tooltip.Provider>

					<Tooltip.Provider>
						<Tooltip.Root>
							<Tooltip.Trigger>
								<Button
									variant="ghost"
									size="sm"
									class="h-5 w-5 p-0"
									onclick={(e) => { e.stopPropagation(); openDepManager(task.id); }}
								>
									<LinkIcon class="w-3 h-3" />
								</Button>
							</Tooltip.Trigger>
							<Tooltip.Content>Manage dependencies</Tooltip.Content>
						</Tooltip.Root>
					</Tooltip.Provider>

					<Tooltip.Provider>
						<Tooltip.Root>
							<Tooltip.Trigger>
								<Button
									variant="ghost"
									size="sm"
									class="h-5 w-5 p-0 text-destructive hover:text-destructive"
									onclick={(e) => { e.stopPropagation(); openDelete(task); }}
								>
									<Trash2 class="w-3 h-3" />
								</Button>
							</Tooltip.Trigger>
							<Tooltip.Content>Delete step</Tooltip.Content>
						</Tooltip.Root>
					</Tooltip.Provider>
				</div>
			</div>
		</div>
	{/each}

	{#if flatRows.length === 0}
		<div class="text-sm text-muted-foreground text-center py-8 px-4">
			<p>No steps yet.</p>
			<Button
				variant="outline"
				size="sm"
				class="mt-3 gap-1.5"
				onclick={() => openAddChild(tree.id)}
			>
				<Plus class="w-4 h-4" />
				Add first step
			</Button>
		</div>
	{/if}
</div>

<!-- Add Child Dialog -->
<Dialog.Root bind:open={addChildOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Add Step</Dialog.Title>
			<Dialog.Description>
				Add a new step under the selected task.
			</Dialog.Description>
		</Dialog.Header>
		<form onsubmit={(e) => { e.preventDefault(); confirmAddChild(); }}>
			<div class="space-y-4 py-4">
				<div class="space-y-2">
					<label for="add-child-title" class="text-sm font-medium">Title</label>
					<Input
						id="add-child-title"
						placeholder="Step title..."
						bind:value={addChildTitle}
						autofocus
					/>
				</div>
			</div>
			<Dialog.Footer>
				<Button variant="outline" type="button" onclick={() => (addChildOpen = false)}>
					Cancel
				</Button>
				<Button type="submit" disabled={!addChildTitle.trim() || isSaving}>
					{isSaving ? 'Adding...' : 'Add Step'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

<!-- Delete Confirmation Dialog -->
<Dialog.Root bind:open={deleteOpen}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Delete Step</Dialog.Title>
			<Dialog.Description>
				{#if deleteHasChildren}
					This will delete "<strong>{deleteTaskTitle}</strong>" and all its children. This cannot be undone.
				{:else}
					Delete "<strong>{deleteTaskTitle}</strong>"? This cannot be undone.
				{/if}
			</Dialog.Description>
		</Dialog.Header>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (deleteOpen = false)}>Cancel</Button>
			<Button variant="destructive" onclick={confirmDelete} disabled={isSaving}>
				{isSaving ? 'Deleting...' : 'Delete'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Dependency Manager Dialog -->
<Dialog.Root bind:open={depOpen}>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>Manage Dependencies</Dialog.Title>
			<Dialog.Description>
				{#if depTaskId}
					{@const task = findTaskInTree(tree, depTaskId)}
					Configure what blocks "<strong>{task?.title ?? depTaskId}</strong>".
				{/if}
			</Dialog.Description>
		</Dialog.Header>
		<div class="space-y-4 py-4">
			<!-- Current blockers -->
			{#if depBlockers.length > 0}
				<div class="space-y-2">
					<h4 class="text-sm font-medium">Blocked by</h4>
					<div class="space-y-1">
						{#each depBlockers as dep (dep.id)}
							{@const blockerTask = findTaskInTree(tree, dep.toTaskId)}
							<div class="flex items-center justify-between py-1 px-2 rounded-md bg-muted/50">
								<span class="text-sm">{blockerTask?.title ?? dep.toTaskId}</span>
								<Button
									variant="ghost"
									size="sm"
									class="h-6 w-6 p-0 text-destructive hover:text-destructive"
									onclick={() => removeDependency(dep)}
									disabled={isSaving}
								>
									<Unlink class="w-3.5 h-3.5" />
								</Button>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Add blocker from siblings -->
			{#if depSiblings.length > 0}
				<div class="space-y-2">
					<h4 class="text-sm font-medium">Add blocker</h4>
					<div class="space-y-1">
						{#each depSiblings as sibling (sibling.id)}
							{@const isBlocker = depBlockerIds.has(sibling.id)}
							{#if !isBlocker}
								<div class="flex items-center justify-between py-1 px-2 rounded-md hover:bg-muted/50">
									<div class="flex items-center gap-2">
										{@render statusIcon(sibling.status, 'w-3.5 h-3.5')}
										<span class="text-sm">{sibling.title}</span>
									</div>
									<Button
										variant="ghost"
										size="sm"
										class="h-6 px-2 text-xs"
										onclick={() => addDependency(sibling.id)}
										disabled={isSaving}
									>
										<LinkIcon class="w-3 h-3 mr-1" />
										Add
									</Button>
								</div>
							{/if}
						{/each}
					</div>
				</div>
			{/if}

			{#if depSiblings.length === 0 && depBlockers.length === 0}
				<p class="text-sm text-muted-foreground text-center py-4">
					No sibling tasks to add as dependencies.
				</p>
			{/if}
		</div>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (depOpen = false)}>Done</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
