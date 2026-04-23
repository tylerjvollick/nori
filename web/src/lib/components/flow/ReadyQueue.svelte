<script lang="ts">
	import type { TaskResponse } from '$lib/types/task';
	import { Badge } from '$lib/components/ui/badge';
	import { Clock, User, Inbox } from '@lucide/svelte';
	import { formatTimeSpent } from '$lib/utils/time';
	import TaskActions from './TaskActions.svelte';

	interface Props {
		/** Ready tasks to display. Sorted by priority then createdAt internally. */
		tasks: TaskResponse[];
		/** Map of station ID -> station name for display. */
		stationMap: Map<string, string>;
		/** Whether data is currently loading. */
		isLoading?: boolean;
		/** Called when a task row is clicked. */
		onclick?: (task: TaskResponse) => void;
		/** Called after a successful task action (start, skip, etc.). */
		onaction?: (updated: TaskResponse) => void;
	}

	let {
		tasks,
		stationMap,
		isLoading = false,
		onclick,
		onaction,
	}: Props = $props();

	// Sort by priority (lower = higher priority), then by createdAt (oldest first)
	let sortedTasks = $derived(
		[...tasks].sort((a, b) => {
			if (a.priority !== b.priority) return a.priority - b.priority;
			return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
		}),
	);

	// --- Helpers ---

	function priorityBadge(priority: number): {
		label: string;
		variant: 'default' | 'secondary' | 'destructive' | 'outline';
		class: string;
	} {
		switch (priority) {
			case 0:
				return { label: 'P0', variant: 'destructive', class: '' };
			case 1:
				return { label: 'P1', variant: 'default', class: 'bg-orange-600 text-white border-transparent' };
			case 2:
				return { label: 'P2', variant: 'secondary', class: '' };
			case 3:
				return { label: 'P3', variant: 'outline', class: '' };
			default:
				return { label: `P${priority}`, variant: 'outline', class: '' };
		}
	}

	function getStationName(stationId: string | null | undefined): string | null {
		if (!stationId) return null;
		return stationMap.get(stationId) ?? stationId.slice(0, 8);
	}

	function handleRowClick(task: TaskResponse): void {
		onclick?.(task);
	}

	function handleRowKeydown(e: KeyboardEvent, task: TaskResponse): void {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onclick?.(task);
		}
	}
</script>

<div class="flex flex-col gap-1">
	{#if isLoading}
		<!-- Loading skeletons -->
		{#each Array(5) as _}
			<div class="animate-pulse rounded-lg border border-border bg-muted/30 p-3">
				<div class="h-4 w-3/4 rounded bg-muted"></div>
				<div class="mt-1.5 h-3 w-1/3 rounded bg-muted"></div>
				<div class="mt-2 flex gap-1.5">
					<div class="h-5 w-8 rounded bg-muted"></div>
					<div class="h-5 w-16 rounded bg-muted"></div>
				</div>
			</div>
		{/each}
	{:else if sortedTasks.length === 0}
		<!-- Empty state -->
		<div class="flex flex-col items-center justify-center py-12 text-muted-foreground">
			<Inbox class="size-10 mb-2 opacity-50" />
			<p class="text-sm font-medium">No ready tasks</p>
			<p class="text-xs mt-1">Tasks will appear here when their dependencies are resolved.</p>
		</div>
	{:else}
		{#each sortedTasks as task (task.id)}
			{@const pBadge = priorityBadge(task.priority)}
			{@const station = getStationName(task.stationId)}
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				class="group flex items-center gap-3 rounded-lg border border-border bg-card p-3 hover:border-primary/40 hover:shadow-sm transition-all cursor-pointer"
				role="button"
				tabindex="0"
				onclick={() => handleRowClick(task)}
				onkeydown={(e) => handleRowKeydown(e, task)}
			>
				<!-- Main content -->
				<div class="flex-1 min-w-0">
					<!-- Title + ID -->
					<h4 class="text-sm font-medium text-foreground group-hover:text-primary transition-colors truncate">
						{task.title}
					</h4>
					<p class="mt-0.5 text-xs font-mono text-muted-foreground truncate">{task.id}</p>

					<!-- Badges -->
					<div class="mt-1.5 flex flex-wrap items-center gap-1.5">
						<Badge variant={pBadge.variant} class="text-[10px] px-1.5 py-0 h-5 {pBadge.class}">
							{pBadge.label}
						</Badge>

						{#if station}
							<Badge variant="outline" class="text-[10px] px-1.5 py-0 h-5">
								{station}
							</Badge>
						{/if}

						{#if task.quantity > 1}
							<Badge variant="outline" class="text-[10px] px-1.5 py-0 h-5">
								x{task.quantity}
							</Badge>
						{/if}
					</div>
				</div>

				<!-- Right side: assignee, actions, time -->
				<div class="flex items-center gap-2 shrink-0">
					{#if task.assignedToId}
						<span
							class="inline-flex items-center justify-center size-5 rounded-full bg-primary/10 text-primary text-[10px] font-medium"
							title={task.assignedToId}
						>
							<User class="size-3" />
						</span>
					{/if}

					<TaskActions {task} layout="compact" {onaction} />

					<span class="flex items-center gap-0.5 text-[11px] text-muted-foreground whitespace-nowrap">
						<Clock class="size-3" />
						{formatTimeSpent(task.actualTimeSeconds, task.status, task.startedAt)}
					</span>
				</div>
			</div>
		{/each}
	{/if}
</div>
