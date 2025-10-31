<script lang="ts">
 	import '../app.css';
 	import { onMount } from 'svelte';
 	import { navigating } from '$app/stores';
 	import favicon from '$lib/assets/favicon.svg';
 	import { authStore } from '$lib/stores/auth';
 	import { themeStore } from '$lib/stores/theme';
 	import { sidebarStore } from '$lib/stores/sidebar';
 	import AppSidebar from '$lib/components/AppSidebar.svelte';
 	import TopNav from '$lib/components/TopNav.svelte';
 	import CreateTaskModal from '$lib/components/CreateTaskModal.svelte';
 	import CreateSOPModal from '$lib/components/CreateSOPModal.svelte';
 	import type { User } from '$lib/api/auth';

 	let { children } = $props();

 	let showCreateTaskModal = $state(false);
 	let showCreateSOPModal = $state(false);
 	let user = $state<User | null>(null);
 	let collapsed = $state(false);

 	// Subscribe to authStore to check if user is logged in
 	authStore.subscribe((state) => {
 		user = state.user;
 	});

 	// Subscribe to sidebar state
 	sidebarStore.subscribe((state) => {
 		collapsed = state.collapsed;
 	});

 	// Check if we're on login or register page
 	let isAuthPage = $derived(
 		$navigating?.to?.url.pathname === '/login' ||
 			$navigating?.to?.url.pathname === '/register' ||
 			(typeof window !== 'undefined' &&
 				(window.location.pathname === '/login' || window.location.pathname === '/register'))
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
 	<!-- Sidebar -->
 	<AppSidebar />

 	<!-- Main Content Area with Sidebar Offset -->
 	<div
 		class="transition-all duration-300 h-screen flex flex-col"
 		style="margin-left: {collapsed ? '4rem' : '16rem'};"
 	>
 		<TopNav onCreateTask={handleOpenCreateTask} onCreateSOP={handleOpenCreateSOP} />
 		<main class="flex-1 overflow-hidden">
 			{@render children()}
 		</main>
 	</div>
{:else}
 	{@render children()}
{/if}

<!-- Modals -->
<CreateTaskModal isOpen={showCreateTaskModal} onClose={handleCloseCreateTask} />
<CreateSOPModal isOpen={showCreateSOPModal} onClose={handleCloseCreateSOP} />
