<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import favicon from '$lib/assets/favicon.svg';
	import { authStore } from '$lib/stores/auth';
	import { themeStore } from '$lib/stores/theme';
	import AppSidebar from '$lib/components/AppSidebar.svelte';
	import CreateSOPModal from '$lib/components/CreateSOPModal.svelte';
	import SearchForm from '$lib/components/SearchForm.svelte';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Avatar from '$lib/components/ui/avatar';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { Separator } from '$lib/components/ui/separator';
	import { Button } from '$lib/components/ui/button';
	import { Moon, Sun, Plus, FileText } from '@lucide/svelte';
	import { Toaster } from '$lib/components/ui/sonner';

	let { children } = $props();

	let showCreateSOPModal = $state(false);
	let theme: 'light' | 'dark' = $state('light');

	// Use server-provided user from hooks.server.ts via +layout.server.ts
	let user = $derived($page.data.user);

	// Derive user initials for Avatar fallback
	let userInitials = $derived(() => {
		const first = user?.firstName?.[0] ?? '';
		const last = user?.lastName?.[0] ?? '';
		return (first + last).toUpperCase() || '?';
	});

	// Subscribe to theme store
	themeStore.subscribe((value) => {
		theme = value;
	});

	// Check if we're on login, change-password, or onboarding page (no app shell needed)
	let isAuthPage = $derived(() => {
		const pathname: string = $page.url.pathname;
		return pathname === '/login' || pathname === '/change-password' || pathname === '/onboarding';
	});

	onMount(() => {
		// Initialize the client-side auth store for API calls (logout, etc.)
		// Skip on auth pages to avoid a 401 → redirect loop when not logged in.
		if (!isAuthPage()) {
			authStore.initialize();
		}
		themeStore.initialize();
	});

	function handleOpenCreateSOP() {
		showCreateSOPModal = true;
	}

	function handleCloseCreateSOP() {
		showCreateSOPModal = false;
	}

	async function handleLogout() {
		await authStore.logout();
		goto('/login');
	}

	function toggleTheme() {
		themeStore.toggle();
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>Nori - Process Management</title>
</svelte:head>

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
					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							{#snippet child({ props })}
								<Button {...props} size="sm" class="flex items-center gap-2">
									<Plus class="w-4 h-4" />
									Create
								</Button>
							{/snippet}
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="end" class="w-56">
							<DropdownMenu.Item onclick={handleOpenCreateSOP} class="flex items-start gap-3 py-3">
								<FileText class="w-5 h-5 text-primary mt-0.5" />
								<div>
									<div class="text-sm font-medium">SOP</div>
									<div class="text-xs text-muted-foreground">Create new template</div>
								</div>
							</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.Root>

					<!-- Theme Toggle -->
					<Tooltip.Root>
						<Tooltip.Trigger>
							{#snippet child({ props })}
								<Button {...props} onclick={toggleTheme} variant="ghost" size="icon" aria-label="Toggle theme">
									{#if theme === 'dark'}
										<Sun class="w-5 h-5" />
									{:else}
										<Moon class="w-5 h-5" />
									{/if}
								</Button>
							{/snippet}
						</Tooltip.Trigger>
						<Tooltip.Content>Toggle light/dark mode</Tooltip.Content>
					</Tooltip.Root>

					<!-- User Menu -->
					{#if user}
						<div class="flex items-center gap-3 pl-3 border-l border-border">
							<Avatar.Root size="sm">
								<Avatar.Fallback>{userInitials()}</Avatar.Fallback>
							</Avatar.Root>
							<span class="text-sm text-foreground hidden sm:inline">
								{user.firstName ?? ''} {user.lastName ?? ''}
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
<CreateSOPModal isOpen={showCreateSOPModal} onClose={handleCloseCreateSOP} />
<Toaster />
