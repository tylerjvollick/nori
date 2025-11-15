<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { spaceStore } from '$lib/stores/space';
	import { authStore } from '$lib/stores/auth';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import * as Collapsible from '$lib/components/ui/collapsible';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { useSidebar } from '$lib/components/ui/sidebar/context.svelte.js';
	import {
		LayoutGrid,
		FileText,
		Package,
		Wrench,
		Folder,
		Plus,
		ChevronRight,
		Ellipsis,
		LogOut,
		User as UserIcon
	} from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import type { ComponentProps } from 'svelte';
	import type { User } from '$lib/api/auth';

	let {
		ref = $bindable(null),
		collapsible = "icon",
		...restProps
	}: ComponentProps<typeof Sidebar.Root> = $props();

	let spaces = $derived($spaceStore.recentSpaces);
	let user = $state<User | null>(null);
	let showCreateDialog = $state(false);
	let newSpaceName = $state('');
	let isCreating = $state(false);

	const sidebar = useSidebar();

	// Subscribe to auth store
	authStore.subscribe((state) => {
		user = state.user;
	});

	// Load recent spaces on mount
	onMount(() => {
		spaceStore.loadRecentSpaces();
	});

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
			spaceStore.loadRecentSpaces();
			goto(`/spaces/${space.id}`);
		}
	}

	function handleLogout() {
		authStore.logout();
		goto('/login');
	}

	// Navigation structure
	const navMain = [
		{
			title: 'SOPs',
			url: '/sops',
			icon: FileText,
			isActive: isActive('/sops')
		}
	];

	const navResources = [
		{
			title: 'Materials',
			url: '#',
			icon: Package
		},
		{
			title: 'Equipment',
			url: '#',
			icon: Wrench
		}
	];
</script>

