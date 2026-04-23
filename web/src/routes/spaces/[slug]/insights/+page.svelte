<script lang="ts">
	import { spaceStore } from '$lib/stores/space';
	import { BarChart3 } from '@lucide/svelte';

	let currentSpace = $derived($spaceStore.currentSpace);
</script>

<svelte:head>
	<title>{currentSpace?.name ?? 'Space'} – Insights - Nori</title>
</svelte:head>

<div class="flex flex-col items-center justify-center h-full min-h-[400px] text-center px-4">
	<div class="rounded-full bg-muted p-4 mb-4">
		<BarChart3 class="size-8 text-muted-foreground" />
	</div>
	<h2 class="text-lg font-semibold text-foreground mb-1">Insights</h2>
	<p class="text-sm text-muted-foreground max-w-sm">
		Analytics and bottleneck analysis are coming soon. You'll be able to see throughput, cycle times, and constraint identification powered by Theory of Constraints.
	</p>
</div>
