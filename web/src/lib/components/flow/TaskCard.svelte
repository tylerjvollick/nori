<script lang="ts">
	import { page } from '$app/stores';
	import { onDestroy } from 'svelte';
	import type { TaskResponse } from '$lib/types/task';
	import { taskApi } from '$lib/api/task';
	import { spaceMembersStore } from '$lib/stores/spaceMembers';
	import { Badge } from '$lib/components/ui/badge';
	import * as Card from '$lib/components/ui/card';
	import * as Avatar from '$lib/components/ui/avatar';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Clock, User, UserPlus, UserX, Check, Loader2 } from '@lucide/svelte';
	import { formatTimeSpent } from '$lib/utils/time';
	import TaskActions from './TaskActions.svelte';

	interface Props {
		task: TaskResponse;
		spaceId: string;
		stationMap: Map<string, string>;
		/** Called after a successful task action with the updated task */
		onaction?: (updated: TaskResponse) => void;
		/** When true, a drag is in progress — suppress navigation on click. */
		isDragging?: boolean;
		/** When provided, clicking the card calls this instead of navigating. */
		onselect?: (taskId: string) => void;
	}

	let { task, spaceId, stationMap, onaction, isDragging = false, onselect }: Props = $props();

	let slug = $derived($page.params.slug);

	// --- Running timer for active tasks ---

	let now = $state(new Date());
	let tickTimer: ReturnType<typeof setInterval> | null = null;

	$effect(() => {
		// Read task.status in the effect body for Svelte 5 dependency tracking
		const status = task.status;
		if (status === 'active') {
			tickTimer = setInterval(() => {
				now = new Date();
			}, 1_000);
		}

		return () => {
			if (tickTimer) {
				clearInterval(tickTimer);
				tickTimer = null;
			}
		};
	});

	onDestroy(() => {
		if (tickTimer) {
			clearInterval(tickTimer);
			tickTimer = null;
		}
	});

	let timeDisplay = $derived(formatTimeSpent(task.actualTimeSeconds, task.status, task.startedAt, now));
	let isActive = $derived(task.status === 'active');

	// --- Space members for assignment picker ---

	let members = $derived($spaceMembersStore.members);

	/** Get initials for a user. Falls back to first char of email. */
	function getInitials(user: { firstName?: string; lastName?: string; email: string }): string {
		if (user.firstName && user.lastName) {
			return `${user.firstName[0]}${user.lastName[0]}`.toUpperCase();
		}
		if (user.firstName) {
			return user.firstName[0].toUpperCase();
		}
		return user.email[0].toUpperCase();
	}

	/** Get display name for a user. */
	function getDisplayName(user: { firstName?: string; lastName?: string; email: string }): string {
		if (user.firstName && user.lastName) {
			return `${user.firstName} ${user.lastName}`;
		}
		if (user.firstName) {
			return user.firstName;
		}
		return user.email;
	}

	/** Find the assigned member from the members list. */
	let assignedMember = $derived(
		task.assignedToId
			? members.find((m) => m.userId === task.assignedToId)
			: null,
	);

	/** The initials to display on the avatar when assigned. */
	let assignedInitials = $derived(
		assignedMember ? getInitials(assignedMember.user) : null,
	);

	/** The display name for the assigned member (for tooltip). */
	let assignedDisplayName = $derived(
		assignedMember ? getDisplayName(assignedMember.user) : null,
	);

	let isAssigning = $state(false);

	async function assignUser(userId: string): Promise<void> {
		if (isAssigning) return;
		isAssigning = true;
		try {
			const updated = await taskApi.updateTask(spaceId, task.id, { assignedToId: userId });
			onaction?.(updated);
		} catch (err) {
			console.error('Failed to assign user:', err);
		} finally {
			isAssigning = false;
		}
	}

	async function unassignUser(): Promise<void> {
		if (isAssigning) return;
		isAssigning = true;
		try {
			const updated = await taskApi.updateTask(spaceId, task.id, { assignedToId: '' });
			onaction?.(updated);
		} catch (err) {
			console.error('Failed to unassign user:', err);
		} finally {
			isAssigning = false;
		}
	}

	// --- Helpers ---

	const priorityLabels: Record<number, string> = {
		0: 'P0 — Critical',
		1: 'P1 — High',
		2: 'P2 — Medium',
		3: 'P3 — Low',
	};

	function priorityBadge(priority: number): {
		label: string;
		tooltip: string;
		variant: 'default' | 'secondary' | 'destructive' | 'outline';
		class: string;
	} {
		const tooltip = priorityLabels[priority] ?? `P${priority}`;
		switch (priority) {
			case 0:
				return { label: 'P0', tooltip, variant: 'destructive', class: '' };
			case 1:
				return { label: 'P1', tooltip, variant: 'default', class: 'bg-orange-600 text-white border-transparent' };
			case 2:
				return { label: 'P2', tooltip, variant: 'secondary', class: '' };
			case 3:
				return { label: 'P3', tooltip, variant: 'outline', class: '' };
			default:
				return { label: `P${priority}`, tooltip, variant: 'outline', class: '' };
		}
	}

	function getStationName(stationId: string | null | undefined): string | null {
		if (!stationId) return null;
		return stationMap.get(stationId) ?? stationId.slice(0, 8);
	}

	let pBadge = $derived(priorityBadge(task.priority));
	let station = $derived(getStationName(task.stationId));
</script>

