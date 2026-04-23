<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { spaceStore } from '$lib/stores/space';
	import { authStore } from '$lib/stores/auth';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import * as Collapsible from '$lib/components/ui/collapsible';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { useSidebar } from '$lib/components/ui/sidebar/context.svelte.js';
	import {
		LayoutGrid,
		FileText,
		BookOpen,
		Package,
		Wrench,
		Folder,
		Plus,
		ChevronRight,
		Ellipsis,
		LogOut,
		User as UserIcon,
		Settings,
		Users,
		Key
	} from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Alert from '$lib/components/ui/alert';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { CircleAlert } from '@lucide/svelte';
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
	let newSpaceSlug = $state('');
	let slugTouched = $state(false);
	let isCreating = $state(false);

	/** Auto-generate a slug from the space name (matches backend logic). */
	function generateSlug(name: string): string {
		const trimmed = name.trim();
		if (!trimmed) return '';
		const words = trimmed.split(/\s+/);
		let slug: string;
		if (words.length >= 2) {
			// Take first letter of each word (up to 5)
			slug = words.slice(0, 5).map((w) => w[0]).join('');
		} else {
			// Single word — take first 3 chars
			slug = trimmed.slice(0, 3);
		}
		return slug.toUpperCase().replace(/[^A-Z]/g, '').slice(0, 5);
	}

	// Auto-update slug when name changes (unless user manually edited the slug)
	$effect(() => {
		if (!slugTouched) {
			newSpaceSlug = generateSlug(newSpaceName);
		}
	});

	// Server-provided user with accessibleSpaces (always fresh from hooks.server.ts)
	let pageUser = $derived($page.data.user);

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
		const pathname = $page.url.pathname;
		return pathname === href || pathname.startsWith(href + '/');
	}

	async function handleCreateSpace() {
		if (!newSpaceName.trim() || isCreating) return;

		isCreating = true;
		const slugToSend = newSpaceSlug.trim() || undefined;
		const space = await spaceStore.createSpace(newSpaceName.trim(), slugToSend);
		isCreating = false;

		if (space) {
			newSpaceName = '';
			newSpaceSlug = '';
			slugTouched = false;
			showCreateDialog = false;
			spaceStore.loadRecentSpaces();
			goto(`/spaces/${space.slug}`);
		}
	}

	function handleLogout() {
		authStore.logout();
		goto('/login');
	}

	// Admin role check
	let isAdmin = $derived(pageUser?.role === 'admin');

	// Navigation structure
	const navMain = [
		{
			title: 'SOPs',
			url: '/sops',
			icon: FileText,
			isActive: isActive('/sops')
		},
		{
			title: 'Recipes',
			url: '/recipes',
			icon: BookOpen,
			isActive: isActive('/recipes')
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

	const navAdmin = [
		{
			title: 'Users',
			url: '/admin/users',
			icon: Users
		},
		{
			title: 'API Keys',
			url: '/admin/api-keys',
			icon: Key
		},
		{
			title: 'Spaces',
			url: '/admin/spaces',
			icon: LayoutGrid
		}
	];
</script>

<Sidebar.Root {collapsible} {...restProps} bind:ref>
	<Sidebar.Header>
		<div class="flex items-center gap-2 px-2 group-data-[collapsible=icon]:justify-center h-(--header-height)">
			<img
				src="/nori_logo.png"
				alt="Nori"
				class="size-8 shrink-0 rounded-lg"
			/>
			<span class="font-bold text-lg text-sidebar-foreground group-data-[collapsible=icon]:hidden">Nori</span>
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
													isActive={isActive(`/spaces/${space.slug}`)}
													onclick={() => goto(`/spaces/${space.slug}`)}
												>
													{#snippet child({ props })}
														<a href={`/spaces/${space.slug}`} {...props}>
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

		<!-- Admin Section (only visible to admins) -->
		{#if isAdmin}
			<Sidebar.Separator />

			<Sidebar.Group class="group-data-[collapsible=icon]:hidden">
				<Sidebar.GroupLabel>
					<Settings class="size-3 mr-1" />
					Admin
				</Sidebar.GroupLabel>
				<Sidebar.Menu>
					{#each navAdmin as item (item.title)}
						<Sidebar.MenuItem>
							<Sidebar.MenuButton
								isActive={isActive(item.url)}
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
		{/if}
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
											>{user?.firstName ?? ''} {user?.lastName ?? ''}</span
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
<Dialog.Root bind:open={showCreateDialog} onOpenChange={(open) => {
	if (!open) {
		newSpaceName = '';
		newSpaceSlug = '';
		slugTouched = false;
		spaceStore.clearError();
	}
}}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Create New Space</Dialog.Title>
			<Dialog.Description>Add a new workspace for your team.</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleCreateSpace();
			}}
		>
			<div class="grid gap-4 py-2">
				<div class="grid gap-2">
					<Label for="space-name">Space Name</Label>
					<Input
						id="space-name"
						type="text"
						bind:value={newSpaceName}
						placeholder="e.g., Marketing, Engineering, HR"
						disabled={isCreating}
					/>
				</div>
				<div class="grid gap-2">
					<Label for="space-slug">Slug</Label>
					<p class="text-xs text-muted-foreground">
						2-5 uppercase letters used in URLs. Auto-generated from the name.
					</p>
					<Input
						id="space-slug"
						type="text"
						bind:value={newSpaceSlug}
						oninput={() => (slugTouched = true)}
						placeholder="e.g., MKT, ENG, HR"
						maxlength={5}
						class="font-mono uppercase placeholder:normal-case"
						disabled={isCreating}
					/>
				</div>
				{#if $spaceStore.error}
					<Alert.Root variant="destructive">
						<CircleAlert class="size-4" />
						<Alert.Description>{$spaceStore.error}</Alert.Description>
					</Alert.Root>
				{/if}
			</div>
			<Dialog.Footer class="pt-2">
				<Button
					type="button"
					variant="outline"
					onclick={() => {
						showCreateDialog = false;
					}}
					disabled={isCreating}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={!newSpaceName.trim() || isCreating}>
					{isCreating ? 'Creating...' : 'Create Space'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
