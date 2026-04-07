<script lang="ts">
	import { page } from '$app/stores';
	import BoardView from '$lib/components/flow/BoardView.svelte';
	import GraphView from '$lib/components/flow/GraphView.svelte';
	import ListView from '$lib/components/flow/ListView.svelte';

	// ---- View mode ----
	type ViewMode = 'board' | 'graph' | 'list';
	let currentView = $derived<ViewMode>(
		(($page.url.searchParams.get('view') as ViewMode) || 'board') as ViewMode,
	);
</script>

<svelte:head>
	<title>Flow - Nori</title>
</svelte:head>

{#if currentView === 'board'}
	<BoardView />
{:else if currentView === 'graph'}
	<GraphView />
{:else if currentView === 'list'}
	<ListView />
{/if}
