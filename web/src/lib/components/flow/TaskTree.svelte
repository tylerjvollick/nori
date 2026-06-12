<script lang="ts">
	import type { TaskTreeResponse, TaskDepsResponse } from '$lib/api/task';
	import type { TaskResponse } from '$lib/types/task';
	import { Badge } from '$lib/components/ui/badge';
	import {
		ChevronRight,
		ChevronDown,
		Circle,
		CircleDot,
		CircleCheck,
		CircleX,
		CircleMinus,
		Clock,
		User,
	} from '@lucide/svelte';
	import { formatDuration } from '$lib/utils/time';
	import { isEditableTarget } from '$lib/utils/keyboard.svelte';

	interface Props {
		/** Flat list of tasks (descendants of the job/parent). */
		tasks: TaskResponse[];
		/** Dependency data per task. */
		deps: Map<string, TaskDepsResponse>;
		stationMap: Map<string, string>;
		selectedTaskId?: string | null;
		onselect: (task: TaskResponse) => void;
	}

	let {
		tasks,
		deps,
		stationMap,
		selectedTaskId = null,
		onselect,
	}: Props = $props();

	// ---- Dependency chain tree building ----

	/** A row in the flattened dependency tree display. */
	interface TreeRow {
		task: TaskResponse;
		depth: number;
		chainRootId: string;
		isChainRoot: boolean;
		hasChildren: boolean;
	}

	/**
	 * Build dependency chains from tasks and their deps.
	 * Returns a flat array of TreeRow entries in display order.
	 *
	 * Root tasks (no predecessors) at the top, final convergence
	 * points / milestones at the bottom. Reading top-to-bottom =
	 * following the flow of work.
	 */
	function buildDependencyChains(
		tasks: TaskResponse[],
		deps: Map<string, TaskDepsResponse>,
	): TreeRow[] {
		if (tasks.length === 0) return [];

		const taskSet = new Set(tasks.map((t) => t.id));
		const taskMap = new Map(tasks.map((t) => [t.id, t]));

		// Build adjacency lists scoped to visible tasks
		const successors = new Map<string, string[]>();
		const predecessors = new Map<string, string[]>();

		for (const id of taskSet) {
			successors.set(id, []);
			predecessors.set(id, []);
		}

		for (const [taskId, taskDeps] of deps) {
			if (!taskSet.has(taskId)) continue;

			// dependents: tasks that depend on taskId (downstream)
			// dep.fromTaskId = downstream task, dep.toTaskId = taskId (upstream/blocker)
			for (const dep of taskDeps.dependents) {
				if (taskSet.has(dep.fromTaskId) && dep.toTaskId === taskId) {
					const succs = successors.get(taskId);
					if (succs && !succs.includes(dep.fromTaskId)) {
						succs.push(dep.fromTaskId);
					}
					const preds = predecessors.get(dep.fromTaskId);
					if (preds && !preds.includes(taskId)) {
						preds.push(taskId);
					}
				}
			}

			// blockers: tasks that block taskId (upstream)
			// dep.fromTaskId = taskId (blocked), dep.toTaskId = blocker/upstream
			for (const dep of taskDeps.blockers) {
				if (taskSet.has(dep.toTaskId) && dep.fromTaskId === taskId) {
					const succs = successors.get(dep.toTaskId);
					if (succs && !succs.includes(taskId)) {
						succs.push(taskId);
					}
					const preds = predecessors.get(taskId);
					if (preds && !preds.includes(dep.toTaskId)) {
						preds.push(dep.toTaskId);
					}
				}
			}
		}

		// Find root tasks: no predecessors within the set
		const roots = tasks.filter((t) => {
			const preds = predecessors.get(t.id);
			return !preds || preds.length === 0;
		});

		// Sort roots by priority (ascending), then by ID for stability
		roots.sort((a, b) => {
			const cmp = a.priority - b.priority;
			if (cmp !== 0) return cmp;
			return a.id.localeCompare(b.id);
		});

		// DFS to build the flat row list
		const rows: TreeRow[] = [];
		const visited = new Set<string>();

		function dfs(taskId: string, depth: number, chainRootId: string): void {
			if (visited.has(taskId)) return;
			visited.add(taskId);

			const task = taskMap.get(taskId);
			if (!task) return;

			const succs = successors.get(taskId) ?? [];
			const unvisitedSuccs = succs.filter((s) => !visited.has(s));

			rows.push({
				task,
				depth,
				chainRootId,
				isChainRoot: depth === 0,
				hasChildren: unvisitedSuccs.length > 0,
			});

			// Sort successors by priority then ID for consistent ordering
			const sortedSuccs = [...unvisitedSuccs].sort((a, b) => {
				const tA = taskMap.get(a);
				const tB = taskMap.get(b);
				if (!tA || !tB) return 0;
				const cmp = tA.priority - tB.priority;
				if (cmp !== 0) return cmp;
				return tA.id.localeCompare(tB.id);
			});

			for (const succId of sortedSuccs) {
				const succPreds = predecessors.get(succId) ?? [];
				const allPredsVisited = succPreds.every((p) => visited.has(p));
				if (allPredsVisited) {
					dfs(succId, depth + 1, chainRootId);
				}
			}
		}

		for (const root of roots) {
			dfs(root.id, 0, root.id);
		}

		// Any tasks not yet visited get added as standalone roots
		for (const task of tasks) {
			if (!visited.has(task.id)) {
				visited.add(task.id);
				rows.push({
					task,
					depth: 0,
					chainRootId: task.id,
					isChainRoot: true,
					hasChildren: false,
				});
			}
		}

		return rows;
	}

	// ---- Build tree rows from current data ----

	let treeRows = $derived.by((): TreeRow[] => {
		if (tasks.length === 0) return [];

		// If no deps data yet, fall back to flat list sorted by displayOrder
		if (deps.size === 0) {
			const sorted = [...tasks].sort((a, b) => a.displayOrder - b.displayOrder);
			return sorted.map((t) => ({
				task: t,
				depth: 0,
				chainRootId: t.id,
				isChainRoot: true,
				hasChildren: false,
			}));
		}

		return buildDependencyChains(tasks, deps);
	});

	// ---- Chain collapse ----

	let collapsedChains = $state<Set<string>>(new Set());

	let visibleRows = $derived.by((): TreeRow[] => {
		if (collapsedChains.size === 0) return treeRows;

		return treeRows.filter((row) => {
			if (row.isChainRoot) return true;
			return !collapsedChains.has(row.chainRootId);
		});
	});

	function toggleChainCollapse(chainRootId: string): void {
		const next = new Set(collapsedChains);
		if (next.has(chainRootId)) {
			next.delete(chainRootId);
		} else {
			next.add(chainRootId);
		}
		collapsedChains = next;
	}

	function chainChildCount(chainRootId: string): number {
		return treeRows.filter(
			(r) => r.chainRootId === chainRootId && !r.isChainRoot,
		).length;
	}

	// ---- Keyboard selection ----

	let selectedRow = $state(-1);

	let selectedTreeRow = $derived.by(() => {
		if (selectedRow < 0 || selectedRow >= visibleRows.length) return null;
		return visibleRows[selectedRow] ?? null;
	});

	// Sync selectedRow with external selectedTaskId prop
	$effect(() => {
		if (selectedTaskId && visibleRows.length > 0) {
			const idx = visibleRows.findIndex((r) => r.task.id === selectedTaskId);
			if (idx >= 0 && idx !== selectedRow) {
				selectedRow = idx;
			}
		}
	});

	function scrollSelectedRowIntoView(): void {
		requestAnimationFrame(() => {
			const el = document.querySelector(
				'[data-tree-selected="true"]',
			) as HTMLElement;
			el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
		});
	}

	function handleKeydown(e: KeyboardEvent): void {
		if (isEditableTarget(e)) return;

		switch (e.key) {
			case 'j': {
				e.preventDefault();
				if (visibleRows.length === 0) break;
				if (selectedRow < 0) {
					selectedRow = 0;
				} else if (selectedRow < visibleRows.length - 1) {
					selectedRow++;
				}
				scrollSelectedRowIntoView();
				const row = visibleRows[selectedRow];
				if (row) onselect(row.task);
				break;
			}
			case 'k': {
				e.preventDefault();
				if (visibleRows.length === 0) break;
				if (selectedRow < 0) {
					selectedRow = 0;
				} else if (selectedRow > 0) {
					selectedRow--;
				}
				scrollSelectedRowIntoView();
				const row = visibleRows[selectedRow];
				if (row) onselect(row.task);
				break;
			}
			case 'h': {
				const row = selectedTreeRow;
				if (row) {
					e.preventDefault();
					toggleChainCollapse(row.chainRootId);
					// If we collapsed and the selected row is now hidden, move to the chain root
					if (collapsedChains.has(row.chainRootId) && !row.isChainRoot) {
						const rootIdx = visibleRows.findIndex(
							(r) => r.task.id === row.chainRootId,
						);
						if (rootIdx >= 0) {
							selectedRow = rootIdx;
						}
					}
				}
				break;
			}
		}
	}

	// ---- Helpers ----

	type StatusConfig = {
		label: string;
		colorClass: string;
		bgClass: string;
	};

	function getStatusConfig(status: string): StatusConfig {
		switch (status) {
			case 'in_progress':
				return { label: 'In Progress', colorClass: 'text-blue-500', bgClass: 'bg-blue-500/10' };
			case 'done':
				return { label: 'Done', colorClass: 'text-green-500', bgClass: 'bg-green-500/10' };
			case 'skipped':
				return { label: 'Skipped', colorClass: 'text-muted-foreground', bgClass: 'bg-muted' };
			case 'cancelled':
				return { label: 'Cancelled', colorClass: 'text-red-500', bgClass: 'bg-red-500/10' };
			default:
				return { label: 'Open', colorClass: 'text-muted-foreground', bgClass: 'bg-muted' };
		}
	}

	function getStationName(stationId: string | null | undefined): string | null {
		if (!stationId) return null;
		return stationMap.get(stationId) ?? stationId.slice(0, 8);
	}
