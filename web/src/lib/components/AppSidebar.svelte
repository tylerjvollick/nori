<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { sidebarStore } from '$lib/stores/sidebar';
	import { spaceStore } from '$lib/stores/space';
	import type { Space } from '$lib/api/space';
	import { cn } from '$lib/utils';
	import {
		LayoutGrid,
		Plus,
		FileText,
		Package,
		Wrench,
		ChevronDown,
		X,
		Menu,
		Folder
	} from 'lucide-svelte';

	let collapsed = $state(false);
	let spacesExpanded = $state(true);
	let mobileOpen = $state(false);
	let showCreateDialog = $state(false);
	let newSpaceName = $state('');
	let isCreating = $state(false);

	// Subscribe to sidebar store
	sidebarStore.subscribe((state) => {
		collapsed = state.collapsed;
		spacesExpanded = state.spacesExpanded;
	});

	// Load recent spaces on mount
	onMount(() => {
		spaceStore.loadRecentSpaces();
	});

	// Get recent spaces from store using derived state
	let spaces = $derived($spaceStore.recentSpaces);

	function toggleSidebar() {
		sidebarStore.toggle();
	}

	function toggleSpaces() {
		sidebarStore.toggleSpaces();
	}

	function toggleMobileSidebar() {
		mobileOpen = !mobileOpen;
	}

	function navigateTo(href: string) {
		goto(href);
		mobileOpen = false; // Close mobile sidebar on navigation
	}

	function isActive(href: string): boolean {
		if (typeof window === 'undefined') return false;
		return (
			window.location.pathname === href || window.location.pathname.startsWith(href + '/')
		);
	}

	async function handleCreateSpace() {
		if (!newSpaceName.trim() || isCreating) return;

		isCreating = true;
		const space = await spaceStore.createSpace(newSpaceName.trim());
		isCreating = false;

		if (space) {
			newSpaceName = '';
			showCreateDialog = false;
			// Refresh recent spaces
			spaceStore.loadRecentSpaces();
			// Navigate to the new space
			goto(`/spaces/${space.id}`);
		}
	}
</script>

<!-- Mobile Menu Button -->
<button
	onclick={toggleMobileSidebar}
	class="fixed top-4 left-4 z-50 p-2 rounded-lg bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 shadow-lg md:hidden"
	aria-label={mobileOpen ? 'Close menu' : 'Open menu'}
