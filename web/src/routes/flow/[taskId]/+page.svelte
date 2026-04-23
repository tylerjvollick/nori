<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { spaceStore } from '$lib/stores/space';

	let taskId = $derived($page.params.taskId);

	/**
	 * Legacy /flow/[taskId] route — redirects to /spaces/:slug/:taskId.
	 * Kept for backwards compatibility with bookmarks and external links.
	 */
	onMount(async () => {
		// Load spaces if not already loaded
		if ($spaceStore.recentSpaces.length === 0) {
			await spaceStore.loadRecentSpaces();
		}

		const spaces = $spaceStore.recentSpaces;
		if (spaces.length > 0 && spaces[0].slug) {
			goto(`/spaces/${spaces[0].slug}/${taskId}`, { replaceState: true });
		} else {
			// No spaces available — go to dashboard
			goto('/', { replaceState: true });
		}
	});
</script>

<div class="flex items-center justify-center h-full">
	<p class="text-sm text-muted-foreground">Redirecting...</p>
</div>
