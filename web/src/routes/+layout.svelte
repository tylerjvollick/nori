<script lang="ts">
 	import '../app.css';
 	import { onMount } from 'svelte';
 	import favicon from '$lib/assets/favicon.svg';
 	import { authStore } from '$lib/stores/auth';
 	import { themeStore } from '$lib/stores/theme';

 	let { children } = $props();

 	onMount(() => {
 		authStore.initialize();
 		themeStore.initialize();
 	});
</script>

<svelte:head>
 	<link rel="icon" href={favicon} />
 	<title>Nori - Process Management</title>
</svelte:head>

{@render children()}
