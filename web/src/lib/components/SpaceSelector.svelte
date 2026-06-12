<script lang="ts">
	import { goto } from '$app/navigation';
	import { spaceApi, type Space } from '$lib/api/space';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Sidebar from '$lib/components/ui/sidebar';
	import { useSidebar } from '$lib/components/ui/sidebar/context.svelte.js';
	import { ChevronsUpDown, Check, Plus } from '@lucide/svelte';
	import type { User } from '$lib/api/auth';

	let { user, onCreateSpace }: { user: User; onCreateSpace?: () => void } = $props();

	const sidebar = useSidebar();

	let activeSpace = $derived(
		user.accessibleSpaces.find((s: Space) => s.id === user.activeSpaceId) ?? null
	);

	async function selectSpace(space: Space) {
		if (space.id === user.activeSpaceId) return;

		// Record visit to update recent spaces
		try {
			await spaceApi.recordVisit(space.id);
		} catch {
			// Best-effort — don't block navigation
		}

		// Navigate to the selected space
		goto(`/spaces/${space.slug}`);
	}
</script>

<Sidebar.Menu>
	<Sidebar.MenuItem>
		<DropdownMenu.Root>
			<DropdownMenu.Trigger>
				{#snippet child({ props: triggerProps })}
					<Sidebar.MenuButton
						size="lg"
						class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
						tooltipContent={activeSpace?.name ?? 'Select space'}
						{...triggerProps}
					>
						<div
							class="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground"
						>
							<span class="text-sm font-bold">
								{activeSpace ? activeSpace.name.charAt(0).toUpperCase() : '?'}
							</span>
						</div>
						<div class="grid flex-1 text-left text-sm leading-tight">
							<span class="truncate font-semibold">
								{activeSpace?.name ?? 'No space selected'}
							</span>
							<span class="truncate text-xs text-muted-foreground">
								{user.accessibleSpaces.length} space{user.accessibleSpaces.length !== 1 ? 's' : ''}
							</span>
						</div>
						<ChevronsUpDown class="ml-auto size-4" />
					</Sidebar.MenuButton>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content
				class="w-[--bits-dropdown-menu-trigger-width] min-w-56 rounded-lg"
				side={sidebar.isMobile ? 'bottom' : 'right'}
				align="start"
				sideOffset={4}
			>
				<DropdownMenu.Label class="text-xs text-muted-foreground">
					Spaces
				</DropdownMenu.Label>
				{#each user.accessibleSpaces as space (space.id)}
					<DropdownMenu.Item onclick={() => selectSpace(space)}>
						<div class="flex size-6 items-center justify-center rounded-sm border">
							<span class="text-xs font-medium">
								{space.name.charAt(0).toUpperCase()}
							</span>
						</div>
						<span class="flex-1 truncate">{space.name}</span>
						{#if space.id === user.activeSpaceId}
							<Check class="ml-auto size-4 text-sidebar-primary" />
						{/if}
					</DropdownMenu.Item>
				{/each}
				{#if user.accessibleSpaces.length === 0}
					<DropdownMenu.Item disabled>
						<span class="text-muted-foreground">No spaces available</span>
					</DropdownMenu.Item>
				{/if}
				{#if onCreateSpace}
					<DropdownMenu.Separator />
					<DropdownMenu.Item onclick={onCreateSpace}>
						<Plus class="size-4" />
						<span>Create space</span>
					</DropdownMenu.Item>
				{/if}
			</DropdownMenu.Content>
		</DropdownMenu.Root>
	</Sidebar.MenuItem>
</Sidebar.Menu>
