<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount, onDestroy } from 'svelte';
	import { taskApi, type ListTasksParams } from '$lib/api/task';
	import { stationApi } from '$lib/api/station';
	import type { TaskResponse, TaskStatus, TaskType } from '$lib/types/task';
	import type { StationResponse } from '$lib/types/station';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import {
		ArrowUpDown,
		ArrowUp,
		ArrowDown,
		RefreshCw,
		AlertCircle,
		Inbox,
	} from 'lucide-svelte';
	import { isEditableTarget, showToast } from '$lib/utils/keyboard';

	/** Optional pre-loaded tasks. When provided, the list uses these instead of fetching. */
	interface Props {
		tasks?: TaskResponse[];
		stationMap?: Map<string, string>;
	}

	let { tasks: externalTasks, stationMap: externalStationMap }: Props = $props();

	/** Whether we're in scoped mode (tasks provided externally). */
	let isScoped = $derived(!!externalTasks);

	const POLL_INTERVAL_MS = 30_000;
	const PAGE_SIZE = 50;

	// ---- Column definitions ----

	type SortKey =
		| 'id'
		| 'title'
		| 'status'
		| 'type'
		| 'station'
		| 'assignee'
		| 'priority'
		| 'quantity'
		| 'createdAt'
		| 'dueDate';
	type SortDir = 'asc' | 'desc';

	interface ColumnDef {
		key: SortKey;
		label: string;
		class?: string;
	}

	const COLUMNS: ColumnDef[] = [
		{ key: 'id', label: 'ID', class: 'w-[140px]' },
		{ key: 'title', label: 'Title' },
		{ key: 'status', label: 'Status', class: 'w-[100px]' },
		{ key: 'type', label: 'Type', class: 'w-[90px]' },
		{ key: 'station', label: 'Station', class: 'w-[120px]' },
		{ key: 'assignee', label: 'Assignee', class: 'w-[120px]' },
		{ key: 'priority', label: 'Priority', class: 'w-[80px]' },
		{ key: 'quantity', label: 'Qty', class: 'w-[60px]' },
		{ key: 'createdAt', label: 'Created', class: 'w-[100px]' },
		{ key: 'dueDate', label: 'Due', class: 'w-[100px]' },
	];

	// ---- State ----

	let allTasks = $state<TaskResponse[]>([]);
	let totalCount = $state(0);
	let isLoading = $state(true);
	let isLoadingMore = $state(false);
	let error = $state<string | null>(null);
	let isRefreshing = $state(false);
	let lastRefreshed = $state<Date | null>(null);
	let pollTimer: ReturnType<typeof setInterval> | null = null;

	let _internalStationMap = $state<Map<string, string>>(new Map());
	let stationMap = $derived(externalStationMap ?? _internalStationMap);

	let sortKey = $state<SortKey>('priority');
	let sortDir = $state<SortDir>('asc');

	// ---- Derived filters from URL ----

	let stationFilter = $derived($page.url.searchParams.get('station') || '');
	let statusFilter = $derived($page.url.searchParams.get('status') || '');
	let priorityFilter = $derived($page.url.searchParams.get('priority') || '');

	// ---- Sorting ----

	let sortedTasks = $derived.by(() => {
		const tasks = [...allTasks];
		const dir = sortDir === 'asc' ? 1 : -1;

		tasks.sort((a, b) => {
			let cmp = 0;
			switch (sortKey) {
				case 'id':
					cmp = a.id.localeCompare(b.id);
					break;
				case 'title':
					cmp = a.title.localeCompare(b.title);
					break;
				case 'status':
					cmp = a.status.localeCompare(b.status);
					break;
				case 'type':
					cmp = a.type.localeCompare(b.type);
					break;
				case 'station': {
					const sA = a.stationId ? (stationMap.get(a.stationId) ?? a.stationId) : '';
					const sB = b.stationId ? (stationMap.get(b.stationId) ?? b.stationId) : '';
					cmp = sA.localeCompare(sB);
					break;
				}
				case 'assignee': {
					const aA = a.assignedToId ?? '';
					const aB = b.assignedToId ?? '';
					cmp = aA.localeCompare(aB);
					break;
				}
				case 'priority':
					cmp = a.priority - b.priority;
					break;
				case 'quantity':
					cmp = a.quantity - b.quantity;
					break;
				case 'createdAt':
					cmp = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
					break;
				case 'dueDate': {
					const dA = a.dueDate ? new Date(a.dueDate).getTime() : Infinity;
					const dB = b.dueDate ? new Date(b.dueDate).getTime() : Infinity;
					cmp = dA - dB;
					break;
				}
			}
			return cmp * dir;
		});

		return tasks;
	});

	let hasMore = $derived(allTasks.length < totalCount);

	// ---- Data fetching ----

	async function fetchStations(): Promise<void> {
		try {
			const stations: StationResponse[] = await stationApi.listStations();
			const map = new Map<string, string>();
			for (const s of stations) {
				map.set(s.id, s.name);
			}
			_internalStationMap = map;
		} catch {
			// Stations endpoint may not exist yet — gracefully degrade.
		}
	}

	function buildParams(offset: number): ListTasksParams {
		const params: ListTasksParams = {
			offset,
			limit: PAGE_SIZE,
		};
		if (stationFilter) params.stationId = stationFilter;
		if (statusFilter) params.status = statusFilter;
		if (priorityFilter) {
			// Priority filter: the API doesn't support priority filtering,
			// so we fetch all and filter client-side. But if a status is set,
			// we still pass it server-side.
		}
		return params;
	}

	async function fetchTasks(opts?: { silent?: boolean; append?: boolean }): Promise<void> {
		if (!opts?.silent && !opts?.append) {
			isLoading = true;
		}
		if (opts?.append) {
			isLoadingMore = true;
		}
		isRefreshing = true;
		error = null;

		try {
			const offset = opts?.append ? allTasks.length : 0;
			const result = await taskApi.listTasks(buildParams(offset));

			let items = result.items;

			// Apply client-side priority filter if set (API doesn't support it)
			if (priorityFilter) {
				const pNum = Number(priorityFilter);
				items = items.filter((t) => t.priority === pNum);
			}

			if (opts?.append) {
				allTasks = [...allTasks, ...items];
			} else {
				allTasks = items;
			}
			totalCount = result.total;
			lastRefreshed = new Date();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load tasks';
		} finally {
			isLoading = false;
			isLoadingMore = false;
			isRefreshing = false;
		}
	}

	function loadMore(): void {
		fetchTasks({ append: true });
	}

	function handleManualRefresh(): void {
		fetchTasks({ silent: true });
	}

	function startPolling(): void {
		stopPolling();
		pollTimer = setInterval(() => fetchTasks({ silent: true }), POLL_INTERVAL_MS);
	}

	function stopPolling(): void {
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	// Re-fetch when filters change
	let prevStation = $state('');
	let prevStatus = $state('');
	let prevPriority = $state('');

	/** Load tasks from external data, applying filters client-side. */
	function loadFromExternalTasks(tasks: TaskResponse[]): void {
		let filtered = tasks;
		if (stationFilter) {
			filtered = filtered.filter((t) => t.stationId === stationFilter);
		}
		if (statusFilter) {
			filtered = filtered.filter((t) => t.status === statusFilter);
		}
		if (priorityFilter) {
			const pNum = Number(priorityFilter);
			filtered = filtered.filter((t) => t.priority === pNum);
		}
		allTasks = filtered;
		totalCount = filtered.length;
		isLoading = false;
		lastRefreshed = new Date();
	}

	// When external tasks change, reload
	$effect(() => {
		if (externalTasks) {
			loadFromExternalTasks(externalTasks);
		}
	});

	$effect(() => {
		const s = stationFilter;
		const st = statusFilter;
		const p = priorityFilter;
		if (s !== prevStation || st !== prevStatus || p !== prevPriority) {
			prevStation = s;
			prevStatus = st;
			prevPriority = p;
			if (lastRefreshed) {
				if (isScoped && externalTasks) {
					loadFromExternalTasks(externalTasks);
				} else {
					fetchTasks({ silent: true });
				}
			}
		}
	});

	onMount(async () => {
		if (isScoped) {
			// In scoped mode, tasks are provided externally. No fetching needed.
			return;
		}
		await Promise.all([fetchTasks(), fetchStations()]);
		startPolling();
	});

	onDestroy(() => {
		stopPolling();
	});

	// ---- Sort handling ----

	function handleSort(key: SortKey): void {
		if (sortKey === key) {
			sortDir = sortDir === 'asc' ? 'desc' : 'asc';
		} else {
			sortKey = key;
			sortDir = 'asc';
		}
	}

	// ---- Row click ----

	function navigateToTask(taskId: string): void {
		goto(`/flow/${taskId}`);
	}

	// ---- Keyboard selection ----

	let selectedRow = $state(-1);

	/** Get the currently selected task, or null. */
	let selectedTask = $derived.by(() => {
		if (selectedRow < 0 || selectedRow >= sortedTasks.length) return null;
		return sortedTasks[selectedRow] ?? null;
	});

	function clearSelection(): void {
		selectedRow = -1;
	}

	function scrollSelectedRowIntoView(): void {
		requestAnimationFrame(() => {
			const el = document.querySelector('[data-kb-selected="true"]') as HTMLElement;
			el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
		});
	}

	function handleKeydown(e: KeyboardEvent): void {
		if (isEditableTarget(e)) return;

		switch (e.key) {
			case 'j': {
				e.preventDefault();
				if (sortedTasks.length === 0) break;
				if (selectedRow < 0) {
					selectedRow = 0;
				} else if (selectedRow < sortedTasks.length - 1) {
					selectedRow++;
				}
				scrollSelectedRowIntoView();
				break;
			}
			case 'k': {
				e.preventDefault();
				if (sortedTasks.length === 0) break;
				if (selectedRow < 0) {
					selectedRow = 0;
				} else if (selectedRow > 0) {
					selectedRow--;
				}
				scrollSelectedRowIntoView();
				break;
			}
			case 'Enter': {
				const task = selectedTask;
				if (task) {
					e.preventDefault();
					navigateToTask(task.id);
				}
				break;
			}
			case 'c': {
				const task = selectedTask;
				if (task && task.status === 'open' && !task.assignedToId) {
					e.preventDefault();
					claimSelectedTask(task);
				}
				break;
			}
			case 'd': {
				const task = selectedTask;
				if (task && task.status === 'active') {
					e.preventDefault();
					completeSelectedTask(task);
				}
				break;
			}
			case 'Escape': {
				if (selectedRow >= 0) {
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
			fetchTasks({ silent: true });
		} catch (err) {
			console.error('Failed to claim task:', err);
			showToast('Failed to claim task');
		}
	}

	async function completeSelectedTask(task: TaskResponse): Promise<void> {
		try {
			await taskApi.completeTask(task.id);
			showToast(`Completed: ${task.title}`);
			fetchTasks({ silent: true });
		} catch (err) {
			console.error('Failed to complete task:', err);
			showToast('Failed to complete task');
		}
	}

	// ---- Helpers ----

	const STATUS_COLORS: Record<string, string> = {
		open: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
		active: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
		paused: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300',
		done: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
		skipped: 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300',
		cancelled: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
	};

	const TYPE_COLORS: Record<string, string> = {
		job: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
		task: 'bg-sky-100 text-sky-800 dark:bg-sky-900/30 dark:text-sky-300',
		milestone: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300',
		gate: 'bg-rose-100 text-rose-800 dark:bg-rose-900/30 dark:text-rose-300',
	};

	const PRIORITY_COLORS: Record<number, string> = {
		0: 'text-red-600 dark:text-red-400 font-semibold',
		1: 'text-orange-600 dark:text-orange-400 font-semibold',
		2: 'text-blue-600 dark:text-blue-400',
		3: 'text-gray-500 dark:text-gray-400',
		4: 'text-gray-400 dark:text-gray-500',
	};

	function getStationName(stationId: string | null | undefined): string {
		if (!stationId) return '\u2014';
		return stationMap.get(stationId) ?? stationId.slice(0, 8);
	}

	function getAssigneeName(assigneeId: string | null | undefined): string {
		if (!assigneeId) return '\u2014';
		return assigneeId.slice(0, 8);
	}

	function formatTimeAgo(dateStr: string | null | undefined): string {
		if (!dateStr) return '\u2014';
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60_000);
		if (diffMins < 1) return 'Just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		const diffHours = Math.floor(diffMins / 60);
		if (diffHours < 24) return `${diffHours}h ago`;
		const diffDays = Math.floor(diffHours / 24);
		if (diffDays < 30) return `${diffDays}d ago`;
		return date.toLocaleDateString();
	}

	function formatLastRefreshed(date: Date | null): string {
		if (!date) return '';
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="flex h-full flex-col overflow-hidden">
	<!-- Header -->
	<div class="flex-shrink-0 flex items-center justify-between px-4 py-2">
		<div class="flex items-center gap-3">
			<h1 class="text-lg font-semibold text-foreground">Tasks</h1>
			{#if !isLoading}
				<span class="text-xs text-muted-foreground">
					{allTasks.length} of {totalCount}
				</span>
			{/if}
		</div>
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
		<div
			class="mx-4 mb-2 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm"
		>
			<AlertCircle class="size-4 shrink-0 text-destructive" />
			<span class="text-destructive">{error}</span>
			<Button variant="outline" size="sm" class="ml-auto" onclick={() => fetchTasks()}>
				Retry
			</Button>
		</div>
	{/if}

	<!-- Table container -->
	<div class="flex-1 overflow-auto px-4 pb-4">
		{#if isLoading}
			<!-- Loading skeleton -->
			<div class="rounded-lg border">
				<div class="border-b bg-muted/50 px-4 py-3">
					<Skeleton class="h-4 w-full" />
				</div>
				{#each Array(10) as _}
					<div class="flex items-center gap-4 border-b px-4 py-3 last:border-b-0">
						<Skeleton class="h-4 w-[120px]" />
						<Skeleton class="h-4 flex-1" />
						<Skeleton class="h-5 w-[70px] rounded-full" />
						<Skeleton class="h-5 w-[60px] rounded-full" />
						<Skeleton class="h-4 w-[100px]" />
						<Skeleton class="h-4 w-[100px]" />
						<Skeleton class="h-4 w-[50px]" />
						<Skeleton class="h-4 w-[40px]" />
						<Skeleton class="h-4 w-[80px]" />
						<Skeleton class="h-4 w-[80px]" />
					</div>
				{/each}
			</div>
		{:else if allTasks.length === 0}
			<!-- Empty state -->
			<div class="rounded-lg border border-border p-12 text-center">
				<Inbox class="mx-auto mb-4 size-12 text-muted-foreground" />
				<h3 class="mb-1 text-lg font-semibold text-foreground">No tasks found</h3>
				<p class="mx-auto max-w-md text-sm text-muted-foreground">
					{#if stationFilter || statusFilter || priorityFilter}
						No tasks match the current filters. Try adjusting or clearing them.
					{:else}
						There are no tasks yet. Create a job or task to get started.
					{/if}
				</p>
			</div>
		{:else}
			<!-- Table -->
			<div class="rounded-lg border">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b bg-muted/50">
							{#each COLUMNS as col (col.key)}
								<th class="px-3 py-2 text-left font-medium text-muted-foreground {col.class ?? ''}">
									<button
										class="inline-flex items-center gap-1 hover:text-foreground transition-colors"
										onclick={() => handleSort(col.key)}
									>
										{col.label}
										{#if sortKey === col.key}
											{#if sortDir === 'asc'}
												<ArrowUp class="size-3" />
											{:else}
												<ArrowDown class="size-3" />
											{/if}
										{:else}
											<ArrowUpDown class="size-3 opacity-30" />
										{/if}
									</button>
								</th>
							{/each}
						</tr>
					</thead>
					<tbody>
					{#each sortedTasks as task, i (task.id)}
						<tr
							class="border-b last:border-b-0 cursor-pointer transition-colors {selectedRow === i ? 'bg-primary/10 ring-1 ring-inset ring-primary/30' : 'hover:bg-accent/50'}"
							onclick={() => navigateToTask(task.id)}
							role="link"
							tabindex="0"
							data-kb-selected={selectedRow === i}
							onkeydown={(e) => {
								if (e.key === 'Enter' || e.key === ' ') {
									e.preventDefault();
									navigateToTask(task.id);
								}
							}}
						>
								<!-- ID -->
								<td class="px-3 py-2.5 font-mono text-xs text-muted-foreground truncate max-w-[140px]" title={task.id}>
									{task.id}
								</td>

								<!-- Title -->
								<td class="px-3 py-2.5 truncate max-w-[300px]" title={task.title}>
									{task.title}
								</td>

								<!-- Status -->
								<td class="px-3 py-2.5">
									<span
										class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {STATUS_COLORS[task.status] ?? ''}"
									>
										{task.status}
									</span>
								</td>

								<!-- Type -->
								<td class="px-3 py-2.5">
									<span
										class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {TYPE_COLORS[task.type] ?? ''}"
									>
										{task.type}
									</span>
								</td>

								<!-- Station -->
								<td class="px-3 py-2.5 text-muted-foreground truncate max-w-[120px]">
									{getStationName(task.stationId)}
								</td>

								<!-- Assignee -->
								<td class="px-3 py-2.5 text-muted-foreground truncate max-w-[120px]">
									{getAssigneeName(task.assignedToId)}
								</td>

								<!-- Priority -->
								<td class="px-3 py-2.5 {PRIORITY_COLORS[task.priority] ?? ''}">
									P{task.priority}
								</td>

								<!-- Quantity -->
								<td class="px-3 py-2.5 text-muted-foreground">
									{#if task.quantity > 1}
										{task.quantity}
									{/if}
								</td>

								<!-- Created -->
								<td class="px-3 py-2.5 text-xs text-muted-foreground" title={new Date(task.createdAt).toLocaleString()}>
									{formatTimeAgo(task.createdAt)}
								</td>

								<!-- Due Date -->
								<td class="px-3 py-2.5 text-xs text-muted-foreground" title={task.dueDate ? new Date(task.dueDate).toLocaleString() : ''}>
									{formatTimeAgo(task.dueDate)}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			<!-- Load more -->
			{#if hasMore}
				<div class="mt-4 flex justify-center">
					<Button variant="outline" size="sm" onclick={loadMore} disabled={isLoadingMore}>
						{#if isLoadingMore}
							<RefreshCw class="size-4 animate-spin mr-1.5" />
							Loading...
						{:else}
							Load more ({totalCount - allTasks.length} remaining)
						{/if}
					</Button>
				</div>
			{/if}
		{/if}
	</div>
</div>
