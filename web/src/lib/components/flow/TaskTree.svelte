<script lang="ts">
	import type { TaskTreeResponse } from '$lib/api/task';
	import type { StationResponse } from '$lib/types/station';
	import Self from './TaskTree.svelte';
	import { Badge } from '$lib/components/ui/badge';
	import {
		ChevronRight,
		ChevronDown,
		Circle,
		CircleDot,
		CircleCheck,
		CirclePause,
		CircleX,
		CircleMinus,
		Clock,
		User,
	} from 'lucide-svelte';

	interface Props {
		nodes: TaskTreeResponse[];
		stationMap: Map<string, string>;
		selectedTaskId?: string | null;
		depth?: number;
		onselect: (task: TaskTreeResponse) => void;
	}

	let {
		nodes,
		stationMap,
		selectedTaskId = null,
		depth = 0,
		onselect,
	}: Props = $props();

	/** Track which node IDs are expanded. */
	let expandedIds = $state<Set<string>>(new Set());

	/** Auto-expand first 2 levels on initial render. */
	function isExpanded(nodeId: string): boolean {
		// On first render, auto-expand nodes at depth < 2
		// After user interaction, use the expandedIds set
		if (!userHasToggled && depth < 2) return true;
		return expandedIds.has(nodeId);
	}

	let userHasToggled = $state(false);

	function toggleNode(e: Event, nodeId: string): void {
		e.stopPropagation();
		if (!userHasToggled) {
			// Initialize expanded set from auto-expanded state
			for (const node of nodes) {
				if (depth < 2) expandedIds.add(node.id);
			}
			userHasToggled = true;
		}
		if (expandedIds.has(nodeId)) {
			expandedIds.delete(nodeId);
		} else {
			expandedIds.add(nodeId);
		}
		// Trigger reactivity
		expandedIds = new Set(expandedIds);
	}

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

<div class="space-y-0.5">
	{#each nodes as node (node.id)}
		{@const hasChildren = node.children && node.children.length > 0}
		{@const expanded = isExpanded(node.id)}
		{@const station = getStationName(node.stationId)}
		{@const isSelected = selectedTaskId === node.id}
		{@const iconSize = depth === 0 ? 'w-5 h-5' : depth === 1 ? 'w-4 h-4' : 'w-3.5 h-3.5'}

		<div>
			<!-- Task row -->
			<div
				class="flex items-center gap-2 py-1.5 px-2 rounded-md transition-colors cursor-pointer
					{isSelected ? 'bg-accent' : 'hover:bg-accent/50'}"
				onclick={() => onselect(node)}
				onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') onselect(node); }}
				role="button"
				tabindex="0"
			>
				<!-- Expand/collapse toggle -->
				{#if hasChildren}
					<button
						class="p-0.5 rounded hover:bg-accent transition-colors shrink-0"
						onclick={(e) => toggleNode(e, node.id)}
					>
						{#if expanded}
							<ChevronDown class="w-4 h-4 text-muted-foreground" />
						{:else}
							<ChevronRight class="w-4 h-4 text-muted-foreground" />
						{/if}
					</button>
				{:else}
					<div class="w-5 shrink-0"></div>
				{/if}

				<!-- Status icon -->
				{@render statusIcon(node.status, iconSize)}

				<!-- Task info -->
				<div class="flex-1 min-w-0 flex items-center gap-2">
					<span class="text-sm {depth === 0 ? 'font-medium' : ''} text-foreground truncate">
						{node.title}
					</span>
					{#if node.type === 'gate'}
						<Badge variant="outline" class="text-[10px] py-0">Gate</Badge>
					{:else if node.type === 'milestone'}
						<Badge variant="outline" class="text-[10px] py-0">Milestone</Badge>
					{/if}
					{#if station}
						<Badge variant="secondary" class="text-[10px] py-0">
							{station}
						</Badge>
					{/if}
				</div>

				<!-- Right side metadata -->
				<div class="flex items-center gap-2 shrink-0">
					{#if node.actualTimeSeconds > 0}
						<span class="text-xs text-muted-foreground flex items-center gap-1">
							<Clock class="w-3 h-3" />
							{formatDuration(node.actualTimeSeconds)}
						</span>
					{/if}
					{#if node.assignedToId}
						<span class="text-xs text-muted-foreground flex items-center gap-1">
							<User class="w-3 h-3" />
						</span>
					{/if}
					<span class="text-[10px] text-muted-foreground font-mono">
						{node.id}
					</span>
				</div>
			</div>

			<!-- Children (recursive) -->
			{#if hasChildren && expanded}
				<div class="ml-5 border-l border-border/50 pl-2">
					<Self
						nodes={node.children}
						{stationMap}
						{selectedTaskId}
						depth={depth + 1}
						{onselect}
					/>
				</div>
			{/if}
		</div>
	{/each}
</div>
