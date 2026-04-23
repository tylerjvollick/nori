<script lang="ts">
	import { page } from '$app/stores';
	import { spaceStore } from '$lib/stores/space';
	import BoardView from '$lib/components/flow/BoardView.svelte';
	import ListView from '$lib/components/flow/ListView.svelte';

	let currentSpace = $derived($spaceStore.currentSpace);

	// ---- View mode ----
	type ViewMode = 'board' | 'list';
	let currentView = $derived<ViewMode>(
		(($page.url.searchParams.get('view') as ViewMode) || 'board') as ViewMode,
	);
</script>

<svelte:head>
	<title>{currentSpace?.name ?? 'Space'} - Nori</title>
</svelte:head>

{#if currentView === 'board'}
	<BoardView />
{:else if currentView === 'list'}
	<ListView />
{/if}