<a
	href="/spaces/{slug}/{task.id}"
	class="group block"
	onclick={(e) => { if (isDragging) { e.preventDefault(); return; } if (onselect) { e.preventDefault(); onselect(task.id); } }}
>
	<Card.Root class="p-3 shadow-sm hover:ring-primary/40 hover:shadow-md transition-all rounded-lg" size="sm">
		<Card.Content class="px-0 py-0">
			<!-- Title -->
			<h4 class="text-sm font-medium text-foreground group-hover:text-primary transition-colors truncate">
				{task.title}
			</h4>

			<!-- Hierarchical ID -->
			<p class="mt-0.5 text-xs font-mono text-muted-foreground truncate">{task.id}</p>

			<!-- Badges row -->
			<div class="mt-2 flex flex-wrap items-center gap-1.5">
				<!-- Priority with Tooltip -->
				<Tooltip.Root>
					<Tooltip.Trigger>
						{#snippet child({ props })}
							<span {...props}>
								<Badge variant={pBadge.variant} class="text-[10px] px-1.5 py-0 h-5 {pBadge.class}">
									{pBadge.label}
								</Badge>
							</span>
						{/snippet}
					</Tooltip.Trigger>
					<Tooltip.Content>{pBadge.tooltip}</Tooltip.Content>
				</Tooltip.Root>

				<!-- Station -->
				{#if station}
					<Badge variant="outline" class="text-[10px] px-1.5 py-0 h-5">
						{station}
					</Badge>
				{/if}

				<!-- Quantity -->
				{#if task.quantity > 1}
					<Badge variant="outline" class="text-[10px] px-1.5 py-0 h-5">
						x{task.quantity}
					</Badge>
				{/if}
			</div>

			<!-- Footer: assignee + actions + time -->
			<div class="mt-2 flex items-center justify-between text-[11px] text-muted-foreground">
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					class="flex items-center gap-1"
					onclick={(e) => { e.preventDefault(); e.stopPropagation(); }}
					onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); e.stopPropagation(); } }}
				>
					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							{#snippet child({ props })}
								{#if assignedMember}
									<Tooltip.Root>
										<Tooltip.Trigger>
											{#snippet child({ props: tooltipProps })}
												<button {...props} {...tooltipProps} class="cursor-pointer focus:outline-none focus:ring-1 focus:ring-primary/50 rounded-full">
													<Avatar.Root size="sm" class="size-5 bg-primary/10 text-primary">
														<Avatar.Fallback class="text-[10px] font-medium bg-primary/10 text-primary">
															{assignedInitials}
														</Avatar.Fallback>
													</Avatar.Root>
												</button>
											{/snippet}
										</Tooltip.Trigger>
										<Tooltip.Content>{assignedDisplayName}</Tooltip.Content>
									</Tooltip.Root>
								{:else if task.assignedToId}
									<button {...props} class="inline-flex items-center justify-center size-5 rounded-full bg-primary/10 text-primary cursor-pointer focus:outline-none focus:ring-1 focus:ring-primary/50">
										<User class="size-3" />
									</button>
								{:else}
									<button {...props} class="inline-flex items-center justify-center size-5 rounded-full bg-muted text-muted-foreground hover:bg-primary/10 hover:text-primary cursor-pointer transition-colors focus:outline-none focus:ring-1 focus:ring-primary/50">
										<UserPlus class="size-3" />
									</button>
								{/if}
							{/snippet}
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="start" class="min-w-[180px]">
							<DropdownMenu.Label>Assign to</DropdownMenu.Label>
							<DropdownMenu.Separator />
							{#each members as member (member.userId)}
								<DropdownMenu.Item
									class="gap-2"
									onclick={() => {
										if (task.assignedToId === member.userId) {
											unassignUser();
										} else {
											assignUser(member.userId);
										}
									}}
								>
									<Avatar.Root size="sm" class="size-5 bg-primary/10 text-primary shrink-0">
										<Avatar.Fallback class="text-[10px] font-medium bg-primary/10 text-primary">
											{getInitials(member.user)}
										</Avatar.Fallback>
									</Avatar.Root>
									<span class="truncate">{getDisplayName(member.user)}</span>
									{#if task.assignedToId === member.userId}
										<Check class="size-3.5 ml-auto text-primary shrink-0" />
									{/if}
								</DropdownMenu.Item>
							{/each}
							{#if task.assignedToId}
								<DropdownMenu.Separator />
								<DropdownMenu.Item class="gap-2 text-muted-foreground" onclick={unassignUser}>
									<UserX class="size-4 shrink-0" />
									Unassign
								</DropdownMenu.Item>
							{/if}
							{#if members.length === 0}
								<DropdownMenu.Item disabled>
									<span class="text-muted-foreground">No members</span>
								</DropdownMenu.Item>
							{/if}
						</DropdownMenu.Content>
					</DropdownMenu.Root>
					{#if isAssigning}
						<Loader2 class="size-3 animate-spin text-muted-foreground" />
					{/if}
				</div>
				<div class="flex items-center gap-1.5">
					<TaskActions {task} {spaceId} layout="compact" {onaction} />
					<span class="flex items-center gap-1 {isActive ? 'text-blue-500 dark:text-blue-400' : ''}">
						{#if isActive}
							<span class="relative flex size-2">
								<span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
								<span class="relative inline-flex rounded-full size-2 bg-green-500"></span>
							</span>
						{/if}
						<Clock class="size-3" />
						{timeDisplay}
					</span>
				</div>
			</div>
		</Card.Content>
	</Card.Root>
</a>
