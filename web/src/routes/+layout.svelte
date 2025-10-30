<script lang="ts">
 	import { onMount } from 'svelte';
 	import favicon from '$lib/assets/favicon.svg';
 	import { authStore } from '$lib/stores/auth';

 	let { children } = $props();

 	onMount(() => {
 		authStore.initialize();
 	});
</script>

<svelte:head>
 	<link rel="icon" href={favicon} />
 	<title>Nori - Process Management</title>
</svelte:head>

{@render children()}
