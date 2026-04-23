<script lang="ts">
	import { Search } from '@lucide/svelte';
	import * as Sidebar from '$lib/components/ui/sidebar';

	let searchQuery = $state('');

	function handleSearch(e: Event) {
		e.preventDefault();
		// TODO: Implement search functionality
		console.log('Search:', searchQuery);
	}
</script>

<form onsubmit={handleSearch} class="relative px-2 py-2">
	<Search class="absolute left-4 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
	<Sidebar.Input
		type="search"
		placeholder="Search recipes, tasks..."
		class="pl-8"
		bind:value={searchQuery}
		disabled
	/>
</form>
