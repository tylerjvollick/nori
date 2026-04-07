<script lang="ts">
	import type { TaskResponse } from '$lib/types/task';
	import { Badge } from '$lib/components/ui/badge';
	import { Clock, User } from 'lucide-svelte';

	interface Props {
		task: TaskResponse;
		stationMap: Map<string, string>;
	}

	let { task, stationMap }: Props = $props();

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

	let pBadge = $derived(priorityBadge(task.priority));
	let station = $derived(getStationName(task.stationId));
</script>

<a
	href="/flow/{task.id}"
	class="group block rounded-lg border border-border bg-card p-3 shadow-sm hover:border-primary/40 hover:shadow-md transition-all"
>
	<!-- Title -->
	<h4 class="text-sm font-medium text-foreground group-hover:text-primary transition-colors truncate">
		{task.title}
	</h4>

	<!-- Hierarchical ID -->
	<p class="mt-0.5 text-xs font-mono text-muted-foreground truncate">{task.id}</p>

	<!-- Badges row -->
	<div class="mt-2 flex flex-wrap items-center gap-1.5">
		<!-- Priority -->
		<Badge variant={pBadge.variant} class="text-[10px] px-1.5 py-0 h-5 {pBadge.class}">
			{pBadge.label}
		</Badge>

		<!-- Station -->
		{#if station}
			<Badge variant="outline" class="text-[10px] px-1.5 py-0 h-5">
				{station}
			</Badge>
		{/if}

		<!-- Quantity -->
		{#if task.quantity > 1}
			<Badge variant="outline" class="text-[10px] px-1.5 py-0 h-5">
				x{task.quantity}
			</Badge>
		{/if}
	</div>

	<!-- Footer: assignee + time ago -->
	<div class="mt-2 flex items-center justify-between text-[11px] text-muted-foreground">
		<div class="flex items-center gap-1">
			{#if task.assignedToId}
				<span
					class="inline-flex items-center justify-center size-5 rounded-full bg-primary/10 text-primary text-[10px] font-medium"
					title={task.assignedToId}
				>
					<User class="size-3" />
				</span>
			{/if}
		</div>
		<span class="flex items-center gap-0.5">
			<Clock class="size-3" />
			{formatTimeAgo(task.updatedAt)}
		</span>
	</div>
</a>
