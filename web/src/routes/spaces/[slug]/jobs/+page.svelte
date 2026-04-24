<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { jobApi } from '$lib/api/job';
	import type { TaskTreeResponse } from '$lib/api/task';
	import { taskApi } from '$lib/api/task';
	import { stationApi } from '$lib/api/station';
	import { spaceStore } from '$lib/stores/space';
	import type { TaskResponse } from '$lib/types/task';
	import type { StationResponse } from '$lib/types/station';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Select from '$lib/components/ui/select';
	import * as Alert from '$lib/components/ui/alert';
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
	<div class="flex-shrink-0 flex items-center justify-between px-4 py-2">
		<Button size="sm" onclick={() => (showCreateDialog = true)}>New Job</Button>
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
	<div class="flex min-h-0 flex-1 gap-4 overflow-x-auto px-4 pb-4">
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
