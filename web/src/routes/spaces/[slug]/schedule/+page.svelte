<script lang="ts">
	import { page } from '$app/stores';
	import { Calendar } from '@lucide/svelte';

	let space = $derived($page.data.space!);
</script>

<svelte:head>
	<title>{space.name} – Schedule - Nori</title>
</svelte:head>

<div class="flex flex-col items-center justify-center h-full min-h-[400px] text-center px-4">
	<div class="rounded-full bg-muted p-4 mb-4">
		<Calendar class="size-8 text-muted-foreground" />
	</div>
	<h2 class="text-lg font-semibold text-foreground mb-1">Schedule</h2>
	<p class="text-sm text-muted-foreground max-w-sm">
		Production scheduling is coming soon. You'll be able to plan capacity and sequence jobs across stations.
	</p>
</div>
