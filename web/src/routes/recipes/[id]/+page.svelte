<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { recipeStore } from '$lib/stores/recipe';
	import { recipeApi } from '$lib/api/recipe';
	import { taskApi } from '$lib/api/task';
	import type { TaskTreeResponse, TaskDepsResponse } from '$lib/api/task';
	import type { TaskResponse } from '$lib/types/task';
	import type { RecipeVersionResponse } from '$lib/types/recipe';
	import { stationApi } from '$lib/api/station';
	import type { StationResponse } from '$lib/types/station';
	import { customerApi } from '$lib/api/customer';
	import type { CustomerResponse } from '$lib/api/customer';
	import TaskTreeEditor from '$lib/components/flow/TaskTreeEditor.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Separator } from '$lib/components/ui/separator';
	import * as Alert from '$lib/components/ui/alert';
	import * as Card from '$lib/components/ui/card';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Select from '$lib/components/ui/select';
	import {
		CircleAlert,
		ArrowLeft,
		Send,
		Play,
		Plus,
		ChevronDown,
		History,
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	let recipeId = $derived($page.params.id);
	let recipe = $derived($recipeStore.currentRecipe);
	let loading = $derived($recipeStore.loading);
	let storeError = $derived($recipeStore.error);

	// Task tree state
	let tree = $state<TaskTreeResponse | null>(null);
	let stations = $state<StationResponse[]>([]);
	let stationMap = $state<Map<string, string>>(new Map());
	let depsMap = $state<Map<string, TaskDepsResponse>>(new Map());
	let treeLoading = $state(false);

	// Roll dialog state
	let showRollDialog = $state(false);
	let rollTitle = $state('');
	let rollOrderQty = $state(1);
	let rollCustomerId = $state('');
	let rollDueDate = $state('');
	let isRolling = $state(false);

	// Customer data
	let customers = $state<CustomerResponse[]>([]);

	// Publish state
	let isPublishing = $state(false);

	// New version dialog state
	let showNewVersionDialog = $state(false);
	let newVersionSummary = $state('');
	let isCreatingVersion = $state(false);

	// Version history state
	let showVersionHistory = $state(false);
	let versions = $state<RecipeVersionResponse[]>([]);

	// Derived recipe properties
	let currentVersion = $derived(recipe?.currentVersion);
	let isDraft = $derived(currentVersion?.status === 'draft');
	let isPublished = $derived(currentVersion?.status === 'published');
	let hasTaskTree = $derived(currentVersion?.rootTaskId != null);

	let lastLoadedId = $state<string | null>(null);

	$effect(() => {
		const id = $page.params.id;
		if (id && id !== lastLoadedId) {
			lastLoadedId = id;
			recipeStore.loadRecipe(id);
		}
	});

	// Load task tree when recipe and rootTaskId are available
	$effect(() => {
		const ver = currentVersion;
		if (ver?.rootTaskId && ver.rootTaskId !== tree?.id) {
			loadTaskTree(ver.rootTaskId);
		}
	});

	onMount(() => {
		loadStations();
		loadCustomers();
		return () => recipeStore.clearCurrent();
	});

	async function loadStations() {
		try {
			stations = await stationApi.listStations();
			const map = new Map<string, string>();
			for (const s of stations) {
				map.set(s.id, s.name);
			}
			stationMap = map;
		} catch {
			// Stations are optional for display
		}
	}

	async function loadCustomers() {
		try {
			customers = await customerApi.listCustomers();
		} catch {
			// Customers are optional for the roll dialog
		}
	}

	async function loadTaskTree(rootTaskId: string) {
		treeLoading = true;
		try {
			tree = await taskApi.getTaskTree(rootTaskId);
			await loadDepsForTree();
		} catch (err) {
			console.error('Failed to load task tree:', err);
		} finally {
			treeLoading = false;
		}
	}

	async function loadDepsForTree() {
		if (!tree) return;
		const tasks = flattenTree(tree);
		const map = new Map<string, TaskDepsResponse>();

		const BATCH_SIZE = 20;
		for (let i = 0; i < tasks.length; i += BATCH_SIZE) {
			const batch = tasks.slice(i, i + BATCH_SIZE);
			const results = await Promise.all(
				batch.map(async (t) => {
					try {
						const deps = await taskApi.getTaskDeps(t.id);
						return { id: t.id, deps };
					} catch {
						return { id: t.id, deps: { blockers: [], dependents: [] } as TaskDepsResponse };
					}
				}),
			);
			for (const r of results) {
				map.set(r.id, r.deps);
			}
		}
		depsMap = map;
	}

	function flattenTree(node: TaskTreeResponse): TaskResponse[] {
		const result: TaskResponse[] = [];
		function walk(n: TaskTreeResponse): void {
			const { children: _, ...task } = n;
			result.push(task as TaskResponse);
			if (n.children) {
				for (const child of n.children) walk(child);
			}
		}
		walk(node);
		return result;
	}

	async function handleTreeMutate() {
		// Reload the tree after edits
		if (currentVersion?.rootTaskId) {
			await loadTaskTree(currentVersion.rootTaskId);
		}
	}

	async function handlePublish() {
		if (!recipe || isPublishing) return;
		isPublishing = true;
		try {
			await recipeStore.publishVersion(recipe.id);
			toast.success('Recipe published successfully');
			// Reload tree for the new published version
			if ($recipeStore.currentRecipe?.currentVersion?.rootTaskId) {
				await loadTaskTree($recipeStore.currentRecipe.currentVersion.rootTaskId);
			}
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to publish');
		} finally {
			isPublishing = false;
		}
	}

	async function handleRoll() {
		if (!recipe || isRolling) return;
		isRolling = true;
		try {
			const job = await recipeApi.rollRecipe(recipe.id, {
				title: rollTitle.trim() || undefined,
				order_qty: rollOrderQty > 1 ? rollOrderQty : undefined,
				customer_id: rollCustomerId || undefined,
				due_date: rollDueDate ? new Date(rollDueDate).toISOString() : undefined,
			});
			showRollDialog = false;
			rollTitle = '';
			rollOrderQty = 1;
			rollCustomerId = '';
			rollDueDate = '';
			toast.success('Job created from recipe');
			// Navigate to the new job
			goto(`/spaces/${$page.params.slug || 'default'}/${job.id}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to roll recipe');
		} finally {
			isRolling = false;
		}
	}

	async function handleNewVersion() {
		if (!recipe || isCreatingVersion) return;
		isCreatingVersion = true;
		try {
			await recipeStore.createNewVersion(recipe.id, newVersionSummary.trim() || undefined);
			showNewVersionDialog = false;
			newVersionSummary = '';
			toast.success('New draft version created');
			// Reload tree for the new draft
			if ($recipeStore.currentRecipe?.currentVersion?.rootTaskId) {
				await loadTaskTree($recipeStore.currentRecipe.currentVersion.rootTaskId);
			}
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to create new version');
		} finally {
			isCreatingVersion = false;
		}
	}

	async function loadVersionHistory() {
		if (!recipe) return;
		showVersionHistory = !showVersionHistory;
		if (showVersionHistory) {
			try {
				versions = await recipeApi.listVersions(recipe.id);
			} catch {
				versions = [];
			}
		}
	}

	function formatDate(dateString: string | null | undefined): string {
		if (!dateString) return 'N/A';
		return new Date(dateString).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
		});
	}

	function getStatusBadge(status: string) {
		switch (status) {
			case 'published':
				return { variant: 'secondary' as const, class: 'bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 border-green-200 dark:border-green-800', label: 'Published' };
			case 'draft':
				return { variant: 'outline' as const, class: 'bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 border-yellow-200 dark:border-yellow-800', label: 'Draft' };
			case 'archived':
				return { variant: 'outline' as const, class: 'bg-gray-50 dark:bg-gray-950 text-gray-500 dark:text-gray-400', label: 'Archived' };
			default:
				return { variant: 'outline' as const, class: '', label: status };
		}
	}
</script>

<div class="container mx-auto px-4 py-8">
	<!-- Back link -->
	<div class="mb-4">
		<Button variant="ghost" size="sm" href="/recipes">
			<ArrowLeft class="size-4 mr-1" />
			Back to Recipes
		</Button>
	</div>

	{#if loading && !recipe}
		<!-- Loading skeleton -->
		<div class="space-y-6">
			<div class="flex justify-between items-start">
				<div class="space-y-2">
					<Skeleton class="h-8 w-64" />
					<Skeleton class="h-4 w-96" />
				</div>
				<div class="flex gap-2">
					<Skeleton class="h-9 w-24" />
					<Skeleton class="h-9 w-24" />
				</div>
			</div>
			<Skeleton class="h-96 w-full" />
		</div>
	{:else if storeError}
		<Alert.Root variant="destructive">
			<CircleAlert />
			<Alert.Title>Error</Alert.Title>
			<Alert.Description>{storeError}</Alert.Description>
		</Alert.Root>
	{:else if recipe}
		<!-- Header -->
		<div class="flex flex-col md:flex-row justify-between items-start gap-4 mb-6">
			<div class="space-y-1">
				<div class="flex items-center gap-3">
					<h1 class="text-3xl font-bold text-foreground">{recipe.name}</h1>
					{#if currentVersion}
						{@const badge = getStatusBadge(currentVersion.status)}
						<Badge variant={badge.variant} class={badge.class}>
							{badge.label}
						</Badge>
						<Badge variant="secondary">v{currentVersion.versionNumber}</Badge>
					{/if}
				</div>
				{#if recipe.description}
					<p class="text-muted-foreground">{recipe.description}</p>
				{/if}
			</div>

			<!-- Actions -->
			<div class="flex items-center gap-2 shrink-0">
				{#if isDraft}
					<Button onclick={handlePublish} disabled={isPublishing}>
						<Send class="size-4 mr-1" />
						{isPublishing ? 'Publishing...' : 'Publish'}
					</Button>
				{/if}

				{#if isPublished}
					<Button onclick={() => (showRollDialog = true)}>
						<Play class="size-4 mr-1" />
						Roll
					</Button>
					<Button variant="outline" onclick={() => (showNewVersionDialog = true)}>
						<Plus class="size-4 mr-1" />
						New Version
					</Button>
				{/if}

				<Button variant="ghost" size="sm" onclick={loadVersionHistory}>
					<History class="size-4 mr-1" />
					History
				</Button>
			</div>
		</div>

		<Separator class="mb-6" />

		<!-- Recipe metadata -->
		<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
			<Card.Root>
				<Card.Content class="pt-4 pb-4 px-4">
					<p class="text-xs text-muted-foreground uppercase tracking-wider">Version</p>
					<p class="text-lg font-semibold">{currentVersion?.versionNumber ?? '-'}</p>
				</Card.Content>
			</Card.Root>
			<Card.Root>
				<Card.Content class="pt-4 pb-4 px-4">
					<p class="text-xs text-muted-foreground uppercase tracking-wider">Status</p>
					<p class="text-lg font-semibold capitalize">{currentVersion?.status ?? 'No version'}</p>
				</Card.Content>
			</Card.Root>
			<Card.Root>
				<Card.Content class="pt-4 pb-4 px-4">
					<p class="text-xs text-muted-foreground uppercase tracking-wider">Created</p>
					<p class="text-sm font-medium">{formatDate(recipe.createdAt)}</p>
				</Card.Content>
			</Card.Root>
			<Card.Root>
				<Card.Content class="pt-4 pb-4 px-4">
					<p class="text-xs text-muted-foreground uppercase tracking-wider">Updated</p>
					<p class="text-sm font-medium">{formatDate(recipe.updatedAt)}</p>
				</Card.Content>
			</Card.Root>
		</div>

		<!-- Version History (collapsible) -->
		{#if showVersionHistory}
			<Card.Root class="mb-6">
				<Card.Header>
					<Card.Title class="text-lg">Version History</Card.Title>
				</Card.Header>
				<Card.Content>
					{#if versions.length === 0}
						<p class="text-sm text-muted-foreground">No version history available.</p>
					{:else}
						<div class="space-y-3">
							{#each versions as version}
								{@const badge = getStatusBadge(version.status)}
								<div class="flex items-center justify-between border-b border-border pb-3 last:border-0 last:pb-0">
									<div class="flex items-center gap-3">
										<Badge variant={badge.variant} class={badge.class}>
											{badge.label}
										</Badge>
										<span class="font-medium">v{version.versionNumber}</span>
										{#if version.changeSummary}
											<span class="text-sm text-muted-foreground">{version.changeSummary}</span>
										{/if}
									</div>
									<div class="text-xs text-muted-foreground">
										{#if version.publishedAt}
											Published {formatDate(version.publishedAt)}
										{:else}
											Created {formatDate(version.createdAt)}
										{/if}
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</Card.Content>
			</Card.Root>
		{/if}

		<!-- Task Tree Editor -->
		{#if hasTaskTree}
			{#if treeLoading}
				<Card.Root>
					<Card.Content class="py-12">
						<div class="flex items-center justify-center">
							<Skeleton class="h-64 w-full" />
						</div>
					</Card.Content>
				</Card.Root>
			{:else if tree}
				<Card.Root>
					<Card.Header>
						<Card.Title class="text-lg">Recipe Steps</Card.Title>
					</Card.Header>
					<Card.Content>
						<TaskTreeEditor
							{tree}
							{stations}
							context="recipe"
							{stationMap}
							deps={depsMap}
							onmutate={handleTreeMutate}
						/>
					</Card.Content>
				</Card.Root>
			{/if}
		{:else}
			<Card.Root>
				<Card.Content class="py-12">
					<div class="text-center text-muted-foreground">
						<p>No task tree associated with this version.</p>
					</div>
				</Card.Content>
			</Card.Root>
		{/if}
	{/if}
</div>

<!-- Roll Recipe Dialog -->
<Dialog.Root
	bind:open={showRollDialog}
	onOpenChange={(open) => {
		if (!open) {
			rollTitle = '';
			rollOrderQty = 1;
			rollCustomerId = '';
			rollDueDate = '';
		}
	}}
>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Roll Recipe into Job</Dialog.Title>
			<Dialog.Description>
				Create a new job from <strong>{recipe?.name ?? 'this recipe'}</strong>
				{#if currentVersion}
					<span class="text-muted-foreground"> (v{currentVersion.versionNumber})</span>
				{/if}.
				The published task tree will be cloned.
			</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleRoll();
			}}
		>
			<div class="grid gap-4 py-2">
				<div class="grid gap-2">
					<Label for="roll-title">Job Title (optional)</Label>
					<Input
						id="roll-title"
						type="text"
						bind:value={rollTitle}
						placeholder="Defaults to recipe name"
						disabled={isRolling}
					/>
				</div>
				<div class="grid gap-2">
					<Label for="roll-qty">Order Quantity</Label>
					<Input
						id="roll-qty"
						type="number"
						bind:value={rollOrderQty}
						min={1}
						disabled={isRolling}
					/>
					<p class="text-xs text-muted-foreground">
						When greater than 1, tasks with batch sizes will be expanded accordingly.
					</p>
				</div>
				{#if customers.length > 0}
					<div class="grid gap-2">
						<Label for="roll-customer">Customer (optional)</Label>
						<Select.Root
							type="single"
							value={rollCustomerId || undefined}
							onValueChange={(v) => { rollCustomerId = v ?? ''; }}
						>
							<Select.Trigger id="roll-customer" class="w-full" disabled={isRolling}>
								{customers.find((c) => c.id === rollCustomerId)?.name ?? 'Select a customer'}
							</Select.Trigger>
							<Select.Content>
								<Select.Item value="" label="None">None</Select.Item>
								{#each customers as customer}
									<Select.Item value={customer.id} label={customer.name}>{customer.name}</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					</div>
				{/if}
				<div class="grid gap-2">
					<Label for="roll-due-date">Due Date (optional)</Label>
					<Input
						id="roll-due-date"
						type="date"
						bind:value={rollDueDate}
						disabled={isRolling}
					/>
				</div>
			</div>
			<Dialog.Footer class="pt-2">
				<Button
					type="button"
					variant="outline"
					onclick={() => (showRollDialog = false)}
					disabled={isRolling}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={isRolling}>
					{isRolling ? 'Creating Job...' : 'Roll'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>

<!-- New Version Dialog -->
<Dialog.Root
	bind:open={showNewVersionDialog}
	onOpenChange={(open) => {
		if (!open) newVersionSummary = '';
	}}
>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Create New Version</Dialog.Title>
			<Dialog.Description>
				Clone the published version into a new draft for editing.
			</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleNewVersion();
			}}
		>
			<div class="grid gap-4 py-2">
				<div class="grid gap-2">
					<Label for="version-summary">Change Summary (optional)</Label>
					<Input
						id="version-summary"
						type="text"
						bind:value={newVersionSummary}
						placeholder="What are you changing?"
						disabled={isCreatingVersion}
					/>
				</div>
			</div>
			<Dialog.Footer class="pt-2">
				<Button
					type="button"
					variant="outline"
					onclick={() => (showNewVersionDialog = false)}
					disabled={isCreatingVersion}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={isCreatingVersion}>
					{isCreatingVersion ? 'Creating...' : 'Create Draft'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
