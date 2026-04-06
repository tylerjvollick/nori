<script lang="ts">
	import type { TaskTreeResponse, TaskDepsResponse } from '$lib/api/task';
	import type { TaskDep } from '$lib/types/task';
	import { Badge } from '$lib/components/ui/badge';
	import { Separator } from '$lib/components/ui/separator';
	import {
		Circle,
		CircleDot,
		CircleCheck,
		CirclePause,
		CircleX,
		CircleMinus,
		Clock,
		User,
		Calendar,
		Link as LinkIcon,
		ArrowRight,
		ArrowLeft,
	} from 'lucide-svelte';

	interface Props {
		task: TaskTreeResponse;
		stationMap: Map<string, string>;
		deps?: TaskDepsResponse | null;
	}

	let { task, stationMap, deps = null }: Props = $props();

	// --- Helpers ---

	type StatusConfig = {
		label: string;
		colorClass: string;
		bgClass: string;
	};

	function getStatusConfig(status: string): StatusConfig {
		switch (status) {
			case 'active':
				return { label: 'Active', colorClass: 'text-blue-500', bgClass: 'bg-blue-500/10' };
			case 'done':
				return { label: 'Done', colorClass: 'text-green-500', bgClass: 'bg-green-500/10' };
			case 'paused':
				return { label: 'Paused', colorClass: 'text-yellow-500', bgClass: 'bg-yellow-500/10' };
			case 'skipped':
				return { label: 'Skipped', colorClass: 'text-muted-foreground', bgClass: 'bg-muted' };
			case 'cancelled':
				return { label: 'Cancelled', colorClass: 'text-red-500', bgClass: 'bg-red-500/10' };
			default:
				return { label: 'Open', colorClass: 'text-muted-foreground', bgClass: 'bg-muted' };
		}
	}

	function priorityLabel(priority: number): string {
		switch (priority) {
			case 0: return 'Critical';
			case 1: return 'High';
			case 2: return 'Medium';
			case 3: return 'Low';
			default: return `P${priority}`;
		}
	}

	function priorityBadgeVariant(priority: number): 'default' | 'secondary' | 'destructive' | 'outline' {
		switch (priority) {
			case 0: return 'destructive';
			case 1: return 'default';
			case 2: return 'secondary';
			default: return 'outline';
		}
	}

	function typeLabel(type: string): string {
		switch (type) {
			case 'job': return 'Job';
			case 'task': return 'Task';
			case 'milestone': return 'Milestone';
			case 'gate': return 'Gate';
			default: return type;
		}
	}

	function getStationName(stationId: string | null | undefined): string | null {
		if (!stationId) return null;
		return stationMap.get(stationId) ?? stationId.slice(0, 8);
	}

	function formatDuration(seconds: number): string {
		if (seconds === 0) return '--';
		const h = Math.floor(seconds / 3600);
		const m = Math.floor((seconds % 3600) / 60);
		if (h > 0) return `${h}h ${m}m`;
		return `${m}m`;
	}

	function formatDate(dateStr: string | null | undefined): string {
		if (!dateStr) return '--';
		return new Date(dateStr).toLocaleDateString(undefined, {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
		});
	}

	function formatDateTime(dateStr: string | null | undefined): string {
		if (!dateStr) return '--';
		return new Date(dateStr).toLocaleString(undefined, {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
		});
	}

	/** Compute completion ratio from children. */
	function computeProgress(node: TaskTreeResponse): { done: number; total: number } {
		if (!node.children || node.children.length === 0) {
			return { done: 0, total: 0 };
		}
		let done = 0;
		for (const child of node.children) {
			if (child.status === 'done' || child.status === 'skipped') done++;
		}
		return { done, total: node.children.length };
	}

	let progress = $derived(computeProgress(task));
	let statusCfg = $derived(getStatusConfig(task.status));
	let station = $derived(getStationName(task.stationId));
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

