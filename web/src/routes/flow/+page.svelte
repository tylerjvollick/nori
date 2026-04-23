<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { spaceStore } from '$lib/stores/space';
	import LoadingPage from '$lib/components/LoadingPage.svelte';

	/**
	 * Legacy /flow route — redirects to the user's first space.
	 * Kept for backwards compatibility with bookmarks and external links.
	 */
	onMount(async () => {
		// Load spaces if not already loaded
		if ($spaceStore.recentSpaces.length === 0) {
			await spaceStore.loadRecentSpaces();
		}

		const spaces = $spaceStore.recentSpaces;
		if (spaces.length > 0 && spaces[0].slug) {
			goto(`/spaces/${spaces[0].slug}`, { replaceState: true });
		} else {
			// No spaces available — go to dashboard
			goto('/', { replaceState: true });
		}
	});
</script>

<LoadingPage message="Redirecting..." />
