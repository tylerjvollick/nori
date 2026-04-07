<script lang="ts">
	import { page } from '$app/stores';
	import { onMount, onDestroy } from 'svelte';
	import { apiClient } from '$lib/api/client';
	import { taskApi } from '$lib/api/task';
	import type { TaskResponse } from '$lib/types/task';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { RefreshCw, Inbox, AlertCircle, Clock, ChevronRight, UserPlus, Loader2 } from 'lucide-svelte';
	import BoardView from '$lib/components/flow/BoardView.svelte';

	/** Matches the backend TaskResponse DTO. */
	interface ReadyTask {
		id: string;
		title: string;
		description?: string | null;
		stationId?: string | null;
		priority: number;
		type: string;
		status: string;
		dueDate?: string | null;
		createdAt: string;
	}

	interface ReadyResponse {
		items: ReadyTask[];
		total: number;
	}

	/** Cached station id → name map. */
	interface StationInfo {
		id: string;
		name: string;
	}

	const POLL_INTERVAL_MS = 30_000;

	// ---- View mode ----
	type ViewMode = 'board' | 'graph' | 'list';
	let currentView = $derived<ViewMode>(
		(($page.url.searchParams.get('view') as ViewMode) || 'board') as ViewMode,
	);

	// ---- Ready queue state (list view) ----
	let tasks = $state<ReadyTask[]>([]);
	let total = $state(0);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let lastRefreshed = $state<Date | null>(null);
	let isRefreshing = $state(false);
	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let stationMap = $state<Map<string, string>>(new Map());

	async function fetchStations(): Promise<void> {
		try {
			const resp = await apiClient.get<{ items: StationInfo[] }>('/api/v1/stations');
			const map = new Map<string, string>();
			for (const s of resp.items) {
				map.set(s.id, s.name);
			}
			stationMap = map;
		} catch {
			// Stations endpoint may not exist yet — gracefully degrade.
		}
	}

	async function fetchReadyTasks(opts?: { silent?: boolean }): Promise<void> {
		if (!opts?.silent) {
			isLoading = true;
		}
		isRefreshing = true;
		error = null;

		try {
			const resp = await apiClient.get<ReadyResponse>('/api/v1/tasks/ready');
			tasks = resp.items ?? [];
			total = resp.total ?? 0;
			lastRefreshed = new Date();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load ready tasks';
		} finally {
			isLoading = false;
			isRefreshing = false;
		}
	}

	function startPolling(): void {
		stopPolling();
		pollTimer = setInterval(() => fetchReadyTasks({ silent: true }), POLL_INTERVAL_MS);
	}

	function stopPolling(): void {
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	function handleManualRefresh(): void {
		fetchReadyTasks({ silent: true });
	}

	onMount(async () => {
		// Only start list-view polling if we're showing the list view
		if (currentView === 'list') {
			await Promise.all([fetchReadyTasks(), fetchStations()]);
			startPolling();
		}
	});

	onDestroy(() => {
		stopPolling();
	});

	// Start/stop list polling when view changes
	$effect(() => {
		if (currentView === 'list') {
			if (!lastRefreshed) {
				// First time switching to list view, fetch data
				Promise.all([fetchReadyTasks(), fetchStations()]);
			}
			startPolling();
		} else {
			stopPolling();
		}
	});

	/** Priority label & styling. Lower number = higher priority. */
	function priorityBadge(priority: number): { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline'; class: string } {
		switch (priority) {
			case 0:
				return { label: 'Critical', variant: 'destructive', class: '' };
			case 1:
				return { label: 'High', variant: 'default', class: 'bg-orange-600 text-white border-transparent' };
			case 2:
				return { label: 'Medium', variant: 'secondary', class: '' };
			case 3:
				return { label: 'Low', variant: 'outline', class: '' };
			default:
				return { label: `P${priority}`, variant: 'outline', class: '' };
		}
	}

	function getStationName(stationId: string | null | undefined): string | null {
		if (!stationId) return null;
		return stationMap.get(stationId) ?? stationId.slice(0, 8);
	}

	function formatTimeAgo(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60_000);
		if (diffMins < 1) return 'Just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		const diffHours = Math.floor(diffMins / 60);
		if (diffHours < 24) return `${diffHours}h ago`;
		const diffDays = Math.floor(diffHours / 24);
		return `${diffDays}d ago`;
	}

	function formatLastRefreshed(date: Date | null): string {
		if (!date) return '';
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	// ---- Claim action for ready queue ----
	let claimingTaskId = $state<string | null>(null);

	async function handleClaimTask(e: MouseEvent, taskId: string): Promise<void> {
		e.preventDefault();
		e.stopPropagation();
		if (claimingTaskId) return;
		claimingTaskId = taskId;
		try {
			await taskApi.claimTask(taskId);
			// Remove the claimed task from the list and refresh
			tasks = tasks.filter((t) => t.id !== taskId);
			total = Math.max(0, total - 1);
			// Trigger a background refresh to sync full state
			fetchReadyTasks({ silent: true });
		} catch (err) {
			console.error(`Failed to claim task ${taskId}:`, err);
		} finally {
			claimingTaskId = null;
		}
	}
</script>

<svelte:head>
	<title>Flow - Nori</title>
</svelte:head>

{#if currentView === 'board'}
	<BoardView />
{:else if currentView === 'list'}
	<!-- Ready queue list view (original page content) -->
	<div class="flex-1 overflow-auto">
		<div class="max-w-4xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
			<!-- Header -->
			<div class="flex items-center justify-between mb-6">
				<div>
					<h1 class="text-2xl font-bold text-foreground">Ready Queue</h1>
					<p class="text-sm text-muted-foreground mt-1">
						Tasks ready for work — no blockers, no unmet dependencies.
					</p>
				</div>
				<div class="flex items-center gap-3">
					{#if lastRefreshed}
						<span class="text-xs text-muted-foreground">
							Updated {formatLastRefreshed(lastRefreshed)}
						</span>
					{/if}
					<Button
						variant="outline"
						size="sm"
						onclick={handleManualRefresh}
						disabled={isRefreshing}
					>
						<RefreshCw class="w-4 h-4 {isRefreshing ? 'animate-spin' : ''}" />
						<span class="ml-1.5">Refresh</span>
					</Button>
				</div>
			</div>

			<!-- Loading state -->
			{#if isLoading}
				<div class="space-y-3">
					{#each Array(5) as _}
						<div class="border border-border rounded-lg p-4">
							<div class="flex items-start justify-between">
								<div class="space-y-2 flex-1">
									<Skeleton class="h-5 w-2/3" />
									<Skeleton class="h-4 w-1/3" />
								</div>
								<Skeleton class="h-5 w-16 rounded-full" />
							</div>
							<div class="flex items-center gap-2 mt-3">
								<Skeleton class="h-5 w-20 rounded-full" />
								<Skeleton class="h-5 w-16 rounded-full" />
							</div>
						</div>
					{/each}
				</div>

			<!-- Error state -->
			{:else if error}
				<div class="border border-destructive/30 bg-destructive/5 rounded-lg p-8 text-center">
					<AlertCircle class="w-10 h-10 text-destructive mx-auto mb-3" />
					<h3 class="text-lg font-semibold text-foreground mb-1">Failed to load ready tasks</h3>
					<p class="text-sm text-muted-foreground mb-4">{error}</p>
					<Button variant="outline" size="sm" onclick={() => fetchReadyTasks()}>
						Try Again
					</Button>
				</div>

			<!-- Empty state -->
			{:else if tasks.length === 0}
				<div class="border border-border rounded-lg p-12 text-center">
					<Inbox class="w-12 h-12 text-muted-foreground mx-auto mb-4" />
					<h3 class="text-lg font-semibold text-foreground mb-1">No tasks ready</h3>
					<p class="text-sm text-muted-foreground max-w-md mx-auto">
						There are no unblocked tasks waiting for work right now. Tasks will appear here
						once their dependencies are resolved.
					</p>
				</div>

			<!-- Task list -->
			{:else}
				<div class="mb-3">
					<span class="text-sm text-muted-foreground">
						{total} task{total === 1 ? '' : 's'} ready
					</span>
				</div>
				<div class="space-y-2">
					{#each tasks as task (task.id)}
						{@const pBadge = priorityBadge(task.priority)}
						{@const station = getStationName(task.stationId)}
						<a
							href="/flow/tasks/{task.id}"
							class="group block border border-border rounded-lg p-4 hover:border-primary/40 hover:bg-accent/30 transition-colors"
						>
							<div class="flex items-start justify-between gap-4">
								<div class="flex-1 min-w-0">
									<h3 class="text-sm font-medium text-foreground group-hover:text-primary transition-colors truncate">
										{task.title}
									</h3>
									{#if task.description}
										<p class="text-sm text-muted-foreground mt-0.5 line-clamp-1">
											{task.description}
										</p>
									{/if}
									<div class="flex items-center gap-2 mt-2 flex-wrap">
										<Badge variant={pBadge.variant} class={pBadge.class}>
											{pBadge.label}
										</Badge>
										{#if station}
											<Badge variant="outline">
												{station}
											</Badge>
										{/if}
										<span class="text-xs text-muted-foreground flex items-center gap-1">
											<Clock class="w-3 h-3" />
											{formatTimeAgo(task.createdAt)}
										</span>
									</div>
								</div>
								<div class="flex items-center gap-2 shrink-0 mt-0.5">
								<Button
									variant="default"
									size="sm"
									disabled={claimingTaskId !== null}
									onclick={(e: MouseEvent) => handleClaimTask(e, task.id)}
								>
									{#if claimingTaskId === task.id}
										<Loader2 class="size-4 animate-spin" />
									{:else}
										<UserPlus class="size-4" />
									{/if}
									Claim
								</Button>
								<ChevronRight class="w-5 h-5 text-muted-foreground group-hover:text-primary transition-colors" />
							</div>
							</div>
						</a>
					{/each}
				</div>
			{/if}
		</div>
	</div>
{:else}
	<!-- Graph view placeholder -->
	<div class="flex flex-1 items-center justify-center text-muted-foreground">
		<p class="text-sm">Graph view coming soon</p>
	</div>
{/if}
