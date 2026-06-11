<script lang="ts">
	import { page } from '$app/stores';
	import type { TaskResponse } from '$lib/types/task';
	import { Badge } from '$lib/components/ui/badge';
	import { Clock, Briefcase } from '@lucide/svelte';
	import { formatDuration } from '$lib/utils/time';

	/** A job with aggregate status/time computed from its children. */
	interface JobWithAggregate extends TaskResponse {
		aggregateStatus: 'done' | 'in_progress' | 'open' | 'backlog';
		totalTimeSeconds: number;
		childCount: number;
		doneChildCount: number;
	}

	interface Props {
		job: JobWithAggregate;
		stationMap: Map<string, string>;
	}

	let { job, stationMap }: Props = $props();

	let slug = $derived($page.params.slug);

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

	function aggregateStatusLabel(status: 'done' | 'in_progress' | 'open' | 'backlog'): { label: string; class: string } {
		switch (status) {
			case 'done':
				return { label: 'Done', class: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' };
			case 'in_progress':
				return { label: 'In Progress', class: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400' };
			case 'open':
				return { label: 'Ready', class: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400' };
			case 'backlog':
				return { label: 'Backlog', class: 'bg-slate-100 text-slate-600 dark:bg-slate-800/60 dark:text-slate-300' };
		}
	}

	let pBadge = $derived(priorityBadge(job.priority));
	let station = $derived(getStationName(job.stationId));
	let statusInfo = $derived(aggregateStatusLabel(job.aggregateStatus));
	let timeDisplay = $derived(formatDuration(job.totalTimeSeconds));
	let progressPct = $derived(
		job.childCount > 0 ? Math.round((job.doneChildCount / job.childCount) * 100) : 0,
	);
</script>

<a
	href="/spaces/{slug}/{job.id}"
	class="group block rounded-lg border border-border bg-card p-3 shadow-sm hover:border-primary/40 hover:shadow-md transition-all"
>
	<!-- Title -->
	<div class="flex items-center gap-1.5">
		<Briefcase class="size-3.5 text-muted-foreground shrink-0" />
		<h4 class="text-sm font-medium text-foreground group-hover:text-primary transition-colors truncate">
			{job.title}
		</h4>
	</div>

	<!-- Hierarchical ID -->
	<p class="mt-0.5 text-xs font-mono text-muted-foreground truncate">{job.id}</p>

	<!-- Badges row -->
	<div class="mt-2 flex flex-wrap items-center gap-1.5">
		<!-- Priority -->
		<Badge variant={pBadge.variant} class="text-[10px] px-1.5 py-0 h-5 {pBadge.class}">
			{pBadge.label}
		</Badge>

		<!-- Aggregate status -->
		<span
			data-testid="job-status-badge"
			class="inline-flex items-center rounded-full px-1.5 py-0 h-5 text-[10px] font-medium {statusInfo.class}"
		>
			{statusInfo.label}
		</span>

		<!-- Station -->
		{#if station}
			<Badge variant="outline" class="text-[10px] px-1.5 py-0 h-5">
				{station}
			</Badge>
		{/if}

		<!-- Quantity -->
		{#if job.quantity > 1}
			<Badge variant="outline" class="text-[10px] px-1.5 py-0 h-5">
				x{job.quantity}
			</Badge>
		{/if}
	</div>

	<!-- Progress bar (only shown when job has children) -->
	{#if job.childCount > 0}
		<div class="mt-2">
			<div class="flex items-center justify-between text-[10px] text-muted-foreground mb-1">
				<span>{job.doneChildCount}/{job.childCount} tasks</span>
				<span>{progressPct}%</span>
			</div>
			<div class="h-1.5 w-full rounded-full bg-muted overflow-hidden">
				<div
					class="h-full rounded-full transition-all duration-300 {job.aggregateStatus === 'done'
						? 'bg-green-500'
						: 'bg-primary'}"
					style="width: {progressPct}%"
				></div>
			</div>
		</div>
	{/if}

	<!-- Footer: total time -->
	<div class="mt-2 flex items-center justify-end text-[11px] text-muted-foreground">
		<span class="flex items-center gap-0.5">
			<Clock class="size-3" />
			{timeDisplay}
		</span>
	</div>
</a>
