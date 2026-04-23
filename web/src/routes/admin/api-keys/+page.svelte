<script lang="ts">
	import { onMount } from 'svelte';
	import { adminApi, type APIKey, type CreateAPIKeyRequest } from '$lib/api/admin';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Badge } from '$lib/components/ui/badge';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Plus, Trash2, Key, Copy, Check } from '@lucide/svelte';

	let apiKeys = $state<APIKey[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	// Create dialog state
	let showCreateDialog = $state(false);
	let createName = $state('');
	let createExpiresAt = $state('');
	let createError = $state<string | null>(null);
	let isCreating = $state(false);

	// Raw key display state (shown once after creation)
	let showRawKeyDialog = $state(false);
	let rawKey = $state('');
	let copied = $state(false);

	// Revoke confirmation state
	let showRevokeDialog = $state(false);
	let revokingKey = $state<APIKey | null>(null);
	let isRevoking = $state(false);

	onMount(() => {
		loadAPIKeys();
	});

	async function loadAPIKeys() {
		isLoading = true;
		error = null;
		try {
			apiKeys = await adminApi.listAPIKeys();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load API keys';
		} finally {
			isLoading = false;
		}
	}

	async function handleCreateKey() {
		if (isCreating || !createName.trim()) return;
		isCreating = true;
		createError = null;
		try {
			const data: CreateAPIKeyRequest = { name: createName.trim() };
			if (createExpiresAt) {
				data.expiresAt = new Date(createExpiresAt).toISOString();
			}
			const response = await adminApi.createAPIKey(data);
			showCreateDialog = false;
			rawKey = response.rawKey;
			showRawKeyDialog = true;
			createName = '';
			createExpiresAt = '';
			await loadAPIKeys();
		} catch (e) {
			createError = e instanceof Error ? e.message : 'Failed to create API key';
		} finally {
			isCreating = false;
		}
	}

	async function copyRawKey() {
		try {
			await navigator.clipboard.writeText(rawKey);
			copied = true;
			setTimeout(() => { copied = false; }, 2000);
		} catch {
			// Fallback: select the text
		}
	}

	function openRevokeDialog(key: APIKey) {
		revokingKey = key;
		showRevokeDialog = true;
	}

	async function handleRevokeKey() {
		if (!revokingKey || isRevoking) return;
		isRevoking = true;
		try {
			await adminApi.revokeAPIKey(revokingKey.id);
			showRevokeDialog = false;
			revokingKey = null;
			await loadAPIKeys();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to revoke API key';
		} finally {
			isRevoking = false;
		}
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return 'Never';
		return new Date(dateStr).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
		});
	}
</script>

<div class="space-y-6">
	<!-- Page header -->
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-xl font-semibold text-foreground">API Key Management</h2>
			<p class="text-sm text-muted-foreground">Create and manage API keys for programmatic access.</p>
		</div>
		<Button onclick={() => { createName = ''; createExpiresAt = ''; createError = null; showCreateDialog = true; }} class="gap-2">
			<Plus class="size-4" />
			Create API Key
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
		<div class="text-center py-12 text-muted-foreground">Loading API keys...</div>
	{:else if apiKeys.length === 0}
		<div class="text-center py-12">
			<Key class="size-12 mx-auto text-muted-foreground/50 mb-4" />
			<p class="text-muted-foreground">No API keys found.</p>
			<p class="text-sm text-muted-foreground mt-1">Create an API key for programmatic access.</p>
		</div>
	{:else}
		<!-- API Keys table -->
		<div class="border rounded-lg overflow-hidden">
			<table class="w-full">
				<thead>
					<tr class="border-b bg-muted/50">
						<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Name</th>
						<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Status</th>
						<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Created</th>
						<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Last Used</th>
						<th class="text-left text-sm font-medium text-muted-foreground px-4 py-3">Expires</th>
						<th class="text-right text-sm font-medium text-muted-foreground px-4 py-3">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each apiKeys as key (key.id)}
						<tr class="border-b last:border-b-0 hover:bg-muted/30 transition-colors">
							<td class="px-4 py-3">
								<div class="flex items-center gap-3">
									<Key class="size-4 text-muted-foreground" />
									<span class="text-sm font-medium text-foreground">{key.name}</span>
								</div>
							</td>
							<td class="px-4 py-3">
								{#if key.isActive}
									<Badge variant="secondary">Active</Badge>
								{:else}
									<Badge variant="outline">Revoked</Badge>
								{/if}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatDate(key.createdAt)}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{formatDate(key.lastUsedAt)}
							</td>
							<td class="px-4 py-3 text-sm text-muted-foreground">
								{key.expiresAt ? formatDate(key.expiresAt) : 'Never'}
							</td>
							<td class="px-4 py-3 text-right">
								{#if key.isActive}
									<Button
										variant="ghost"
										size="sm"
										onclick={() => openRevokeDialog(key)}
										aria-label="Revoke API key"
										class="text-destructive hover:text-destructive"
									>
										<Trash2 class="size-4" />
									</Button>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Create API Key Dialog -->
<Dialog.Root bind:open={showCreateDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Create API Key</Dialog.Title>
			<Dialog.Description>Create a new API key for programmatic access. The key will only be shown once.</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => { e.preventDefault(); handleCreateKey(); }}
			class="space-y-4 mt-4"
		>
			<div class="space-y-2">
				<label for="key-name" class="text-sm font-medium text-foreground">Name</label>
				<Input id="key-name" bind:value={createName} placeholder="e.g., CI/CD Pipeline" required />
			</div>
			<div class="space-y-2">
				<label for="key-expires" class="text-sm font-medium text-foreground">Expires At (optional)</label>
				<Input id="key-expires" type="date" bind:value={createExpiresAt} />
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
				<Button type="submit" disabled={isCreating || !createName.trim()}>
					{isCreating ? 'Creating...' : 'Create Key'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

<!-- Raw Key Display Dialog -->
<Dialog.Root bind:open={showRawKeyDialog}>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title>API Key Created</Dialog.Title>
			<Dialog.Description>Copy your API key now. You will not be able to see it again.</Dialog.Description>
		</Dialog.Header>
		<div class="mt-4 space-y-4">
			<div class="flex items-center gap-2">
				<code class="flex-1 p-3 bg-muted rounded-lg text-sm font-mono break-all select-all">
					{rawKey}
				</code>
				<Button variant="outline" size="sm" onclick={copyRawKey} class="shrink-0">
					{#if copied}
						<Check class="size-4" />
					{:else}
						<Copy class="size-4" />
					{/if}
				</Button>
			</div>
			<div class="p-3 bg-amber-500/10 border border-amber-500/20 rounded-lg text-sm text-amber-700 dark:text-amber-400">
				Make sure to copy this key. It will not be shown again.
			</div>
		</div>
		<Dialog.Footer class="mt-4">
			<Button onclick={() => { showRawKeyDialog = false; rawKey = ''; }}>
				Done
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Revoke Confirmation Dialog -->
<Dialog.Root bind:open={showRevokeDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Revoke API Key</Dialog.Title>
			<Dialog.Description>
				Are you sure you want to revoke the API key "{revokingKey?.name}"? Any integrations using this key will stop working.
			</Dialog.Description>
		</Dialog.Header>
		<Dialog.Footer class="mt-4">
			<Button variant="outline" onclick={() => showRevokeDialog = false} disabled={isRevoking}>
				Cancel
			</Button>
			<Button variant="destructive" onclick={handleRevokeKey} disabled={isRevoking}>
				{isRevoking ? 'Revoking...' : 'Revoke Key'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