<div class="p-6 space-y-5">
	<!-- Header: title, ID, type -->
	<div>
		<div class="flex items-center gap-2 mb-2">
			<Badge variant="outline" class="text-xs font-mono">
				{task.id}
			</Badge>
			<Badge variant="outline" class="text-xs">
				{typeLabel(task.type)}
			</Badge>
		</div>
		<h2 class="text-lg font-semibold text-foreground">{task.title}</h2>
		{#if task.description}
			<p class="text-sm text-muted-foreground mt-1">{task.description}</p>
		{/if}
	</div>

	<Separator />

	<!-- Core metadata -->
	<div class="space-y-3">
		<!-- Status -->
		<div class="flex items-center justify-between">
			<span class="text-sm text-muted-foreground">Status</span>
			<Badge class="{statusCfg.bgClass} {statusCfg.colorClass} border-transparent">
				{@render statusIcon(task.status, 'w-3 h-3 mr-1')}
				{statusCfg.label}
			</Badge>
		</div>

		<!-- Priority -->
		<div class="flex items-center justify-between">
			<span class="text-sm text-muted-foreground">Priority</span>
			<Badge variant={priorityBadgeVariant(task.priority)}>
				{priorityLabel(task.priority)}
			</Badge>
		</div>

		<!-- Station -->
		{#if station}
			<div class="flex items-center justify-between">
				<span class="text-sm text-muted-foreground">Station</span>
				<span class="text-sm text-foreground">{station}</span>
			</div>
		{/if}

		<!-- Assignee -->
		{#if task.assignedToId}
			<div class="flex items-center justify-between">
				<span class="text-sm text-muted-foreground">Assigned To</span>
				<span class="text-sm text-foreground font-mono flex items-center gap-1">
					<User class="w-3 h-3" />
					{task.assignedToId.slice(0, 8)}
				</span>
			</div>
		{/if}

		<!-- Quantity -->
		{#if task.quantity > 1}
			<div class="flex items-center justify-between">
				<span class="text-sm text-muted-foreground">Quantity</span>
				<span class="text-sm text-foreground">{task.quantity}</span>
			</div>
		{/if}

		<!-- Recipe link -->
		{#if task.recipeId}
			<div class="flex items-center justify-between">
				<span class="text-sm text-muted-foreground">Recipe</span>
				<span class="text-sm text-foreground font-mono flex items-center gap-1">
					<LinkIcon class="w-3 h-3" />
					{task.recipeId.slice(0, 8)}
				</span>
			</div>
		{/if}
	</div>

	<Separator />

	<!-- Time data -->
	<div class="space-y-3">
		<h4 class="text-sm font-medium text-foreground">Time</h4>

		<div class="flex items-center justify-between">
			<span class="text-sm text-muted-foreground">Actual Time</span>
			<span class="text-sm text-foreground flex items-center gap-1">
				<Clock class="w-3 h-3 text-muted-foreground" />
				{formatDuration(task.actualTimeSeconds)}
			</span>
		</div>

		{#if task.startedAt}
			<div class="flex items-center justify-between">
				<span class="text-sm text-muted-foreground">Started</span>
				<span class="text-sm text-foreground">{formatDateTime(task.startedAt)}</span>
			</div>
		{/if}

		{#if task.completedAt}
			<div class="flex items-center justify-between">
				<span class="text-sm text-muted-foreground">Completed</span>
				<span class="text-sm text-foreground">{formatDateTime(task.completedAt)}</span>
			</div>
		{/if}

		{#if task.dueDate}
			<div class="flex items-center justify-between">
				<span class="text-sm text-muted-foreground">Due Date</span>
				<span class="text-sm text-foreground flex items-center gap-1">
					<Calendar class="w-3 h-3 text-muted-foreground" />
					{formatDate(task.dueDate)}
				</span>
			</div>
		{/if}
	</div>

	<!-- Children progress -->
	{#if progress.total > 0}
		<Separator />
		<div class="space-y-2">
			<h4 class="text-sm font-medium text-foreground">Sub-tasks</h4>
			<div class="flex items-center justify-between text-sm text-muted-foreground">
				<span>{progress.done} of {progress.total} complete</span>
				<span>{Math.round((progress.done / progress.total) * 100)}%</span>
			</div>
			<div class="h-2 bg-muted rounded-full overflow-hidden">
				<div
					class="h-full bg-green-500 transition-all duration-300"
					style="width: {(progress.done / progress.total) * 100}%"
				></div>
			</div>
		</div>
	{/if}

	<!-- Dependencies -->
	{#if deps && (deps.blockers.length > 0 || deps.dependents.length > 0)}
		<Separator />
		<div class="space-y-3">
			<h4 class="text-sm font-medium text-foreground">Dependencies</h4>

			{#if deps.blockers.length > 0}
				<div class="space-y-1">
					<span class="text-xs text-muted-foreground uppercase tracking-wide">Blocked by</span>
					{#each deps.blockers as dep (dep.id)}
						<div class="flex items-center gap-2 text-sm text-foreground py-1 px-2 bg-muted/50 rounded">
							<ArrowLeft class="w-3 h-3 text-red-400 shrink-0" />
							<span class="font-mono text-xs">{dep.toTaskId}</span>
						</div>
					{/each}
				</div>
			{/if}

			{#if deps.dependents.length > 0}
				<div class="space-y-1">
					<span class="text-xs text-muted-foreground uppercase tracking-wide">Blocks</span>
					{#each deps.dependents as dep (dep.id)}
						<div class="flex items-center gap-2 text-sm text-foreground py-1 px-2 bg-muted/50 rounded">
							<ArrowRight class="w-3 h-3 text-orange-400 shrink-0" />
							<span class="font-mono text-xs">{dep.fromTaskId}</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- Deviation notes -->
	{#if task.deviationNotes}
		<Separator />
		<div class="space-y-2">
			<h4 class="text-sm font-medium text-foreground">Deviation Notes</h4>
			<p class="text-sm text-muted-foreground whitespace-pre-wrap">{task.deviationNotes}</p>
		</div>
	{/if}

	<!-- Timestamps -->
	<Separator />
	<div class="space-y-2 text-xs text-muted-foreground">
		<div class="flex items-center justify-between">
			<span>Created</span>
			<span>{formatDateTime(task.createdAt)}</span>
		</div>
		<div class="flex items-center justify-between">
			<span>Updated</span>
			<span>{formatDateTime(task.updatedAt)}</span>
		</div>
	</div>
</div>
