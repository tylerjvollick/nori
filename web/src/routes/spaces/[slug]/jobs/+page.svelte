<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { jobApi } from '$lib/api/job';
	import type { TaskTreeResponse } from '$lib/api/task';
	import { taskApi } from '$lib/api/task';
	import { stationApi } from '$lib/api/station';
	import type { TaskResponse } from '$lib/types/task';
	import type { StationResponse } from '$lib/types/station';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Select from '$lib/components/ui/select';
	import * as Alert from '$lib/components/ui/alert';
	import {
		RefreshCw,
		CircleAlert,
		LayoutGrid,
		List,
		ArrowUp,
		ArrowDown,
		ArrowUpDown,
		Inbox,
	} from '@lucide/svelte';
	import KanbanColumn from '$lib/components/flow/KanbanColumn.svelte';
	import JobCard from '$lib/components/flow/JobCard.svelte';
	import { isEditableTarget } from '$lib/utils/keyboard.svelte';
	import { formatDuration } from '$lib/utils/time';
	import { onDestroy } from 'svelte';

	let space = $derived($page.data.space!);
	let spaceId = $derived(space.id);

	const POLL_INTERVAL_MS = 30_000;
	const DONE_LIMIT = 20;

	// ---- View mode ----

	type ViewMode = 'board' | 'list';

	const VIEW_MODES: { value: ViewMode; label: string; icon: typeof LayoutGrid }[] = [
		{ value: 'board', label: 'Board', icon: LayoutGrid },
		{ value: 'list', label: 'List', icon: List },
	];

	let currentView = $derived<ViewMode>(
		(($page.url.searchParams.get('view') as ViewMode) || 'board') as ViewMode,
	);

	function setView(mode: ViewMode): void {
		const url = new URL($page.url);
		if (mode === 'board') {
			url.searchParams.delete('view');
		} else {
			url.searchParams.set('view', mode);
		}
		goto(url.toString(), { replaceState: true, keepFocus: true, noScroll: true });
	}

	// ---- List view sort state ----

	type SortKey = 'id' | 'title' | 'status' | 'progress' | 'priority' | 'station' | 'time' | 'dueDate';
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
		{ key: 'progress', label: 'Progress', class: 'w-[110px]' },
		{ key: 'priority', label: 'Priority', class: 'w-[80px]' },
		{ key: 'station', label: 'Station', class: 'w-[120px]' },
		{ key: 'time', label: 'Time', class: 'w-[100px]' },
		{ key: 'dueDate', label: 'Due', class: 'w-[100px]' },
	];

	let sortKey = $state<SortKey>('priority');
	let sortDir = $state<SortDir>('asc');

	function handleSort(key: SortKey): void {
		if (sortKey === key) {
			sortDir = sortDir === 'asc' ? 'desc' : 'asc';
		} else {
			sortKey = key;
			sortDir = 'asc';
		}
	}

	// ---- Job aggregate type ----

	interface JobWithAggregate extends TaskResponse {
		aggregateStatus: 'done' | 'active' | 'open';
		totalTimeSeconds: number;
		childCount: number;
		doneChildCount: number;
	}

	// ---- State ----

	let allJobs = $state<JobWithAggregate[]>([]);
	let readyJobs = $state<JobWithAggregate[]>([]);
	let inProgressJobs = $state<JobWithAggregate[]>([]);
	let doneJobs = $state<JobWithAggregate[]>([]);

	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let isRefreshing = $state(false);
	let lastRefreshed = $state<Date | null>(null);
	let pollTimer: ReturnType<typeof setInterval> | null = null;

	let stationMap = $state<Map<string, string>>(new Map());
	let stations = $state<StationResponse[]>([]);

	// ---- Create Job Dialog ----
	let showCreateDialog = $state(false);
	let newJobTitle = $state('');
	let newJobDescription = $state('');
	let newJobPriority = $state('2');
	let newJobStationId = $state('');
	let isCreating = $state(false);
	let createError = $state('');

	const PRIORITY_OPTIONS = [
		{ value: '0', label: 'P0 – Critical' },
		{ value: '1', label: 'P1 – High' },
		{ value: '2', label: 'P2 – Medium' },
		{ value: '3', label: 'P3 – Low' },
		{ value: '4', label: 'P4 – Backlog' },
	];

	async function handleCreate(): Promise<void> {
		if (!newJobTitle.trim() || isCreating) return;
		isCreating = true;
		createError = '';
		try {
			const job = await taskApi.createTask(spaceId, {
				title: newJobTitle.trim(),
				description: newJobDescription.trim() || undefined,
				type: 'job',
				priority: parseInt(newJobPriority, 10),
				stationId: newJobStationId || undefined,
			});
			showCreateDialog = false;
			const slug = $page.params.slug;
			goto(`/spaces/${slug}/${job.id}`);
		} catch (error) {
			createError = error instanceof Error ? error.message : 'Failed to create job';
		} finally {
			isCreating = false;
		}
	}

	function resetCreateDialog(): void {
		newJobTitle = '';
		newJobDescription = '';
		newJobPriority = '2';
		newJobStationId = '';
		createError = '';
	}

	// ---- Helpers ----

	function computeAggregateStatus(children: TaskResponse[]): 'done' | 'active' | 'open' {
		if (children.length === 0) return 'open';
		const allDone = children.every(
			(c) => c.status === 'done' || c.status === 'skipped' || c.status === 'cancelled',
		);
		if (allDone) return 'done';
		const anyActive = children.some(
			(c) => c.status === 'active' || c.status === 'paused',
		);
		if (anyActive) return 'active';
		return 'open';
	}

	function sumTreeTime(node: TaskTreeResponse): number {
		let total = node.actualTimeSeconds;
		for (const child of node.children) {
			total += sumTreeTime(child);
		}
		return total;
	}

	function getDirectChildren(tree: TaskTreeResponse): TaskResponse[] {
		return tree.children.map(({ children: _, ...rest }) => rest as TaskResponse);
	}

	function statusToAggregate(status: string): 'done' | 'active' | 'open' {
		if (status === 'done' || status === 'skipped' || status === 'cancelled') return 'done';
		if (status === 'active' || status === 'paused') return 'active';
		return 'open';
	}

	// ---- List view helpers ----

	const AGGREGATE_STATUS_COLORS: Record<string, string> = {
		open: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
		active: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
		done: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
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

	// ---- Sorted jobs for list view ----

	let sortedJobs = $derived.by(() => {
		const jobs = [...allJobs];
		const dir = sortDir === 'asc' ? 1 : -1;

		jobs.sort((a, b) => {
			let cmp = 0;
			switch (sortKey) {
				case 'id':
					cmp = a.id.localeCompare(b.id);
					break;
				case 'title':
					cmp = a.title.localeCompare(b.title);
					break;
				case 'status':
					cmp = a.aggregateStatus.localeCompare(b.aggregateStatus);
					break;
				case 'progress': {
					const pA = a.childCount > 0 ? a.doneChildCount / a.childCount : 0;
					const pB = b.childCount > 0 ? b.doneChildCount / b.childCount : 0;
					cmp = pA - pB;
					break;
				}
				case 'priority':
					cmp = a.priority - b.priority;
					break;
				case 'station': {
					const sA = a.stationId ? (stationMap.get(a.stationId) ?? a.stationId) : '';
					const sB = b.stationId ? (stationMap.get(b.stationId) ?? b.stationId) : '';
					cmp = sA.localeCompare(sB);
					break;
				}
				case 'time':
					cmp = a.totalTimeSeconds - b.totalTimeSeconds;
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

		return jobs;
	});

	// ---- Data fetching ----

	async function fetchStations(): Promise<void> {
		if (!spaceId) return;
		try {
			const fetched: StationResponse[] = await stationApi.listStations(spaceId);
			stations = fetched;
			const map = new Map<string, string>();
			for (const s of fetched) {
				map.set(s.id, s.name);
			}
			stationMap = map;
		} catch {
			// gracefully degrade
		}
	}

	async function fetchJobs(opts?: { silent?: boolean }): Promise<void> {
		if (!spaceId) return;
		if (!opts?.silent) {
			isLoading = true;
		}
		isRefreshing = true;
		error = null;

		try {
			const jobsResult = await jobApi.listJobs(spaceId, { limit: 200 });
			const rootJobs = jobsResult.items;

			const jobTrees = await Promise.all(
				rootJobs.map(async (job) => {
					try {
						const tree = await jobApi.getJobTasks(spaceId, job.id);
						return { job, tree };
					} catch {
						return { job, tree: null };
					}
				}),
			);

			const jobsWithAggregates: JobWithAggregate[] = jobTrees.map(({ job, tree }) => {
				if (!tree || tree.children.length === 0) {
					return {
						...job,
						aggregateStatus: statusToAggregate(job.status),
						totalTimeSeconds: job.actualTimeSeconds,
						childCount: 0,
						doneChildCount: 0,
					};
				}

				const directChildren = getDirectChildren(tree);
				const aggregateStatus = computeAggregateStatus(directChildren);
				const totalTimeSeconds = sumTreeTime(tree);
				const doneChildCount = directChildren.filter(
					(c) => c.status === 'done' || c.status === 'skipped' || c.status === 'cancelled',
				).length;

				return {
					...job,
					aggregateStatus,
					totalTimeSeconds,
					childCount: directChildren.length,
					doneChildCount,
				};
			});

			allJobs = jobsWithAggregates;
			readyJobs = jobsWithAggregates.filter((j) => j.aggregateStatus === 'open');
			inProgressJobs = jobsWithAggregates.filter((j) => j.aggregateStatus === 'active');
			const allDoneJobs = jobsWithAggregates.filter((j) => j.aggregateStatus === 'done');
			allDoneJobs.sort(
				(a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime(),
			);
			doneJobs = allDoneJobs.slice(0, DONE_LIMIT);

			lastRefreshed = new Date();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load jobs';
		} finally {
			isLoading = false;
			isRefreshing = false;
		}
	}

	function formatLastRefreshed(date: Date | null): string {
		if (!date) return '';
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	function startPolling(): void {
		stopPolling();
		pollTimer = setInterval(() => fetchJobs({ silent: true }), POLL_INTERVAL_MS);
	}

	function stopPolling(): void {
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	let initialized = $state(false);

	$effect(() => {
		if (!spaceId) return;
		if (!initialized) {
			initialized = true;
			Promise.all([fetchJobs(), fetchStations()]).then(() => startPolling());
		}
	});

	onDestroy(() => {
		stopPolling();
	});

	// ---- Keyboard shortcuts ----

	function handleKeydown(e: KeyboardEvent): void {
		if (isEditableTarget(e)) return;

		switch (e.key) {
			case 'b':
				setView('board');
				break;
			case 'l':
				setView('list');
				break;
		}
	}

	// ---- Row navigation ----

	let slug = $derived($page.params.slug);

	function navigateToJob(jobId: string): void {
		goto(`/spaces/${slug}/${jobId}`);
	}
</script>

<svelte:head>
	<title>{space.name} – Jobs - Nori</title>
</svelte:head>

<svelte:window onkeydown={handleKeydown} />

<div class="flex h-full flex-col overflow-hidden">
	<!-- Header toolbar -->
	<div class="flex-shrink-0 border-b bg-background px-4 py-2">
		<div class="flex items-center justify-between gap-4">
			<!-- View mode switcher (segmented control) -->
			<div class="flex items-center rounded-lg border bg-muted/50 p-0.5">
				{#each VIEW_MODES as mode (mode.value)}
					<Button
						variant={currentView === mode.value ? 'secondary' : 'ghost'}
						size="sm"
						class="gap-1.5 rounded-md px-3 {currentView === mode.value
							? 'bg-background shadow-sm'
							: 'hover:bg-transparent hover:text-foreground'}"
						onclick={() => setView(mode.value)}
					>
						<mode.icon class="size-4" />
						{mode.label}
					</Button>
				{/each}
			</div>

			<!-- Right side actions -->
			<div class="flex items-center gap-3">
				{#if lastRefreshed}
					<span class="text-xs text-muted-foreground">
						Updated {formatLastRefreshed(lastRefreshed)}
					</span>
				{/if}
				<Button
					variant="outline"
					size="sm"
					onclick={() => fetchJobs({ silent: true })}
					disabled={isRefreshing}
				>
					<RefreshCw class="size-4 {isRefreshing ? 'animate-spin' : ''}" />
					<span class="ml-1.5">Refresh</span>
				</Button>
				<Button size="sm" onclick={() => (showCreateDialog = true)}>New Job</Button>
			</div>
		</div>
	</div>

	<!-- Error banner -->
	{#if error}
		<div
			class="mx-4 mt-2 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm"
		>
			<CircleAlert class="size-4 text-destructive shrink-0" />
			<span class="text-destructive">{error}</span>
			<Button variant="outline" size="sm" class="ml-auto" onclick={() => fetchJobs()}>
				Retry
			</Button>
		</div>
	{/if}

	<!-- Board or list view -->
	<div class="flex-1 overflow-hidden">
		{#if currentView === 'board'}
			<!-- Kanban columns -->
			<div class="flex min-h-0 h-full gap-4 overflow-x-auto px-4 pb-4 pt-4">
				<KanbanColumn
					title="Ready"
					count={readyJobs.length}
					colorClass="bg-blue-500"
					{isLoading}
					emptyLabel="No jobs"
				>
					{#each readyJobs as job (job.id)}
						<JobCard {job} {stationMap} />
					{/each}
				</KanbanColumn>

				<KanbanColumn
					title="In Progress"
					count={inProgressJobs.length}
					colorClass="bg-yellow-500"
					{isLoading}
					emptyLabel="No jobs"
				>
					{#each inProgressJobs as job (job.id)}
						<JobCard {job} {stationMap} />
					{/each}
				</KanbanColumn>

				<KanbanColumn
					title="Done"
					count={doneJobs.length}
					colorClass="bg-green-500"
					{isLoading}
					emptyLabel="No jobs"
				>
					{#each doneJobs as job (job.id)}
						<JobCard {job} {stationMap} />
					{/each}
				</KanbanColumn>
			</div>
		{:else if currentView === 'list'}
			<!-- List view -->
			<div class="flex-1 overflow-auto px-4 pb-4 pt-4 h-full">
				{#if isLoading}
					<div class="rounded-lg border">
						<div class="border-b bg-muted/50 px-4 py-3">
							<Skeleton class="h-4 w-full" />
						</div>
						{#each Array(8) as _}
							<div class="flex items-center gap-4 border-b px-4 py-3 last:border-b-0">
								<Skeleton class="h-4 w-[120px]" />
								<Skeleton class="h-4 flex-1" />
								<Skeleton class="h-5 w-[70px] rounded-full" />
								<Skeleton class="h-4 w-[80px]" />
								<Skeleton class="h-4 w-[50px]" />
								<Skeleton class="h-4 w-[100px]" />
								<Skeleton class="h-4 w-[80px]" />
								<Skeleton class="h-4 w-[80px]" />
							</div>
						{/each}
					</div>
				{:else if allJobs.length === 0}
					<div class="rounded-lg border border-border p-12 text-center">
						<Inbox class="mx-auto mb-4 size-12 text-muted-foreground" />
						<h3 class="mb-1 text-lg font-semibold text-foreground">No jobs found</h3>
						<p class="mx-auto max-w-md text-sm text-muted-foreground">
							There are no jobs yet. Click "New Job" to create one.
						</p>
					</div>
				{:else}
					<div class="rounded-lg border" data-testid="jobs-list-table">
						<table class="w-full text-sm">
							<thead>
								<tr class="border-b bg-muted/50">
									{#each COLUMNS as col (col.key)}
										<th
											class="px-3 py-2 text-left font-medium text-muted-foreground {col.class ?? ''}"
										>
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
								{#each sortedJobs as job (job.id)}
									<tr
										class="border-b last:border-b-0 cursor-pointer transition-colors hover:bg-accent/50"
										onclick={() => navigateToJob(job.id)}
										role="link"
										tabindex="0"
										onkeydown={(e) => {
											if (e.key === 'Enter' || e.key === ' ') {
												e.preventDefault();
												navigateToJob(job.id);
											}
										}}
									>
										<!-- ID -->
										<td
											class="px-3 py-2.5 font-mono text-xs text-muted-foreground truncate max-w-[140px]"
											title={job.id}
										>
											{job.id}
										</td>

										<!-- Title -->
										<td class="px-3 py-2.5 truncate max-w-[300px]" title={job.title}>
											{job.title}
										</td>

										<!-- Status (aggregate) -->
										<td class="px-3 py-2.5">
											<span
												class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {AGGREGATE_STATUS_COLORS[
													job.aggregateStatus
												] ?? ''}"
											>
												{job.aggregateStatus}
											</span>
										</td>

										<!-- Progress -->
										<td class="px-3 py-2.5">
											{#if job.childCount > 0}
												<div class="flex items-center gap-2">
													<div class="h-1.5 w-16 rounded-full bg-muted overflow-hidden">
														<div
															class="h-full rounded-full bg-green-500 transition-all"
															style="width: {Math.round((job.doneChildCount / job.childCount) * 100)}%"
														></div>
													</div>
													<span class="text-xs text-muted-foreground whitespace-nowrap">
														{job.doneChildCount}/{job.childCount}
													</span>
												</div>
											{:else}
												<span class="text-xs text-muted-foreground">&mdash;</span>
											{/if}
										</td>

										<!-- Priority -->
										<td class="px-3 py-2.5 {PRIORITY_COLORS[job.priority] ?? ''}">
											P{job.priority}
										</td>

										<!-- Station -->
										<td class="px-3 py-2.5 text-muted-foreground truncate max-w-[120px]">
											{getStationName(job.stationId)}
										</td>

										<!-- Time -->
										<td class="px-3 py-2.5 text-xs text-muted-foreground">
											{formatDuration(job.totalTimeSeconds)}
										</td>

										<!-- Due Date -->
										<td
											class="px-3 py-2.5 text-xs text-muted-foreground"
											title={job.dueDate ? new Date(job.dueDate).toLocaleString() : ''}
										>
											{formatTimeAgo(job.dueDate)}
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>

<!-- Create Job Dialog -->
<Dialog.Root
	bind:open={showCreateDialog}
	onOpenChange={(open) => {
		if (!open) resetCreateDialog();
	}}
>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Create New Job</Dialog.Title>
			<Dialog.Description>Add a new job to track work through stations.</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleCreate();
			}}
		>
			<div class="grid gap-4 py-2">
				<div class="grid gap-2">
					<Label for="job-title">Title</Label>
					<Input
						id="job-title"
						type="text"
						bind:value={newJobTitle}
						placeholder="e.g., Oak Dining Table, Custom Bookshelf"
						disabled={isCreating}
					/>
				</div>
				<div class="grid gap-2">
					<Label for="job-description">Description (optional)</Label>
					<Input
						id="job-description"
						type="text"
						bind:value={newJobDescription}
						placeholder="Brief description of this job"
						disabled={isCreating}
					/>
				</div>
				<div class="grid gap-2">
					<Label>Priority</Label>
					<Select.Root type="single" bind:value={newJobPriority}>
						<Select.Trigger disabled={isCreating}>
							{PRIORITY_OPTIONS.find((p) => p.value === newJobPriority)?.label ?? 'P2 – Medium'}
						</Select.Trigger>
						<Select.Content>
							{#each PRIORITY_OPTIONS as opt (opt.value)}
								<Select.Item value={opt.value} label={opt.label}>{opt.label}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>
				<div class="grid gap-2">
					<Label>Station (optional)</Label>
					<Select.Root type="single" bind:value={newJobStationId}>
						<Select.Trigger disabled={isCreating}>
							{newJobStationId
								? (stations.find((s) => s.id === newJobStationId)?.name ?? 'Station')
								: 'None'}
						</Select.Trigger>
						<Select.Content>
							<Select.Item value="" label="None">None</Select.Item>
							{#each stations as station (station.id)}
								<Select.Item value={station.id} label={station.name}>{station.name}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>
				{#if createError}
					<Alert.Root variant="destructive">
						<CircleAlert class="size-4" />
						<Alert.Description>{createError}</Alert.Description>
					</Alert.Root>
				{/if}
			</div>
			<Dialog.Footer class="pt-2">
				<Button
					type="button"
					variant="outline"
					onclick={() => (showCreateDialog = false)}
					disabled={isCreating}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={!newJobTitle.trim() || isCreating}>
					{isCreating ? 'Creating...' : 'Create Job'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
