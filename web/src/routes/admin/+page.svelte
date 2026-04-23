<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import LoadingPage from '$lib/components/LoadingPage.svelte';

	onMount(() => {
		goto('/admin/users', { replaceState: true });
	});
</script>

<LoadingPage message="Redirecting..." />