>
	{#if mobileOpen}
		<X class="w-5 h-5" />
	{:else}
		<Menu class="w-5 h-5" />
	{/if}
</button>

<!-- Mobile Overlay -->
{#if mobileOpen}
	<button
		onclick={toggleMobileSidebar}
		class="fixed inset-0 bg-black/50 z-40 md:hidden"
		aria-label="Close sidebar"
	></button>
{/if}

<!-- Sidebar -->
<aside
	class={cn(
		'fixed left-0 top-0 h-screen bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700 transition-all duration-300 z-50 flex flex-col',
		// Desktop
		'hidden md:flex',
		collapsed ? 'w-16' : 'w-64',
		// Mobile
		'md:translate-x-0',
		mobileOpen ? 'flex translate-x-0' : '-translate-x-full',
		mobileOpen && 'w-64'
	)}
>
	<!-- Sidebar Header -->
	<div class={cn(
		"h-16 flex items-center px-4 border-b border-gray-200 dark:border-gray-700",
		collapsed ? "justify-center" : "justify-start"
	)}>
		{#if !collapsed}
			<button
				onclick={() => navigateTo('/')}
				class="flex items-center gap-2 hover:opacity-80 transition-opacity"
			>
				<div
					class="w-8 h-8 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-lg flex items-center justify-center"
				>
					<svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M13 10V3L4 14h7v7l9-11h-7z"
						/>
					</svg>
				</div>
				<span class="font-bold text-gray-900 dark:text-white">Nori</span>
			</button>
		{:else}
			<button
				onclick={() => navigateTo('/')}
				class="w-8 h-8 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-lg flex items-center justify-center hover:opacity-80 transition-opacity"
			>
				<svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M13 10V3L4 14h7v7l9-11h-7z"
					/>
				</svg>
			</button>
		{/if}
	</div>

	<!-- Navigation -->
	<nav class="flex-1 overflow-y-auto p-3 space-y-1">
		<!-- Spaces Section -->
		<div>
			<button
				onclick={toggleSpaces}
				class={cn(
					'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
					'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800',
					collapsed && 'justify-center'
				)}
				title={collapsed ? 'Spaces' : ''}
			>
				<LayoutGrid class="w-5 h-5 shrink-0" />
				{#if !collapsed}
					<span class="flex-1 text-left">Spaces</span>
					<ChevronDown
						class={cn('w-4 h-4 transition-transform', spacesExpanded && 'rotate-180')}
					/>
				{/if}
			</button>

			<!-- Spaces List -->
			{#if spacesExpanded && !collapsed}
				<div class="ml-4 mt-1 space-y-1 border-l border-gray-200 dark:border-gray-700 pl-4">
					{#if $spaceStore.isLoading}
						<div class="px-3 py-1.5 text-sm text-gray-400">Loading...</div>
					{:else if spaces.length === 0}
						<div class="px-3 py-1.5 text-sm text-gray-400">No recent spaces</div>
					{:else}
						{#each spaces as space}
							<button
								onclick={() => navigateTo(`/spaces/${space.id}`)}
								class={cn(
									'w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors',
									isActive(`/spaces/${space.id}`)
										? 'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-400 font-medium'
										: 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'
								)}
							>
								<Folder class="w-4 h-4 shrink-0" />
								<span class="truncate">{space.name}</span>
							</button>
						{/each}
					{/if}

					<!-- Create Space Button -->
					<button
						onclick={() => (showCreateDialog = true)}
						class="w-full flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm text-gray-500 dark:text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800 hover:text-emerald-600 dark:hover:text-emerald-400 transition-colors"
					>
						<Plus class="w-4 h-4 shrink-0" />
						<span>Create Space</span>
					</button>
				</div>
			{/if}
		</div>

		<!-- SOPs -->
		<button
			onclick={() => navigateTo('/sops')}
			class={cn(
				'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
				isActive('/sops')
					? 'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-400'
					: 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800',
				collapsed && 'justify-center'
			)}
			title={collapsed ? 'SOPs' : ''}
		>
			<FileText class="w-5 h-5 shrink-0" />
			{#if !collapsed}
				<span>SOPs</span>
			{/if}
		</button>

		<!-- Divider -->
		<div class="my-3 border-t border-gray-200 dark:border-gray-700"></div>

		<!-- Materials (Placeholder) -->
		<button
			onclick={() => alert('Materials feature coming soon!')}
			class={cn(
				'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
				'text-gray-500 dark:text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800',
				collapsed && 'justify-center'
			)}
			title={collapsed ? 'Materials' : ''}
		>
			<Package class="w-5 h-5 shrink-0" />
			{#if !collapsed}
				<span>Materials</span>
			{/if}
		</button>

		<!-- Equipment (Placeholder) -->
		<button
			onclick={() => alert('Equipment feature coming soon!')}
			class={cn(
				'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
				'text-gray-500 dark:text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-800',
				collapsed && 'justify-center'
			)}
			title={collapsed ? 'Equipment' : ''}
		>
			<Wrench class="w-5 h-5 shrink-0" />
			{#if !collapsed}
				<span>Equipment</span>
			{/if}
		</button>
	</nav>
</aside>

<!-- Create Space Dialog -->
{#if showCreateDialog}
	<div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
		<div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full p-6">
			<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-4">Create New Space</h2>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleCreateSpace();
				}}
			>
				<div class="mb-4">
					<label for="space-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
						Space Name
					</label>
					<input
						id="space-name"
						type="text"
						bind:value={newSpaceName}
						placeholder="e.g., Marketing, Engineering, HR"
						class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
						disabled={isCreating}
						autofocus
					/>
				</div>
				{#if $spaceStore.error}
					<div class="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-600 dark:text-red-400">
						{$spaceStore.error}
					</div>
				{/if}
				<div class="flex gap-3 justify-end">
					<button
						type="button"
						onclick={() => {
							showCreateDialog = false;
							newSpaceName = '';
							spaceStore.clearError();
						}}
						disabled={isCreating}
						class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={!newSpaceName.trim() || isCreating}
						class="px-4 py-2 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
					>
						{isCreating ? 'Creating...' : 'Create Space'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
