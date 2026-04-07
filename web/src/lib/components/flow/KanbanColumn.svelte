<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Skeleton } from '$lib/components/ui/skeleton';

	interface Props {
		title: string;
		count: number;
		colorClass: string;
		isLoading: boolean;
		children: Snippet;
	}

	let { title, count, colorClass, isLoading, children }: Props = $props();
</script>

<div class="flex min-w-[280px] flex-1 flex-col">
	<!-- Column header -->
	<div class="mb-3 flex items-center gap-2">
		<div class="size-2.5 rounded-full {colorClass}"></div>
		<h3 class="text-sm font-semibold text-foreground">{title}</h3>
		<span class="text-xs text-muted-foreground">({count})</span>
	</div>

	<!-- Card list -->
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
				No tasks
			</div>
		{:else}
			{@render children()}
		{/if}
	</div>
</div>