<Sidebar.Root {collapsible} {...restProps} bind:ref>
	<Sidebar.Header>
		<div class="flex items-center gap-2 px-2 group-data-[collapsible=icon]:justify-center h-(--header-height)">
			<div class="size-10 bg-sidebar-primary rounded-lg flex items-center justify-center shrink-0">
				<svg
					class="size-6 text-sidebar-primary-foreground"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M13 10V3L4 14h7v7l9-11h-7z"
					/>
				</svg>
			</div>
			<span class="font-bold text-sidebar-foreground group-data-[collapsible=icon]:hidden">Nori</span>
		</div>
	</Sidebar.Header>

	<Sidebar.Content>
		<!-- Spaces Section with Collapsible -->
		<Sidebar.Group>
			<Sidebar.GroupLabel>Workspace</Sidebar.GroupLabel>
			<Sidebar.Menu>
				<Collapsible.Root open={true} class="group/collapsible">
					{#snippet child({ props })}
						<Sidebar.MenuItem {...props}>
							<Collapsible.Trigger>
								{#snippet child({ props })}
									<Sidebar.MenuButton {...props} tooltipContent="Spaces">
										<LayoutGrid />
										<span>Spaces</span>
										<ChevronRight
											class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
										/>
									</Sidebar.MenuButton>
								{/snippet}
							</Collapsible.Trigger>
							<Collapsible.Content>
								<Sidebar.MenuSub>
									{#if $spaceStore.isLoading}
										<Sidebar.MenuSubItem>
											<span class="text-sm text-muted-foreground pl-8">Loading...</span>
										</Sidebar.MenuSubItem>
									{:else if spaces.length === 0}
										<Sidebar.MenuSubItem>
											<span class="text-sm text-muted-foreground pl-8">No spaces yet</span>
										</Sidebar.MenuSubItem>
									{:else}
										{#each spaces as space (space.id)}
											<Sidebar.MenuSubItem>
												<Sidebar.MenuSubButton
													isActive={isActive(`/spaces/${space.id}`)}
													onclick={() => goto(`/spaces/${space.id}`)}
												>
													{#snippet child({ props })}
														<a href={`/spaces/${space.id}`} {...props}>
															<Folder />
															<span>{space.name}</span>
														</a>
													{/snippet}
												</Sidebar.MenuSubButton>
											</Sidebar.MenuSubItem>
										{/each}
									{/if}
									<!-- Create Space Button -->
									<Sidebar.MenuSubItem>
										<Sidebar.MenuSubButton
											onclick={() => (showCreateDialog = true)}
											class="text-muted-foreground"
										>
											{#snippet child({ props })}
												<button type="button" {...props}>
													<Plus />
													<span>Create Space</span>
												</button>
											{/snippet}
										</Sidebar.MenuSubButton>
									</Sidebar.MenuSubItem>
								</Sidebar.MenuSub>
							</Collapsible.Content>
						</Sidebar.MenuItem>
					{/snippet}
				</Collapsible.Root>

				<!-- SOPs -->
				{#each navMain as item (item.title)}
					<Sidebar.MenuItem>
						<Sidebar.MenuButton
							isActive={item.isActive}
							onclick={() => goto(item.url)}
							tooltipContent={item.title}
						>
							{#snippet child({ props })}
								<a href={item.url} {...props}>
									<item.icon />
									<span>{item.title}</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				{/each}
			</Sidebar.Menu>
		</Sidebar.Group>

		<Sidebar.Separator />

		<!-- Resources Section -->
		<Sidebar.Group class="group-data-[collapsible=icon]:hidden">
			<Sidebar.GroupLabel>Resources</Sidebar.GroupLabel>
			<Sidebar.Menu>
				{#each navResources as item (item.title)}
					<Sidebar.MenuItem>
						<Sidebar.MenuButton onclick={() => alert(`${item.title} feature coming soon!`)}>
							{#snippet child({ props })}
								<button type="button" {...props}>
									<item.icon />
									<span>{item.title}</span>
								</button>
							{/snippet}
						</Sidebar.MenuButton>
						<DropdownMenu.Root>
							<DropdownMenu.Trigger>
								{#snippet child({ props: actionProps })}
									<Sidebar.MenuAction showOnHover {...actionProps}>
										<Ellipsis />
										<span class="sr-only">More</span>
									</Sidebar.MenuAction>
								{/snippet}
							</DropdownMenu.Trigger>
							<DropdownMenu.Content
								class="w-48 rounded-lg"
								side={sidebar.isMobile ? 'bottom' : 'right'}
								align={sidebar.isMobile ? 'end' : 'start'}
							>
								<DropdownMenu.Item disabled>
									<span class="text-muted-foreground">Coming Soon</span>
								</DropdownMenu.Item>
							</DropdownMenu.Content>
						</DropdownMenu.Root>
					</Sidebar.MenuItem>
				{/each}
			</Sidebar.Menu>
		</Sidebar.Group>
	</Sidebar.Content>

	<Sidebar.Footer>
		<!-- User Menu -->
		{#if user}
			<Sidebar.Menu>
				<Sidebar.MenuItem>
					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							{#snippet child({ props: triggerProps })}
								<Sidebar.MenuButton
									size="lg"
									class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
									{...triggerProps}
								>
									<div
										class="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground"
									>
										<UserIcon class="size-4" />
									</div>
									<div class="grid flex-1 text-left text-sm leading-tight">
										<span class="truncate font-semibold"
											>{user?.firstName} {user?.lastName}</span
										>
										<span class="truncate text-xs text-muted-foreground">{user?.email}</span>
									</div>
									<ChevronRight class="ml-auto size-4" />
								</Sidebar.MenuButton>
							{/snippet}
						</DropdownMenu.Trigger>
						<DropdownMenu.Content
							class="w-56 rounded-lg"
							side={sidebar.isMobile ? 'bottom' : 'right'}
							align="end"
						>
							<DropdownMenu.Separator />
							<DropdownMenu.Item onclick={handleLogout}>
								<LogOut />
								<span>Log out</span>
							</DropdownMenu.Item>
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</Sidebar.MenuItem>
			</Sidebar.Menu>
		{/if}
	</Sidebar.Footer>

	<Sidebar.Rail />
</Sidebar.Root>

<!-- Create Space Dialog -->
{#if showCreateDialog}
	<div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
		<div class="bg-card rounded-lg shadow-xl max-w-md w-full p-6">
			<h2 class="text-xl font-semibold text-card-foreground mb-4">Create New Space</h2>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleCreateSpace();
				}}
			>
				<div class="mb-4">
					<label for="space-name" class="block text-sm font-medium text-foreground mb-2">
						Space Name
					</label>
					<input
						id="space-name"
						type="text"
						bind:value={newSpaceName}
						placeholder="e.g., Marketing, Engineering, HR"
						class="w-full px-3 py-2 border border-input rounded-lg bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent"
						disabled={isCreating}
					/>
				</div>
				{#if $spaceStore.error}
					<div
						class="mb-4 p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-sm text-destructive"
					>
						{$spaceStore.error}
					</div>
				{/if}
				<div class="flex gap-3 justify-end">
					<Button
						type="button"
						variant="outline"
						onclick={() => {
							showCreateDialog = false;
							newSpaceName = '';
							spaceStore.clearError();
						}}
						disabled={isCreating}
					>
						Cancel
					</Button>
					<Button type="submit" disabled={!newSpaceName.trim() || isCreating}>
						{isCreating ? 'Creating...' : 'Create Space'}
					</Button>
				</div>
			</form>
		</div>
	</div>
{/if}
