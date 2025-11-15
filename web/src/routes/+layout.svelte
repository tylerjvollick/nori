<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import favicon from '$lib/assets/favicon.svg';
	import { authStore } from '$lib/stores/auth';
	import { themeStore } from '$lib/stores/theme';
	import AppSidebar from '$lib/components/AppSidebar.svelte';
	import CreateTaskModal from '$lib/components/CreateTaskModal.svelte';
	import CreateSOPModal from '$lib/components/CreateSOPModal.svelte';
	import SearchForm from '$lib/components/SearchForm.svelte';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import { Separator } from '$lib/components/ui/separator';
	import { Button } from '$lib/components/ui/button';
	import { Moon, Sun, Plus } from 'lucide-svelte';
	import type { User } from '$lib/api/auth';

	let { children } = $props();

	let showCreateTaskModal = $state(false);
	let showCreateSOPModal = $state(false);
	let showCreateDropdown = $state(false);
	let user = $state<User | null>(null);
	let theme: 'light' | 'dark' = $state('light');

	// Subscribe to authStore to check if user is logged in
	authStore.subscribe((state) => {
		user = state.user;
	});

	// Subscribe to theme store
	themeStore.subscribe((value) => {
		theme = value;
	});

	// Check if we're on login or register page
	let isAuthPage = $derived(() => {
		const pathname = $page.url.pathname;
		return pathname === '/login' || pathname === '/register';
	});

	onMount(() => {
		authStore.initialize();
		themeStore.initialize();
	});

	function handleOpenCreateTask() {
		showCreateTaskModal = true;
		showCreateDropdown = false;
	}

	function handleOpenCreateSOP() {
		showCreateSOPModal = true;
		showCreateDropdown = false;
	}

	function handleCloseCreateTask() {
		showCreateTaskModal = false;
	}

	function handleCloseCreateSOP() {
		showCreateSOPModal = false;
	}

	function handleLogout() {
		authStore.logout();
	}

	function toggleTheme() {
		themeStore.toggle();
	}

	function closeDropdown() {
		showCreateDropdown = false;
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Nori - Process Management</title>
</svelte:head>

<svelte:window onclick={closeDropdown} />

{#if !isAuthPage() && user}
	<Sidebar.Provider class="h-svh overflow-hidden">
		<AppSidebar />
		<Sidebar.Inset class="h-full overflow-hidden">
			<!-- Header - Sticky Top Nav -->
			<header class="flex-shrink-0 z-10 flex h-16 items-center gap-2 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 px-4">
				<Sidebar.Trigger class="-ml-1" />
				<Separator orientation="vertical" class="mr-2 h-4" />
				
				<!-- Search Bar -->
				<div class="flex-1 max-w-md">
					<SearchForm />
				</div>

				<!-- Right side actions -->
				<div class="ml-auto flex items-center gap-3">
					<!-- Create Dropdown -->
					<div class="relative">
						<Button
							onclick={(e) => {
								e.stopPropagation();
								showCreateDropdown = !showCreateDropdown;
							}}
							class="flex items-center gap-2"
							size="sm"
						>
							<Plus class="w-4 h-4" />
							Create
						</Button>

						{#if showCreateDropdown}
							<div
								role="menu"
								tabindex="-1"
								class="absolute right-0 mt-2 w-56 bg-card rounded-lg shadow-lg border border-border py-1 z-50"
								onclick={(e) => e.stopPropagation()}
								onkeydown={(e) => e.key === 'Escape' && closeDropdown()}
							>
								<Button
									onclick={handleOpenCreateTask}
									variant="ghost"
									class="w-full px-4 py-3 text-left hover:bg-accent transition-colors flex items-start gap-3 h-auto justify-start"
								>
									<svg
										class="w-5 h-5 text-primary mt-0.5"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
										/>
									</svg>
									<div>
										<div class="text-sm font-medium text-foreground">Task</div>
										<div class="text-xs text-muted-foreground">From SOP template</div>
									</div>
								</Button>
								<Button
									onclick={handleOpenCreateSOP}
									variant="ghost"
									class="w-full px-4 py-3 text-left hover:bg-accent transition-colors flex items-start gap-3 h-auto justify-start"
								>
									<svg
										class="w-5 h-5 text-primary mt-0.5"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
										/>
									</svg>
									<div>
										<div class="text-sm font-medium text-foreground">SOP</div>
										<div class="text-xs text-muted-foreground">Create new template</div>
									</div>
								</Button>
							</div>
						{/if}
					</div>

					<!-- Theme Toggle -->
					<Button onclick={toggleTheme} variant="ghost" size="icon" aria-label="Toggle theme">
						{#if theme === 'dark'}
							<Sun class="w-5 h-5" />
						{:else}
							<Moon class="w-5 h-5" />
						{/if}
					</Button>

					<!-- User Menu -->
					{#if user}
						<div class="flex items-center gap-3 pl-3 border-l border-border">
							<span class="text-sm text-foreground hidden sm:inline">
								{user.firstName}
								{user.lastName}
							</span>
							<Button onclick={handleLogout} variant="ghost" size="sm" class="text-sm">
								Logout
							</Button>
						</div>
					{/if}
				</div>
			</header>

			<!-- Main Content -->
			<div class="flex flex-1 flex-col overflow-hidden">
				{@render children()}
			</div>
		</Sidebar.Inset>
	</Sidebar.Provider>
{:else}
	{@render children()}
{/if}

<!-- Modals -->
<CreateTaskModal isOpen={showCreateTaskModal} onClose={handleCloseCreateTask} />
<CreateSOPModal isOpen={showCreateSOPModal} onClose={handleCloseCreateSOP} />