</script>

{#snippet statusIcon(status: string, sizeClass: string)}
	{@const cfg = getStatusConfig(status)}
	{#if status === 'in_progress'}
		<CircleDot class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'done'}
		<CircleCheck class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'cancelled'}
		<CircleX class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'skipped'}
		<CircleMinus class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else}
		<Circle class="{sizeClass} {cfg.colorClass} shrink-0" />
	{/if}
{/snippet}

<svelte:window onkeydown={handleKeydown} />

<div class="space-y-0.5">
	{#each visibleRows as row, i (row.task.id)}
		{@const task = row.task}
		{@const isCollapsed = collapsedChains.has(row.chainRootId)}
		{@const childCount = row.isChainRoot ? chainChildCount(row.chainRootId) : 0}
		{@const isSelected = selectedRow === i}
		{@const station = getStationName(task.stationId)}
		{@const iconSize = row.depth === 0 ? 'w-5 h-5' : row.depth === 1 ? 'w-4 h-4' : 'w-3.5 h-3.5'}

		<div
			class="flex items-center gap-2 py-1.5 px-2 rounded-md transition-colors cursor-pointer
				{isSelected ? 'bg-accent ring-1 ring-inset ring-primary/30' : 'hover:bg-accent/50'}
				{row.isChainRoot && row.hasChildren ? 'font-medium' : ''}"
			onclick={() => {
				selectedRow = i;
				onselect(task);
			}}
			onkeydown={(e) => {
				if (e.key === 'Enter' || e.key === ' ') {
					selectedRow = i;
					onselect(task);
				}
			}}
			role="button"
			tabindex="0"
			data-tree-selected={isSelected}
		>
			<!-- Tree indentation -->
			{#if row.depth > 0}
				<span
					class="inline-flex shrink-0"
					style="width: {row.depth * 16}px"
				>
					{#each Array(row.depth - 1) as _}
						<span class="inline-block w-4 border-l border-border/50"></span>
					{/each}
					<span class="inline-flex w-4 items-center text-muted-foreground/40">&#x2514;</span>
				</span>
			{/if}

			<!-- Collapse toggle for chain roots with children -->
			{#if row.isChainRoot && row.hasChildren}
				<button
					class="p-0.5 rounded hover:bg-accent transition-colors shrink-0"
					onclick={(e) => {
						e.stopPropagation();
						toggleChainCollapse(row.chainRootId);
					}}
					title={isCollapsed
						? `Expand chain (${childCount} tasks)`
						: `Collapse chain (${childCount} tasks)`}
				>
					{#if isCollapsed}
						<ChevronRight class="w-4 h-4 text-muted-foreground" />
					{:else}
						<ChevronDown class="w-4 h-4 text-muted-foreground" />
					{/if}
				</button>
			{:else if row.depth === 0}
				<div class="w-5 shrink-0"></div>
			{/if}

			<!-- Status icon -->
			{@render statusIcon(task.status, iconSize)}

			<!-- Task info -->
			<div class="flex-1 min-w-0 flex items-center gap-2">
				<span class="text-sm text-foreground truncate">
					{task.title}
				</span>
				{#if task.type === 'gate'}
					<Badge variant="outline" class="text-[10px] py-0">Gate</Badge>
				{:else if task.type === 'milestone'}
					<Badge variant="outline" class="text-[10px] py-0">Milestone</Badge>
				{/if}
				{#if station}
					<Badge variant="secondary" class="text-[10px] py-0">
						{station}
					</Badge>
				{/if}
				{#if row.isChainRoot && row.hasChildren && isCollapsed}
					<span class="text-xs text-muted-foreground/60 shrink-0">
						+{childCount}
					</span>
				{/if}
			</div>

			<!-- Right side metadata -->
			<div class="flex items-center gap-2 shrink-0">
				{#if task.actualTimeSeconds > 0}
					<span class="text-xs text-muted-foreground flex items-center gap-1">
						<Clock class="w-3 h-3" />
						{formatDuration(task.actualTimeSeconds)}
					</span>
				{/if}
				{#if task.assignedToId}
					<span class="text-xs text-muted-foreground flex items-center gap-1">
						<User class="w-3 h-3" />
					</span>
				{/if}
				<span class="text-[10px] text-muted-foreground font-mono">
					{task.id}
				</span>
			</div>
		</div>
	{/each}

	{#if visibleRows.length === 0 && tasks.length > 0}
		<div class="text-sm text-muted-foreground text-center py-4">
			Loading dependency chains...
		</div>
	{/if}
</div>
