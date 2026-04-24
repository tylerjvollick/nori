<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { jobApi } from '$lib/api/job';
	import type { TaskTreeResponse } from '$lib/api/task';
	import { stationApi } from '$lib/api/station';
	import { spaceStore } from '$lib/stores/space';
	import type { TaskResponse } from '$lib/types/task';
	import type { StationResponse } from '$lib/types/station';
	import { Button } from '$lib/components/ui/button';
	import { RefreshCw, CircleAlert } from '@lucide/svelte';
	import KanbanColumn from '$lib/components/flow/KanbanColumn.svelte';
	import JobCard from '$lib/components/flow/JobCard.svelte';
	import { onDestroy } from 'svelte';

	let currentSpace = $derived($spaceStore.currentSpace);
	let spaceId = $derived(currentSpace?.id ?? '');

	const POLL_INTERVAL_MS = 30_000;
	const DONE_LIMIT = 20;

	// ---- Job aggregate type ----

	interface JobWithAggregate extends TaskResponse {
		aggregateStatus: 'done' | 'active' | 'open';
		totalTimeSeconds: number;
		childCount: number;
		doneChildCount: number;
	}

	// ---- State ----

	let readyJobs = $state<JobWithAggregate[]>([]);
	let inProgressJobs = $state<JobWithAggregate[]>([]);
	let doneJobs = $state<JobWithAggregate[]>([]);

	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let isRefreshing = $state(false);
	let lastRefreshed = $state<Date | null>(null);
	let pollTimer: ReturnType<typeof setInterval> | null = null;

	let stationMap = $state<Map<string, string>>(new Map());

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

	// ---- Data fetching ----

	async function fetchStations(): Promise<void> {
		if (!spaceId) return;
		try {
			const stations: StationResponse[] = await stationApi.listStations(spaceId);
			const map = new Map<string, string>();
			for (const s of stations) {
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
</script>

<svelte:head>
	<title>{currentSpace?.name ?? 'Space'} – Jobs - Nori</title>
</svelte:head>

<div class="flex h-full flex-col overflow-hidden">
	<!-- Header -->
	<div class="flex-shrink-0 flex items-center justify-end px-4 py-2">
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
		</div>
	</div>

	<!-- Error banner -->
	{#if error}
		<div
			class="mx-4 mb-2 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm"
		>
			<CircleAlert class="size-4 text-destructive shrink-0" />
			<span class="text-destructive">{error}</span>
			<Button variant="outline" size="sm" class="ml-auto" onclick={() => fetchJobs()}>
				Retry
			</Button>
		</div>
	{/if}

	<!-- Kanban columns -->
	<div class="flex flex-1 gap-4 overflow-x-auto px-4 pb-4">
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
</div>
