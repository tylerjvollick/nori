<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi, type AdminUser, type CreateUserRequest, type UpdateUserRequest } from '$lib/api/admin';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Badge } from '$lib/components/ui/badge';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Select from '$lib/components/ui/select';
	import { Plus, Pencil, Trash2, Shield, User as UserIcon } from 'lucide-svelte';

	let users = $state<AdminUser[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	// Create user dialog state
	let showCreateDialog = $state(false);
	let createForm = $state<CreateUserRequest>({
		email: '',
		firstName: '',
		lastName: '',
		tempPassword: '',
		role: 'user',
	});
	let createError = $state<string | null>(null);
	let isCreating = $state(false);

	// Edit user dialog state
	let showEditDialog = $state(false);
	let editingUser = $state<AdminUser | null>(null);
	let editForm = $state<UpdateUserRequest>({});
	let editError = $state<string | null>(null);
	let isEditing = $state(false);

	// Delete confirmation state
	let showDeleteDialog = $state(false);
	let deletingUser = $state<AdminUser | null>(null);
	let isDeleting = $state(false);

	// Select value bindings
	let createRoleValue = $state<string>('user');
	let editRoleValue = $state<string>('user');

	onMount(() => {
		loadUsers();
	});

	async function loadUsers() {
		isLoading = true;
		error = null;
		try {
			users = await adminApi.listUsers();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load users';
		} finally {
			isLoading = false;
		}
	}

	async function handleCreateUser() {
		if (isCreating) return;
		isCreating = true;
		createError = null;
		try {
			createForm.role = createRoleValue as 'admin' | 'user';
			await adminApi.createUser(createForm);
			showCreateDialog = false;
			resetCreateForm();
			await loadUsers();
		} catch (e) {
			createError = e instanceof Error ? e.message : 'Failed to create user';
		} finally {
			isCreating = false;
		}
	}

	function resetCreateForm() {
		createForm = { email: '', firstName: '', lastName: '', tempPassword: '', role: 'user' };
		createRoleValue = 'user';
		createError = null;
	}

	function openEditDialog(user: AdminUser) {
		editingUser = user;
		editForm = {
			firstName: user.firstName ?? '',
			lastName: user.lastName ?? '',
			role: user.role ?? 'user',
		};
		editRoleValue = user.role ?? 'user';
		editError = null;
		showEditDialog = true;
	}

	async function handleUpdateUser() {
		if (!editingUser || isEditing) return;
		isEditing = true;
		editError = null;
		try {
			editForm.role = editRoleValue as 'admin' | 'user';
			await adminApi.updateUser(editingUser.id, editForm);
			showEditDialog = false;
			editingUser = null;
			await loadUsers();
		} catch (e) {
			editError = e instanceof Error ? e.message : 'Failed to update user';
		} finally {
			isEditing = false;
		}
	}

	function openDeleteDialog(user: AdminUser) {
		deletingUser = user;
		showDeleteDialog = true;
	}

	async function handleDeleteUser() {
		if (!deletingUser || isDeleting) return;
		isDeleting = true;
		try {
			await adminApi.deleteUser(deletingUser.id);
			showDeleteDialog = false;
			deletingUser = null;
			await loadUsers();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete user';
		} finally {
			isDeleting = false;
		}
	}
</script>

<div class="space-y-6">
	<!-- Page header -->
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-xl font-semibold text-foreground">User Management</h2>
			<p class="text-sm text-muted-foreground">Create, edit, and manage user accounts.</p>
		</div>
		<Button onclick={() => { resetCreateForm(); showCreateDialog = true; }} class="gap-2">
			<Plus class="size-4" />
			Create User
		</Button>
	</div>

	<!-- Error display -->
	{#if error}
		<div class="p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-sm text-destructive">
			{error}
		</div>
	{/if}

	<!-- Loading state -->
	{#if isLoading}
		<div class="text-center py-12 text-muted-foreground">Loading users...</div>
	{:else if users.length === 0}
		<div class="text-center py-12">
			<UserIcon class="size-12 mx-auto text-muted-foreground/50 mb-4" />
			<p class="text-muted-foreground">No users found.</p>
			<p class="text-sm text-muted-foreground mt-1">Create your first user to get started.</p>
		</div>
	{:else}
		<!-- Users table -->
		<div class="border rounded-lg overflow-hidden">
			<table class="w-full">
				<thead>
					<tr class="border-b bg-muted/50">
						<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Name</th>
						<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Email</th>
						<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Role</th>
						<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Status</th>
						<th class="text-right text-sm font-medium text-muted-foreground px-4 py-3">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each users as user (user.id)}
						<tr class="border-b last:border-b-0 hover:bg-muted/30 transition-colors">
							<td class="px-4 py-3">
								<div class="flex items-center gap-3">
									<div class="flex items-center justify-center size-8 rounded-full bg-primary/10 text-primary">
										{#if user.role === 'admin'}
											<Shield class="size-4" />
										{:else}
											<UserIcon class="size-4" />
										{/if}
									</div>
									<span class="text-sm font-medium text-foreground">
										{user.firstName ?? ''} {user.lastName ?? ''}
									</span>
								</div>
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">{user.email}</td>
							<td class="px-4 py-3">
								<Badge variant={user.role === 'admin' ? 'default' : 'secondary'}>
									{user.role ?? 'user'}
								</Badge>
							</td>
							<td class="px-4 py-3">
								{#if user.mustChangePassword}
									<Badge variant="outline">Pending Setup</Badge>
								{:else}
									<Badge variant="secondary">Active</Badge>
								{/if}
							</td>
							<td class="px-4 py-3 text-right">
								<div class="flex items-center justify-end gap-1">
									<Button
										variant="ghost"
										size="sm"
										onclick={() => openEditDialog(user)}
										aria-label="Edit user"
									>
										<Pencil class="size-4" />
									</Button>
									<Button
										variant="ghost"
										size="sm"
										onclick={() => openDeleteDialog(user)}
										aria-label="Delete user"
										class="text-destructive hover:text-destructive"
									>
										<Trash2 class="size-4" />
									</Button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Create User Dialog -->
<Dialog.Root bind:open={showCreateDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Create User</Dialog.Title>
			<Dialog.Description>Add a new user to this account. They will be required to change their password on first login.</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => { e.preventDefault(); handleCreateUser(); }}
			class="space-y-4 mt-4"
		>
			<div class="space-y-2">
				<label for="create-email" class="text-sm font-medium text-foreground">Email</label>
				<Input id="create-email" type="email" bind:value={createForm.email} placeholder="user@example.com" required />
			</div>
			<div class="grid grid-cols-2 gap-4">
				<div class="space-y-2">
					<label for="create-first-name" class="text-sm font-medium text-foreground">First Name</label>
					<Input id="create-first-name" bind:value={createForm.firstName} placeholder="Jane" required />
				</div>
				<div class="space-y-2">
					<label for="create-last-name" class="text-sm font-medium text-foreground">Last Name</label>
					<Input id="create-last-name" bind:value={createForm.lastName} placeholder="Doe" required />
				</div>
			</div>
			<div class="space-y-2">
				<label for="create-password" class="text-sm font-medium text-foreground">Temporary Password</label>
				<Input id="create-password" type="password" bind:value={createForm.tempPassword} placeholder="Temporary password" required />
			</div>
			<div class="space-y-2">
				<span class="text-sm font-medium text-foreground">Role</span>
				<Select.Root type="single" bind:value={createRoleValue}>
					<Select.Trigger class="w-full">
						<span>{createRoleValue === 'admin' ? 'Admin' : 'User'}</span>
					</Select.Trigger>
					<Select.Content>
						<Select.Item value="user" label="User">User</Select.Item>
						<Select.Item value="admin" label="Admin">Admin</Select.Item>
					</Select.Content>
				</Select.Root>
			</div>
			{#if createError}
				<div class="p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-sm text-destructive">
					{createError}
				</div>
			{/if}
			<Dialog.Footer>
				<Button type="button" variant="outline" onclick={() => showCreateDialog = false} disabled={isCreating}>
					Cancel
				</Button>
				<Button type="submit" disabled={isCreating || !createForm.email || !createForm.firstName || !createForm.lastName || !createForm.tempPassword}>
					{isCreating ? 'Creating...' : 'Create User'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

<!-- Edit User Dialog -->
<Dialog.Root bind:open={showEditDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Edit User</Dialog.Title>
			<Dialog.Description>Update user details for {editingUser?.email ?? ''}.</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => { e.preventDefault(); handleUpdateUser(); }}
			class="space-y-4 mt-4"
		>
			<div class="grid grid-cols-2 gap-4">
				<div class="space-y-2">
					<label for="edit-first-name" class="text-sm font-medium text-foreground">First Name</label>
					<Input id="edit-first-name" bind:value={editForm.firstName} placeholder="Jane" />
				</div>
				<div class="space-y-2">
					<label for="edit-last-name" class="text-sm font-medium text-foreground">Last Name</label>
					<Input id="edit-last-name" bind:value={editForm.lastName} placeholder="Doe" />
				</div>
			</div>
			<div class="space-y-2">
				<span class="text-sm font-medium text-foreground">Role</span>
				<Select.Root type="single" bind:value={editRoleValue}>
					<Select.Trigger class="w-full">
						<span>{editRoleValue === 'admin' ? 'Admin' : 'User'}</span>
					</Select.Trigger>
					<Select.Content>
						<Select.Item value="user" label="User">User</Select.Item>
						<Select.Item value="admin" label="Admin">Admin</Select.Item>
					</Select.Content>
				</Select.Root>
			</div>
			{#if editError}
				<div class="p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-sm text-destructive">
					{editError}
				</div>
			{/if}
			<Dialog.Footer>
				<Button type="button" variant="outline" onclick={() => showEditDialog = false} disabled={isEditing}>
					Cancel
				</Button>
				<Button type="submit" disabled={isEditing}>
					{isEditing ? 'Saving...' : 'Save Changes'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

<!-- Delete Confirmation Dialog -->
<Dialog.Root bind:open={showDeleteDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Delete User</Dialog.Title>
			<Dialog.Description>
				Are you sure you want to delete {deletingUser?.firstName ?? ''} {deletingUser?.lastName ?? ''} ({deletingUser?.email})? This action cannot be undone.
			</Dialog.Description>
		</Dialog.Header>
		<Dialog.Footer class="mt-4">
			<Button variant="outline" onclick={() => showDeleteDialog = false} disabled={isDeleting}>
				Cancel
			</Button>
			<Button variant="destructive" onclick={handleDeleteUser} disabled={isDeleting}>
				{isDeleting ? 'Deleting...' : 'Delete User'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
