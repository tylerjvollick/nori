<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { dndzone, type DndEvent } from 'svelte-dnd-action';
	import type { TaskResponse } from '$lib/types/task';

	interface Props {
		title: string;
		count: number;
		colorClass: string;
		isLoading: boolean;
		emptyLabel?: string;
		/** Label shown when filters are active and no items match. */
		filteredEmptyLabel?: string;
		children: Snippet;
		/** When provided, enables drag-and-drop on this column's card container. */
		dndItems?: TaskResponse[];
		dndType?: string;
		dndDropFromOthersDisabled?: boolean;
		dndDragDisabled?: boolean;
		onconsider?: (e: CustomEvent<DndEvent<TaskResponse>>) => void;
		onfinalize?: (e: CustomEvent<DndEvent<TaskResponse>>) => void;
	}

	let {
		title,
		count,
		colorClass,
		isLoading,
		emptyLabel = 'No tasks',
		filteredEmptyLabel,
		children,
		dndItems,
		dndType = 'board-tasks',
		dndDropFromOthersDisabled = false,
		dndDragDisabled = false,
		onconsider,
		onfinalize,
	}: Props = $props();

	let effectiveEmptyLabel = $derived(filteredEmptyLabel ?? emptyLabel);

	let isDndEnabled = $derived(!!dndItems && !isLoading);

	const FLIP_DURATION_MS = 200;

	function handleConsider(e: CustomEvent<DndEvent<TaskResponse>>): void {
		onconsider?.(e);
	}

	function handleFinalize(e: CustomEvent<DndEvent<TaskResponse>>): void {
		onfinalize?.(e);
	}
</script>

<div class="flex min-w-[280px] flex-1 flex-col">
	<!-- Column header -->
	<div class="mb-3 flex items-center gap-2">
		<div class="size-2.5 rounded-full {colorClass}"></div>
		<h3 class="text-sm font-semibold text-foreground">{title}</h3>
		<span class="text-xs text-muted-foreground">({count})</span>
	</div>

	<!-- Card list -->
	{#if isDndEnabled && dndItems}
		<div class="flex flex-1 flex-col relative">
			<div
				class="flex flex-1 flex-col gap-2 rounded-lg bg-muted/30 p-2 overflow-y-auto min-h-[80px]"
				use:dndzone={{
					items: dndItems,
					type: dndType,
					flipDurationMs: FLIP_DURATION_MS,
					dropFromOthersDisabled: dndDropFromOthersDisabled,
					dragDisabled: dndDragDisabled,
					centreDraggedOnCursor: true,
				}}
				onconsider={handleConsider}
				onfinalize={handleFinalize}
			>
				{@render children()}
			</div>
			{#if count === 0}
				<div class="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground pointer-events-none">
					{effectiveEmptyLabel}
				</div>
			{/if}
		</div>
	{:else}
		<div class="flex flex-1 flex-col gap-2 rounded-lg bg-muted/30 p-2 overflow-y-auto">
			{#if isLoading}
				{#each Array(3) as _}
					<div class="rounded-lg border border-border bg-card p-3">
						<Skeleton class="h-4 w-3/4 mb-2" />
						<Skeleton class="h-3 w-1/2 mb-2" />
						<div class="flex gap-1.5">
							<Skeleton class="h-5 w-10 rounded-full" />
							<Skeleton class="h-5 w-16 rounded-full" />
						</div>
					</div>
				{/each}
			{:else if count === 0}
				<div class="flex items-center justify-center py-8 text-sm text-muted-foreground">
					{effectiveEmptyLabel}
				</div>
			{:else}
				{@render children()}
			{/if}
		</div>
	{/if}
</div>
