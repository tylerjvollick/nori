<script lang="ts">
 	import '../app.css';
 	import { onMount } from 'svelte';
 	import { page } from '$app/stores';
 	import favicon from '$lib/assets/favicon.svg';
 	import { authStore } from '$lib/stores/auth';
 	import { themeStore } from '$lib/stores/theme';
 	import TopNav from '$lib/components/TopNav.svelte';
 	import CreateTaskModal from '$lib/components/CreateTaskModal.svelte';
 	import CreateSOPModal from '$lib/components/CreateSOPModal.svelte';

 	let { children } = $props();

 	let showCreateTaskModal = $state(false);
 	let showCreateSOPModal = $state(false);
 	let user = $state(null);

 	// Subscribe to authStore to check if user is logged in
 	authStore.subscribe((state) => {
 		user = state.user;
 	});

 	// Check if we're on login or register page
 	let isAuthPage = $derived(
 		$page.url.pathname === '/login' || $page.url.pathname === '/register'
 	);

 	onMount(() => {
 		authStore.initialize();
 		themeStore.initialize();
 	});

 	function handleOpenCreateTask() {
 		showCreateTaskModal = true;
 	}

 	function handleOpenCreateSOP() {
 		showCreateSOPModal = true;
 	}

 	function handleCloseCreateTask() {
 		showCreateTaskModal = false;
 	}

 	function handleCloseCreateSOP() {
 		showCreateSOPModal = false;
 	}
</script>

<svelte:head>
 	<link rel="icon" href={favicon} />
 	<title>Nori - Process Management</title>
</svelte:head>

{#if !isAuthPage && user}
 	<TopNav onCreateTask={handleOpenCreateTask} onCreateSOP={handleOpenCreateSOP} />
{/if}

{@render children()}

<!-- Modals -->
<CreateTaskModal isOpen={showCreateTaskModal} onClose={handleCloseCreateTask} />
<CreateSOPModal isOpen={showCreateSOPModal} onClose={handleCloseCreateSOP} />
