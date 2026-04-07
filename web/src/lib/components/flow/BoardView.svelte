<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { taskApi } from '$lib/api/task';
	import { stationApi } from '$lib/api/station';
	import type { TaskResponse } from '$lib/types/task';
	import type { StationResponse } from '$lib/types/station';
	import { Button } from '$lib/components/ui/button';
	import { RefreshCw, AlertCircle } from 'lucide-svelte';
	import { isEditableTarget, showToast } from '$lib/utils/keyboard';
	import KanbanColumn from './KanbanColumn.svelte';
	import TaskCard from './TaskCard.svelte';

	const POLL_INTERVAL_MS = 30_000;
	const DONE_LIMIT = 20;

	// ---- State ----

	let readyTasks = $state<TaskResponse[]>([]);
	let blockedTasks = $state<TaskResponse[]>([]);
	let inProgressTasks = $state<TaskResponse[]>([]);
	let doneTasks = $state<TaskResponse[]>([]);

	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let isRefreshing = $state(false);
	let lastRefreshed = $state<Date | null>(null);
	let pollTimer: ReturnType<typeof setInterval> | null = null;

	let stationMap = $state<Map<string, string>>(new Map());

	// ---- Keyboard selection state ----
	// [columnIndex, cardIndex] — -1 means nothing selected
	let selCol = $state(-1);
	let selCard = $state(-1);

	/** Get the tasks array for a given column index. */
	function columnTasks(col: number): TaskResponse[] {
		switch (col) {
			case 0: return blockedTasks;
			case 1: return readyTasks;
			case 2: return inProgressTasks;
			case 3: return doneTasks;
			default: return [];
		}
	}

	const COLUMN_COUNT = 4;

	/** Get the currently selected task, or null. */
	let selectedTask = $derived.by(() => {
		if (selCol < 0 || selCard < 0) return null;
		const tasks = columnTasks(selCol);
		return tasks[selCard] ?? null;
	});

	/** Clamp the card index to the column's bounds. */
	function clampCard(col: number, card: number): number {
		const len = columnTasks(col).length;
		if (len === 0) return -1;
		return Math.max(0, Math.min(card, len - 1));
	}

	function clearSelection(): void {
		selCol = -1;
		selCard = -1;
	}

	// ---- Derived filters from URL ----

	let stationFilter = $derived($page.url.searchParams.get('station') || '');
	let priorityFilter = $derived($page.url.searchParams.get('priority') || '');

	// ---- Data fetching ----

	async function fetchStations(): Promise<void> {
		try {
			const stations: StationResponse[] = await stationApi.listStations();
			const map = new Map<string, string>();
			for (const s of stations) {
				map.set(s.id, s.name);
			}
			stationMap = map;
		} catch {
			// Stations endpoint may not exist yet — gracefully degrade.
		}
	}

	/** Build common filter params from URL state. */
	function filterParams(): { stationId?: string; priority?: string } {
		const params: { stationId?: string; priority?: string } = {};
		if (stationFilter) params.stationId = stationFilter;
		// Priority filter is applied client-side for the ready endpoint
		// and as a query param for listTasks
		return params;
	}

	async function fetchAllColumns(opts?: { silent?: boolean }): Promise<void> {
		if (!opts?.silent) {
			isLoading = true;
		}
		isRefreshing = true;
		error = null;

		try {
			const filters = filterParams();

			// Fetch all data in parallel
			const [readyResult, openResult, activeResult, pausedResult, doneResult, skippedResult] =
				await Promise.all([
					// Ready tasks
					taskApi.getReadyTasks({
						stationId: filters.stationId,
					}),
					// Open tasks (to compute blocked = open minus ready)
					taskApi.listTasks({
						status: 'open',
						stationId: filters.stationId,
						limit: 200,
					}),
					// Active tasks
					taskApi.listTasks({
						status: 'active',
						stationId: filters.stationId,
						limit: 200,
					}),
					// Paused tasks
					taskApi.listTasks({
						status: 'paused',
						stationId: filters.stationId,
						limit: 200,
					}),
					// Done tasks
					taskApi.listTasks({
						status: 'done',
						stationId: filters.stationId,
						limit: DONE_LIMIT,
					}),
					// Skipped tasks
					taskApi.listTasks({
						status: 'skipped',
						stationId: filters.stationId,
						limit: DONE_LIMIT,
					}),
				]);

			// Build a set of ready task IDs to exclude from "open" → "blocked"
			const readyIds = new Set(readyResult.map((t) => t.id));

			// Apply client-side priority filter if set
			const pFilter = priorityFilter ? Number(priorityFilter) : null;
			const filterByPriority = (tasks: TaskResponse[]): TaskResponse[] => {
				if (pFilter === null) return tasks;
				return tasks.filter((t) => t.priority === pFilter);
			};

			readyTasks = filterByPriority(readyResult);
			blockedTasks = filterByPriority(openResult.items.filter((t) => !readyIds.has(t.id)));
			inProgressTasks = filterByPriority([...activeResult.items, ...pausedResult.items]);

			// Merge done + skipped, sort by updatedAt desc, cap at DONE_LIMIT
			const allDone = [...doneResult.items, ...skippedResult.items];
			allDone.sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime());
			doneTasks = filterByPriority(allDone.slice(0, DONE_LIMIT));

			lastRefreshed = new Date();

			// After data refresh, clamp selection to stay in bounds
			if (selCol >= 0) {
				selCard = clampCard(selCol, selCard);
				if (selCard < 0) {
					// Column is now empty, try to find a non-empty column
					let found = false;
					for (let i = 0; i < COLUMN_COUNT; i++) {
						if (columnTasks(i).length > 0) {
							selCol = i;
							selCard = 0;
							found = true;
							break;
						}
					}
					if (!found) clearSelection();
				}
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load board data';
		} finally {
			isLoading = false;
			isRefreshing = false;
		}
	}

	function startPolling(): void {
		stopPolling();
		pollTimer = setInterval(() => fetchAllColumns({ silent: true }), POLL_INTERVAL_MS);
	}

	function stopPolling(): void {
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	function handleManualRefresh(): void {
		fetchAllColumns({ silent: true });
	}

	// Re-fetch when filters change
	let prevStation = $state('');
	let prevPriority = $state('');

	$effect(() => {
		const s = stationFilter;
		const p = priorityFilter;
		if (s !== prevStation || p !== prevPriority) {
			prevStation = s;
			prevPriority = p;
			// Don't refetch on initial mount — onMount handles that
			if (lastRefreshed) {
				fetchAllColumns({ silent: true });
			}
		}
	});

	onMount(async () => {
		await Promise.all([fetchAllColumns(), fetchStations()]);
		startPolling();
	});

	onDestroy(() => {
		stopPolling();
	});

	function formatLastRefreshed(date: Date | null): string {
		if (!date) return '';
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	/** After any task action, refresh the board to reflect the new state. */
	function handleTaskAction(): void {
		fetchAllColumns({ silent: true });
	}

	// ---- Keyboard navigation ----

	function handleKeydown(e: KeyboardEvent): void {
		if (isEditableTarget(e)) return;

		switch (e.key) {
			case 'h': {
				// Move selection left (previous column)
				e.preventDefault();
				if (selCol <= 0) {
					// Not selected yet or at leftmost — select first card of first non-empty column
					for (let i = 0; i < COLUMN_COUNT; i++) {
						if (columnTasks(i).length > 0) {
							selCol = i;
							selCard = 0;
							break;
						}
					}
				} else {
					// Move left, find previous non-empty column
					for (let i = selCol - 1; i >= 0; i--) {
						if (columnTasks(i).length > 0) {
							selCol = i;
							selCard = clampCard(i, selCard);
							break;
						}
					}
				}
				scrollSelectedIntoView();
				break;
			}
			case 'l': {
				// Move selection right (next column)
				// Also consumed here to prevent layout from switching to list view
				e.preventDefault();
				e.stopPropagation();
				if (selCol < 0) {
					// Not selected — select first card of first non-empty column
					for (let i = 0; i < COLUMN_COUNT; i++) {
						if (columnTasks(i).length > 0) {
							selCol = i;
							selCard = 0;
							break;
						}
					}
				} else {
					// Move right, find next non-empty column
					for (let i = selCol + 1; i < COLUMN_COUNT; i++) {
						if (columnTasks(i).length > 0) {
							selCol = i;
							selCard = clampCard(i, selCard);
							break;
						}
					}
				}
				scrollSelectedIntoView();
				break;
			}
			case 'j': {
				// Move selection down (next card in column)
				e.preventDefault();
				if (selCol < 0) {
					// Not selected — select first card of first non-empty column
					for (let i = 0; i < COLUMN_COUNT; i++) {
						if (columnTasks(i).length > 0) {
							selCol = i;
							selCard = 0;
							break;
						}
					}
				} else {
					const len = columnTasks(selCol).length;
					if (selCard < len - 1) {
						selCard++;
					}
				}
				scrollSelectedIntoView();
				break;
			}
			case 'k': {
				// Move selection up (previous card in column)
				e.preventDefault();
				if (selCol < 0) {
					// Not selected — select first card of first non-empty column
					for (let i = 0; i < COLUMN_COUNT; i++) {
						if (columnTasks(i).length > 0) {
							selCol = i;
							selCard = 0;
							break;
						}
					}
				} else {
					if (selCard > 0) {
						selCard--;
					}
				}
				scrollSelectedIntoView();
				break;
			}
			case 'Enter': {
				// Open selected task detail
				const task = selectedTask;
				if (task) {
					e.preventDefault();
					goto(`/flow/${task.id}`);
				}
				break;
			}
			case 'c': {
				// Claim selected task
				const task = selectedTask;
				if (task && task.status === 'open' && !task.assignedToId) {
					e.preventDefault();
					claimSelectedTask(task);
				}
				break;
			}
			case 'd': {
				// Complete selected task (mark done)
				const task = selectedTask;
				if (task && task.status === 'active') {
					e.preventDefault();
					completeSelectedTask(task);
				}
				break;
			}
			case 'Escape': {
				if (selCol >= 0) {
					clearSelection();
				}
				break;
			}
		}
	}

	async function claimSelectedTask(task: TaskResponse): Promise<void> {
		try {
			await taskApi.claimTask(task.id);
			showToast(`Claimed: ${task.title}`);
			fetchAllColumns({ silent: true });
		} catch (err) {
			console.error('Failed to claim task:', err);
			showToast('Failed to claim task');
		}
	}

	async function completeSelectedTask(task: TaskResponse): Promise<void> {
		try {
			await taskApi.completeTask(task.id);
			showToast(`Completed: ${task.title}`);
			fetchAllColumns({ silent: true });
		} catch (err) {
			console.error('Failed to complete task:', err);
			showToast('Failed to complete task');
		}
	}

	function scrollSelectedIntoView(): void {
		requestAnimationFrame(() => {
			const el = document.querySelector('[data-kb-selected="true"]') as HTMLElement;
			el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
		});
	}

	/** Check if a specific card in a column is the keyboard-selected one. */
	function isSelected(colIndex: number, cardIndex: number): boolean {
		return selCol === colIndex && selCard === cardIndex;
	}
</script>

<svelte:head>
	<title>Flow Board - Nori</title>
</svelte:head>

<svelte:window onkeydown={handleKeydown} />

<div class="flex h-full flex-col overflow-hidden">
	<!-- Board header -->
	<div class="flex-shrink-0 flex items-center justify-between px-4 py-2">
		<h1 class="text-lg font-semibold text-foreground">Board</h1>
		<div class="flex items-center gap-3">
			{#if lastRefreshed}
				<span class="text-xs text-muted-foreground">
					Updated {formatLastRefreshed(lastRefreshed)}
				</span>
			{/if}
			<Button variant="outline" size="sm" onclick={handleManualRefresh} disabled={isRefreshing}>
				<RefreshCw class="size-4 {isRefreshing ? 'animate-spin' : ''}" />
				<span class="ml-1.5">Refresh</span>
			</Button>
		</div>
	</div>

	<!-- Error banner -->
	{#if error}
		<div class="mx-4 mb-2 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm">
			<AlertCircle class="size-4 text-destructive shrink-0" />
			<span class="text-destructive">{error}</span>
			<Button variant="outline" size="sm" class="ml-auto" onclick={() => fetchAllColumns()}>
				Retry
			</Button>
		</div>
	{/if}

	<!-- Kanban columns -->
	<div class="flex flex-1 gap-4 overflow-x-auto px-4 pb-4">
		<!-- Blocked -->
		<KanbanColumn
			title="Blocked"
			count={blockedTasks.length}
			colorClass="bg-red-500"
			isLoading={isLoading}
		>
			{#each blockedTasks as task, i (task.id)}
				<div
					class="rounded-lg {isSelected(0, i) ? 'ring-2 ring-primary ring-offset-1' : ''}"
					data-kb-selected={isSelected(0, i)}
				>
					<TaskCard {task} {stationMap} onaction={handleTaskAction} />
				</div>
			{/each}
		</KanbanColumn>

		<!-- Ready -->
		<KanbanColumn
			title="Ready"
			count={readyTasks.length}
			colorClass="bg-blue-500"
			isLoading={isLoading}
		>
			{#each readyTasks as task, i (task.id)}
				<div
					class="rounded-lg {isSelected(1, i) ? 'ring-2 ring-primary ring-offset-1' : ''}"
					data-kb-selected={isSelected(1, i)}
				>
					<TaskCard {task} {stationMap} onaction={handleTaskAction} />
				</div>
			{/each}
		</KanbanColumn>

		<!-- In Progress -->
		<KanbanColumn
			title="In Progress"
			count={inProgressTasks.length}
			colorClass="bg-yellow-500"
			isLoading={isLoading}
		>
			{#each inProgressTasks as task, i (task.id)}
				<div
					class="rounded-lg {isSelected(2, i) ? 'ring-2 ring-primary ring-offset-1' : ''}"
					data-kb-selected={isSelected(2, i)}
				>
					<TaskCard {task} {stationMap} onaction={handleTaskAction} />
				</div>
			{/each}
		</KanbanColumn>

		<!-- Done -->
		<KanbanColumn
			title="Done"
			count={doneTasks.length}
			colorClass="bg-green-500"
			isLoading={isLoading}
		>
			{#each doneTasks as task, i (task.id)}
				<div
					class="rounded-lg {isSelected(3, i) ? 'ring-2 ring-primary ring-offset-1' : ''}"
					data-kb-selected={isSelected(3, i)}
				>
					<TaskCard {task} {stationMap} onaction={handleTaskAction} />
				</div>
			{/each}
		</KanbanColumn>
	</div>
</div>
