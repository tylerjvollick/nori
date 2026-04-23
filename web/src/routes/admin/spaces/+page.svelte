<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { adminApi, type AdminUser, type SpaceMember } from '$lib/api/admin';
	import { spaceApi, type Space } from '$lib/api/space';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Select from '$lib/components/ui/select';
	import { Plus, Trash2, LayoutGrid, Users, User as UserIcon } from '@lucide/svelte';

	let spaces = $state<Space[]>([]);
	let selectedSpaceId = $state<string>('');
	let members = $state<SpaceMember[]>([]);
	let allUsers = $state<AdminUser[]>([]);
	let isLoadingSpaces = $state(true);
	let isLoadingMembers = $state(false);
	let error = $state<string | null>(null);

	// Add member dialog state
	let showAddDialog = $state(false);
	let selectedUserId = $state<string>('');
	let addError = $state<string | null>(null);
	let isAdding = $state(false);

	// Remove member dialog state
	let showRemoveDialog = $state(false);
	let removingMember = $state<SpaceMember | null>(null);
	let isRemoving = $state(false);

	// Derive non-member users (available to add)
	let availableUsers = $derived(
		allUsers.filter((u) => !members.some((m) => m.userId === u.id))
	);

	onMount(async () => {
		await loadSpaces();
		await loadAllUsers();
	});

	async function loadSpaces() {
		isLoadingSpaces = true;
		error = null;
		try {
			spaces = await spaceApi.getAll();
			// Auto-select the first space if available
			if (spaces.length > 0 && !selectedSpaceId) {
				selectedSpaceId = spaces[0].id;
				await loadMembers();
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load spaces';
		} finally {
			isLoadingSpaces = false;
		}
	}

	async function loadAllUsers() {
		try {
			allUsers = await adminApi.listUsers();
		} catch (e) {
			// Non-critical, will just limit add-member functionality
		}
	}

	async function loadMembers() {
		if (!selectedSpaceId) return;
		isLoadingMembers = true;
		error = null;
		try {
			members = await adminApi.getSpaceMembers(selectedSpaceId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load space members';
		} finally {
			isLoadingMembers = false;
		}
	}

	async function handleSpaceChange(spaceId: string) {
		selectedSpaceId = spaceId;
		await loadMembers();
	}

	async function handleAddMember() {
		if (isAdding || !selectedUserId || !selectedSpaceId) return;
		isAdding = true;
		addError = null;
		try {
			await adminApi.addSpaceMember(selectedSpaceId, { userId: selectedUserId });
			showAddDialog = false;
			selectedUserId = '';
			await loadMembers();
		} catch (e) {
			addError = e instanceof Error ? e.message : 'Failed to add member';
		} finally {
			isAdding = false;
		}
	}

	function openRemoveDialog(member: SpaceMember) {
		removingMember = member;
		showRemoveDialog = true;
	}

	async function handleRemoveMember() {
		if (!removingMember || isRemoving || !selectedSpaceId) return;
		isRemoving = true;
		try {
			await adminApi.removeSpaceMember(selectedSpaceId, removingMember.userId);
			showRemoveDialog = false;
			removingMember = null;
			await loadMembers();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to remove member';
		} finally {
			isRemoving = false;
		}
	}

	function getSelectedSpaceName(): string {
		return spaces.find((s) => s.id === selectedSpaceId)?.name ?? 'Select a space';
	}
</script>

<div class="space-y-6">
	<!-- Page header -->
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-xl font-semibold text-foreground">Space Members</h2>
			<p class="text-sm text-muted-foreground">Manage user membership within each space.</p>
		</div>
		{#if selectedSpaceId}
			<Button onclick={() => { selectedUserId = ''; addError = null; showAddDialog = true; }} class="gap-2">
				<Plus class="size-4" />
				Add Member
			</Button>
		{/if}
	</div>

	<!-- Error display -->
	{#if error}
		<div class="p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-sm text-destructive">
			{error}
		</div>
	{/if}

	<!-- Space selector -->
	{#if isLoadingSpaces}
		<div class="text-center py-12 text-muted-foreground">Loading spaces...</div>
	{:else if spaces.length === 0}
		<div class="text-center py-12">
			<LayoutGrid class="size-12 mx-auto text-muted-foreground/50 mb-4" />
			<p class="text-muted-foreground">No spaces found.</p>
			<p class="text-sm text-muted-foreground mt-1">Create a space first to manage its members.</p>
		</div>
	{:else}
		<div class="flex items-center gap-3">
			<span class="text-sm font-medium text-foreground">Space:</span>
			<Select.Root type="single" value={selectedSpaceId} onValueChange={(v) => { if (v) handleSpaceChange(v); }}>
				<Select.Trigger class="w-64">
					<span>{getSelectedSpaceName()}</span>
				</Select.Trigger>
				<Select.Content>
					{#each spaces as space (space.id)}
						<Select.Item value={space.id} label={space.name}>{space.name}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>

		<!-- Members list -->
		{#if isLoadingMembers}
			<div class="text-center py-12 text-muted-foreground">Loading members...</div>
		{:else if members.length === 0 && selectedSpaceId}
			<div class="text-center py-12">
				<Users class="size-12 mx-auto text-muted-foreground/50 mb-4" />
				<p class="text-muted-foreground">No members in this space.</p>
				<p class="text-sm text-muted-foreground mt-1">Add users to grant them access to this space.</p>
			</div>
		{:else}
			<div class="border rounded-lg overflow-hidden">
				<table class="w-full">
					<thead>
						<tr class="border-b bg-muted/50">
							<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">User</th>
							<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Email</th>
							<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Role</th>
							<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Added</th>
							<th class="text-right text-sm font-medium text-muted-foreground px-4 py-3">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each members as member (member.id)}
							<tr class="border-b last:border-b-0 hover:bg-muted/30 transition-colors">
								<td class="px-4 py-3">
									<div class="flex items-center gap-3">
										<div class="flex items-center justify-center size-8 rounded-full bg-primary/10 text-primary">
											<UserIcon class="size-4" />
										</div>
										<span class="text-sm font-medium text-foreground">
											{member.user.firstName ?? ''} {member.user.lastName ?? ''}
										</span>
									</div>
								</td>
								<td class="px-4 py-3 text-sm text-muted-foreground">{member.user.email}</td>
								<td class="px-4 py-3">
									<Badge variant={member.user.role === 'admin' ? 'default' : 'secondary'}>
										{member.user.role ?? 'user'}
									</Badge>
								</td>
								<td class="px-4 py-3 text-sm text-muted-foreground">
									{new Date(member.createdAt).toLocaleDateString()}
								</td>
								<td class="px-4 py-3 text-right">
									<Button
										variant="ghost"
										size="sm"
										onclick={() => openRemoveDialog(member)}
										aria-label="Remove member"
										class="text-destructive hover:text-destructive"
									>
										<Trash2 class="size-4" />
									</Button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</div>

<!-- Add Member Dialog -->
<Dialog.Root bind:open={showAddDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Add Member</Dialog.Title>
			<Dialog.Description>Add a user to the "{getSelectedSpaceName()}" space.</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => { e.preventDefault(); handleAddMember(); }}
			class="space-y-4 mt-4"
		>
			<div class="space-y-2">
				<span class="text-sm font-medium text-foreground">User</span>
				{#if availableUsers.length === 0}
					<p class="text-sm text-muted-foreground">All users are already members of this space.</p>
				{:else}
					<Select.Root type="single" bind:value={selectedUserId}>
						<Select.Trigger class="w-full">
							<span>
								{#if selectedUserId}
									{@const u = allUsers.find((u) => u.id === selectedUserId)}
									{u?.firstName ?? ''} {u?.lastName ?? ''} ({u?.email})
								{:else}
									Select a user...
								{/if}
							</span>
						</Select.Trigger>
						<Select.Content>
							{#each availableUsers as user (user.id)}
								<Select.Item value={user.id} label="{user.firstName ?? ''} {user.lastName ?? ''}">
									{user.firstName ?? ''} {user.lastName ?? ''} ({user.email})
								</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				{/if}
			</div>
			{#if addError}
				<div class="p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-sm text-destructive">
					{addError}
				</div>
			{/if}
			<Dialog.Footer>
				<Button type="button" variant="outline" onclick={() => showAddDialog = false} disabled={isAdding}>
					Cancel
				</Button>
				<Button type="submit" disabled={isAdding || !selectedUserId || availableUsers.length === 0}>
					{isAdding ? 'Adding...' : 'Add Member'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

<!-- Remove Member Confirmation Dialog -->
<Dialog.Root bind:open={showRemoveDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Remove Member</Dialog.Title>
			<Dialog.Description>
				Are you sure you want to remove {removingMember?.user.firstName ?? ''} {removingMember?.user.lastName ?? ''} from the "{getSelectedSpaceName()}" space? They will lose access to this space.
			</Dialog.Description>
		</Dialog.Header>
		<Dialog.Footer class="mt-4">
			<Button variant="outline" onclick={() => showRemoveDialog = false} disabled={isRemoving}>
				Cancel
			</Button>
			<Button variant="destructive" onclick={handleRemoveMember} disabled={isRemoving}>
				{isRemoving ? 'Removing...' : 'Remove Member'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
