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
	import GraphView from '$lib/components/flow/GraphView.svelte';
	import TaskDetailPanel from '$lib/components/flow/TaskDetailPanel.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Separator } from '$lib/components/ui/separator';
	import * as Alert from '$lib/components/ui/alert';
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Select from '$lib/components/ui/select';
	import * as Card from '$lib/components/ui/card';
	import {
		CircleAlert,
		ArrowLeft,
		Send,
		Play,
		Plus,
		History,
		PanelRight,
		X,
	} from '@lucide/svelte';
	import { toast } from 'svelte-sonner';

	let recipeId = $derived($page.params.id);
	let spaceId = $derived($page.data.space!.id);
	let recipe = $derived($recipeStore.currentRecipe);
	let loading = $derived($recipeStore.loading);
	let storeError = $derived($recipeStore.error);

	// Task tree state
	let tree = $state<TaskTreeResponse | null>(null);
	let flatTasks = $state<TaskResponse[]>([]);
	let depsMap = $state<Map<string, TaskDepsResponse>>(new Map());
	let stationMap = $state<Map<string, string>>(new Map());
	let treeLoading = $state(false);

	let taskTitleMap = $derived(new Map(flatTasks.map((t) => [t.id, t.title])));

	// Graph detail panel state
	let graphPanelOpen = $state(true);
	let graphFullscreen = $state(false);
	let graphAreaEl: HTMLDivElement | undefined = $state(undefined);
	let graphPanelTask = $state<TaskTreeResponse | null>(null);
	let graphPanelDeps = $state<TaskDepsResponse | null>(null);
	let graphPanelLoading = $state(false);

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
	let rootTaskId = $derived(currentVersion?.rootTaskId ?? '');

	let lastLoadedId = $state<string | null>(null);

	$effect(() => {
		const id = $page.params.id;
		if (id && id !== lastLoadedId) {
			lastLoadedId = id;
			recipeStore.loadRecipe(spaceId, id);
		}
	});

	// Load task tree when recipe and rootTaskId are available
	$effect(() => {
		const ver = currentVersion;
		if (ver?.rootTaskId && ver.rootTaskId !== tree?.id) {
			loadTaskTree(ver.rootTaskId);
		}
	});

	// Initialize graph panel with root task when tree loads
	$effect(() => {
		if (tree && !graphPanelTask) {
			graphPanelTask = tree;
		}
	});

	onMount(() => {
		loadStations();
		loadCustomers();
		return () => recipeStore.clearCurrent();
	});

	async function loadStations() {
		try {
			const stations: StationResponse[] = await stationApi.listStations(spaceId);
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
			customers = await customerApi.listCustomers(spaceId);
		} catch {
			// Customers are optional for the roll dialog
		}
	}

	async function loadTaskTree(rootId: string, silent = false) {
		if (!silent) treeLoading = true;
		try {
			tree = await taskApi.getTaskTree(spaceId, rootId);
			flatTasks = flattenTree(tree).slice(1); // exclude root
			await loadDepsForTasks(flatTasks);
		} catch (err) {
			console.error('Failed to load task tree:', err);
		} finally {
			treeLoading = false;
		}
	}

	async function loadDepsForTasks(tasks: TaskResponse[]) {
		const map = new Map<string, TaskDepsResponse>();
		const BATCH_SIZE = 20;
		for (let i = 0; i < tasks.length; i += BATCH_SIZE) {
			const batch = tasks.slice(i, i + BATCH_SIZE);
			const results = await Promise.all(
				batch.map(async (t) => {
					try {
						const deps = await taskApi.getTaskDeps(spaceId, t.id);
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

	function findNode(node: TaskTreeResponse, id: string): TaskTreeResponse | null {
		if (node.id === id) return node;
		if (node.children) {
			for (const child of node.children) {
				const found = findNode(child, id);
				if (found) return found;
			}
		}
		return null;
	}

	/** Called when a node is clicked in the graph — loads its details into the side panel. */
	async function handleGraphNodeSelect(taskId: string): Promise<void> {
		if (!tree) return;
		graphPanelLoading = true;
		graphPanelOpen = true;
		try {
			const found = findNode(tree, taskId);
			if (found) {
				graphPanelTask = found;
			} else {
				const flat = flatTasks.find((t) => t.id === taskId);
				if (flat) {
					graphPanelTask = { ...flat, children: [] } as TaskTreeResponse;
				}
			}
			try {
				graphPanelDeps = await taskApi.getTaskDeps(spaceId, taskId);
			} catch {
				graphPanelDeps = null;
			}
		} finally {
			graphPanelLoading = false;
		}
	}

	/** Navigate the detail panel back to the recipe root task. */
	function handleNavToRecipeRoot(): void {
		if (tree) {
			graphPanelTask = tree;
			graphPanelDeps = null;
		}
	}

	/** Select a task in the graph (called from dep badge clicks in detail panel). */
	function handleSelectTaskInGraph(taskId: string): void {
		handleGraphNodeSelect(taskId);
	}

	/** Whether the currently shown panel task is the root (hide sub-tasks for root). */
	let panelIsRoot = $derived(graphPanelTask?.id === tree?.id);

	/** Reload tree after graph mutations (node add/delete/rename). */
	async function handleTreeMutate(): Promise<void> {
		if (currentVersion?.rootTaskId) {
			await loadTaskTree(currentVersion.rootTaskId, true);
		}
	}

	function toggleGraphFullscreen(): void {
		graphFullscreen = !graphFullscreen;
	}

	async function handlePublish() {
		if (!recipe || isPublishing) return;
		isPublishing = true;
		try {
			await recipeStore.publishVersion(spaceId, recipe.id);
			toast.success('Recipe published successfully');
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
			const job = await recipeApi.rollRecipe(spaceId, recipe.id, {
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
			goto(`/spaces/${$page.params.slug}/${job.id}`);
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
			await recipeStore.createNewVersion(spaceId, recipe.id, newVersionSummary.trim() || undefined);
			showNewVersionDialog = false;
			newVersionSummary = '';
			// Reset panel and tree state
			graphPanelTask = null;
			graphPanelDeps = null;
			tree = null;
			flatTasks = [];
			depsMap = new Map();
			toast.success('New draft version created');
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
				versions = await recipeApi.listVersions(spaceId, recipe.id);
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

<svelte:head>
	<title>{recipe?.name ?? 'Recipe'} - Nori</title>
</svelte:head>

<div class="flex-1 overflow-hidden flex flex-col">
	{#if loading && !recipe}
		<!-- Loading skeleton -->
		<div class="p-6 space-y-4">
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
		<div class="p-6">
			<Alert.Root variant="destructive">
				<CircleAlert />
				<Alert.Title>Error</Alert.Title>
				<Alert.Description>{storeError}</Alert.Description>
			</Alert.Root>
		</div>
	{:else if recipe}
		<!-- Header -->
		<div class="px-4 sm:px-6 pt-3 pb-2 border-b border-border shrink-0 space-y-2">
			<div class="mb-1">
				<Button variant="ghost" size="sm" href="/spaces/{$page.params.slug}/recipes" class="h-7 px-2 text-xs text-muted-foreground">
					<ArrowLeft class="size-3 mr-1" />
					Back to Recipes
				</Button>
			</div>

			<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
				<div class="flex items-center gap-2 min-w-0">
					<h1 class="text-lg font-bold text-foreground truncate">{recipe.name}</h1>
					{#if currentVersion}
						{@const badge = getStatusBadge(currentVersion.status)}
						<Badge variant={badge.variant} class="{badge.class} shrink-0">
							{badge.label}
						</Badge>
						<Badge variant="secondary" class="shrink-0">v{currentVersion.versionNumber}</Badge>
					{/if}
				</div>

				<!-- Actions -->
				<div class="flex items-center gap-2 shrink-0">
					{#if isDraft}
						<Button size="sm" onclick={handlePublish} disabled={isPublishing}>
							<Send class="size-4 mr-1" />
							{isPublishing ? 'Publishing...' : 'Publish'}
						</Button>
					{/if}

					{#if isPublished}
						<Button size="sm" onclick={() => (showRollDialog = true)}>
							<Play class="size-4 mr-1" />
							Roll
						</Button>
						<Button variant="outline" size="sm" onclick={() => (showNewVersionDialog = true)}>
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

			{#if recipe.description}
				<p class="text-sm text-muted-foreground">{recipe.description}</p>
			{/if}
		</div>

		<!-- Version History (collapsible, inline below header) -->
		{#if showVersionHistory}
			<div class="border-b border-border px-4 sm:px-6 py-3 shrink-0 bg-muted/30">
				<h3 class="text-sm font-medium text-foreground mb-2">Version History</h3>
				{#if versions.length === 0}
					<p class="text-sm text-muted-foreground">No version history available.</p>
				{:else}
					<div class="space-y-2">
						{#each versions as version}
							{@const badge = getStatusBadge(version.status)}
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-2">
									<Badge variant={badge.variant} class={badge.class}>
										{badge.label}
									</Badge>
									<span class="text-sm font-medium">v{version.versionNumber}</span>
									{#if version.changeSummary}
										<span class="text-xs text-muted-foreground">{version.changeSummary}</span>
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
			</div>
		{/if}

		<!-- Graph + detail panel area -->
		{#if hasTaskTree}
			{#if treeLoading}
				<div class="flex-1 flex items-center justify-center">
					<div class="flex flex-col items-center gap-3">
						<Skeleton class="h-8 w-8 rounded-full" />
						<p class="text-sm text-muted-foreground">Loading recipe steps...</p>
					</div>
				</div>
			{:else}
				<div bind:this={graphAreaEl} class="{graphFullscreen ? 'fixed inset-0 z-50 bg-background' : 'flex-1'} flex overflow-hidden">
					<!-- Graph canvas -->
					<div class="flex-1 min-w-0 overflow-hidden relative">
						<GraphView
							{spaceId}
							tasks={flatTasks}
							deps={depsMap}
							{stationMap}
							mode="recipe"
							{rootTaskId}
							onselect={handleGraphNodeSelect}
							onmutate={handleTreeMutate}
							onfullscreentoggle={toggleGraphFullscreen}
							isFullscreen={graphFullscreen}
						/>
					</div>

					<!-- Detail panel (desktop sidebar) -->
					{#if graphPanelOpen && !graphFullscreen}
						<div class="hidden lg:flex w-1/2 flex-col border-l border-border overflow-y-auto shrink-0">
							<TaskDetailPanel
								task={graphPanelTask ?? tree ?? undefined}
								{spaceId}
								{stationMap}
								{taskTitleMap}
								deps={graphPanelDeps}
								isLoading={graphPanelLoading}
								mode="recipe"
								parentName={recipe?.name}
								parentType="recipe"
								onnavparent={handleNavToRecipeRoot}
								onselecttask={handleSelectTaskInGraph}
								hideSubTasks={panelIsRoot}
							/>
						</div>
					{/if}

					<!-- Detail panel (mobile drawer overlay) -->
					{#if graphPanelTask}
						<div class="lg:hidden fixed inset-y-0 right-0 w-4/5 max-w-sm z-50 flex flex-col bg-background border-l border-border shadow-xl overflow-y-auto">
							<div class="flex items-center justify-between px-4 py-3 border-b border-border shrink-0">
								<span class="text-sm font-medium text-foreground truncate">{graphPanelTask.title}</span>
								<button
									onclick={() => { graphPanelTask = null; }}
									class="ml-2 text-muted-foreground hover:text-foreground"
								>
									<X class="size-4" />
								</button>
							</div>
							<div class="flex-1 overflow-y-auto">
								<TaskDetailPanel
									task={graphPanelTask}
									{spaceId}
									{stationMap}
									{taskTitleMap}
									deps={graphPanelDeps}
									isLoading={graphPanelLoading}
									mode="recipe"
									parentName={recipe?.name}
									parentType="recipe"
									onnavparent={handleNavToRecipeRoot}
									onselecttask={handleSelectTaskInGraph}
									hideSubTasks={panelIsRoot}
								/>
							</div>
						</div>
						<!-- Backdrop -->
						<button
							class="lg:hidden fixed inset-0 z-40 bg-black/20"
							onclick={() => { graphPanelTask = tree ?? null; }}
							aria-label="Close panel"
						></button>
					{/if}
				</div>
			{/if}
		{:else}
			<div class="flex-1 flex items-center justify-center">
				<div class="text-center text-muted-foreground">
					<p class="text-sm">No task tree associated with this version.</p>
				</div>
			</div>
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
