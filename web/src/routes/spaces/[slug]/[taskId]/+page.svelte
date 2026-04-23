<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { taskApi } from '$lib/api/task';
	import type { TaskTreeResponse, TaskDepsResponse } from '$lib/api/task';
	import { jobApi } from '$lib/api/job';
	import type { TaskResponse } from '$lib/types/task';
	import type { CompleteTaskResponse } from '$lib/types/task';
	import { stationApi } from '$lib/api/station';
	import type { StationResponse } from '$lib/types/station';
	import { spaceStore } from '$lib/stores/space';
	import TaskTree from '$lib/components/flow/TaskTree.svelte';
	import TaskDetailPanel from '$lib/components/flow/TaskDetailPanel.svelte';
	import BoardView from '$lib/components/flow/BoardView.svelte';
	import GraphView from '$lib/components/flow/GraphView.svelte';
	import ListView from '$lib/components/flow/ListView.svelte';
	import JobCostSummary from '$lib/components/flow/JobCostSummary.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Separator } from '$lib/components/ui/separator';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb';
	import { CircleAlert, TreePine, LayoutGrid, GitBranch, List, DollarSign, BookOpen } from '@lucide/svelte';
	import { isEditableTarget } from '$lib/utils/keyboard.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Switch } from '$lib/components/ui/switch';
	import { toast } from 'svelte-sonner';

	let slug = $derived($page.params.slug);
	let taskId = $derived($page.params.taskId);
	let currentSpace = $derived($spaceStore.currentSpace);
	let spaceId = $derived(currentSpace?.id ?? '');

	let tree = $state<TaskTreeResponse | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let stationMap = $state<Map<string, string>>(new Map());

	// Selection state (for tree view)
	let selectedTask = $state<TaskTreeResponse | null>(null);
	let selectedDeps = $state<TaskDepsResponse | null>(null);
	/** Whether initial deps fetch for root has completed (success or failure). */
	let rootDepsAttempted = $state(false);

	// ---- View mode ----
	type ViewMode = 'tree' | 'board' | 'graph' | 'list' | 'cost';
	const ALL_VIEW_MODES: { value: ViewMode; label: string; icon: typeof LayoutGrid }[] = [
		{ value: 'tree', label: 'Tree', icon: TreePine },
		{ value: 'board', label: 'Board', icon: LayoutGrid },
		{ value: 'graph', label: 'Graph', icon: GitBranch },
		{ value: 'list', label: 'List', icon: List },
		{ value: 'cost', label: 'Cost', icon: DollarSign },
	];

	/** Whether the root is a non-job leaf task (no children). */
	let isLeafTask = $derived(
		tree !== null && tree.type !== 'job' && (!tree.children || tree.children.length === 0),
	);

	/** For non-job leaf tasks, hide the view toggle entirely (detail + graph only). */
	let availableViewModes = $derived(isLeafTask ? [] : ALL_VIEW_MODES);

	let currentView = $derived<ViewMode>(
		(($page.url.searchParams.get('view') as ViewMode) || 'tree') as ViewMode,
	);

	function setView(mode: ViewMode): void {
		const url = new URL($page.url);
		if (mode === 'tree') {
			url.searchParams.delete('view');
		} else {
			url.searchParams.set('view', mode);
		}
		goto(url.toString(), { replaceState: true, keepFocus: true, noScroll: true });
	}

	// ---- Flatten tree into TaskResponse[] for scoped views ----

	function flattenTree(node: TaskTreeResponse): TaskResponse[] {
		const result: TaskResponse[] = [];
		function walk(n: TaskTreeResponse): void {
			// Add the node as a TaskResponse (strip children)
			// eslint-disable-next-line @typescript-eslint/no-unused-vars
			const { children, ...task } = n;
			result.push(task as TaskResponse);
			if (n.children) {
				for (const child of n.children) {
					walk(child);
				}
			}
		}
		walk(node);
		return result;
	}

	/** Flattened descendants (excludes root) for scoped views. */
	let flatTasks = $derived.by((): TaskResponse[] => {
		if (!tree) return [];
		const all = flattenTree(tree);
		// Exclude the root task itself — views should show descendants only
		return all.slice(1);
	});

	// ---- Neighborhood graph for leaf tasks ----
	let neighborhoodTasks = $state<TaskResponse[]>([]);
	let neighborhoodDeps = $state<Map<string, TaskDepsResponse>>(new Map());
	let neighborhoodLoaded = $state(false);

	async function loadNeighborhoodGraph(): Promise<void> {
		if (neighborhoodLoaded || !tree || !selectedDeps) return;

		// Collect unique neighbor IDs from blockers + dependents
		const neighborIds = new Set<string>();
		for (const dep of selectedDeps.blockers) {
			// blocker: fromTaskId = this task (blocked), toTaskId = blocker — toTaskId is the neighbor
			if (dep.fromTaskId !== tree.id) neighborIds.add(dep.fromTaskId);
			if (dep.toTaskId !== tree.id) neighborIds.add(dep.toTaskId);
		}
		for (const dep of selectedDeps.dependents) {
			// dependent: fromTaskId = downstream task, toTaskId = this task (upstream) — fromTaskId is the neighbor
			if (dep.fromTaskId !== tree.id) neighborIds.add(dep.fromTaskId);
			if (dep.toTaskId !== tree.id) neighborIds.add(dep.toTaskId);
		}

		// Fetch task details for all neighbors
		const neighborResults = await Promise.all(
			[...neighborIds].map(async (id) => {
				try {
					return await taskApi.getTask(spaceId, id);
				} catch {
					return null;
				}
			}),
		);

		// Build the task list: center task + neighbors
		const { children: _, ...rootTask } = tree;
		const tasks: TaskResponse[] = [rootTask as TaskResponse];
		for (const t of neighborResults) {
			if (t) tasks.push(t);
		}

		// Build deps map: only the center task's deps (1 hop)
		const dMap = new Map<string, TaskDepsResponse>();
		dMap.set(tree.id, selectedDeps);

		neighborhoodTasks = tasks;
		neighborhoodDeps = dMap;
		neighborhoodLoaded = true;
	}

	/** Deps map for all descendants — built lazily when graph view is active. */
	let depsMap = $state<Map<string, TaskDepsResponse>>(new Map());
	let depsLoaded = $state(false);

	async function loadDepsForDescendants(): Promise<void> {
		if (depsLoaded || !tree) return;
		const tasks = flatTasks;
		const map = new Map<string, TaskDepsResponse>();

		// Fetch deps in batches of 20
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
		depsLoaded = true;
	}

	// Load deps when switching to graph view (or automatically for leaf tasks)
	$effect(() => {
		// Read all reactive values at top level so Svelte 5 tracks them
		const view = currentView;
		const treeVal = tree;
		const leaf = isLeafTask;
		const nLoaded = neighborhoodLoaded;
		const deps = selectedDeps;
		const dLoaded = depsLoaded;
		const depsReady = rootDepsAttempted;

		if (treeVal && leaf && !nLoaded && depsReady) {
			if (deps) {
				// Leaf task with deps: load neighborhood graph
				loadNeighborhoodGraph();
			} else {
				// Deps fetch failed: show just the task node (no edges)
				const { children: _c, ...rootTask } = treeVal;
				neighborhoodTasks = [rootTask as TaskResponse];
				neighborhoodDeps = new Map();
				neighborhoodLoaded = true;
			}
		} else if ((view === 'graph' || view === 'list' || view === 'tree') && treeVal && !leaf && !dLoaded) {
			// Job/parent task: load deps for all descendants when graph, list, or tree view selected
			loadDepsForDescendants();
		}
	});

	// ---- Keyboard shortcuts ----

	function handleKeydown(e: KeyboardEvent): void {
		if (isEditableTarget(e)) return;
		// No view switching shortcuts for leaf tasks (no view toggle)
		if (isLeafTask) return;

		switch (e.key) {
			case 't':
				setView('tree');
				break;
			case 'b':
				setView('board');
				break;
			case 'g':
				setView('graph');
				break;
			case 'l':
				// Only switch if not in board view (board uses 'l' for column navigation)
				if (currentView !== 'board') {
					setView('list');
				}
				break;
			case 'c':
				setView('cost');
				break;
		}
	}

	async function handleSelect(task: TaskTreeResponse | TaskResponse): Promise<void> {
		// If given a plain TaskResponse, find the full tree node if possible
		if (tree && !('children' in task)) {
			const found = findNode(tree, task.id);
			if (found) {
				selectedTask = found;
			} else {
				// Wrap as a tree response for the detail panel
				selectedTask = { ...task, children: [] } as TaskTreeResponse;
			}
		} else {
			selectedTask = task as TaskTreeResponse;
		}
		// Load deps for the selected task
		selectedDeps = null;
		try {
			selectedDeps = await taskApi.getTaskDeps(spaceId, task.id);
		} catch {
			// Deps endpoint may fail — degrade gracefully
			selectedDeps = null;
		}
	}

	/** After a task action, reload the tree to reflect the new state. */
	async function handleTaskAction(updated: TaskResponse): Promise<void> {
		if (!taskId) return;
		try {
			const newTree = await taskApi.getTaskTree(spaceId, taskId);
			tree = newTree;
			// Reset deps cache so graph view re-fetches on next switch
			depsLoaded = false;
			depsMap = new Map();
			neighborhoodLoaded = false;
			neighborhoodTasks = [];
			neighborhoodDeps = new Map();
			// Re-select the same task within the new tree
			if (selectedTask) {
				const found = findNode(newTree, selectedTask.id);
				if (found) {
					selectedTask = found;
					try {
						selectedDeps = await taskApi.getTaskDeps(spaceId, found.id);
					} catch {
						selectedDeps = null;
					}
				}
			}
		} catch {
			// Silently fail — user can refresh manually
		}
	}

	/** After completion flow, navigate to the next task if provided. */
	function handleCompletion(response: CompleteTaskResponse): void {
		if (response.nextTaskId) {
			goto(`/spaces/${slug}/${response.nextTaskId}`);
		}
	}

	/** Find a node by ID in the task tree. */
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

	onMount(async () => {
		if (!taskId) {
			error = 'No task ID provided';
			isLoading = false;
			return;
		}

		try {
			// Fetch tree and stations in parallel
			const [treeData, stations] = await Promise.all([
				taskApi.getTaskTree(spaceId, taskId),
				stationApi.listStations(spaceId).catch(() => [] as StationResponse[]),
			]);

			tree = treeData;

			// Build station map
			const map = new Map<string, string>();
			for (const s of stations) {
				map.set(s.id, s.name);
			}
			stationMap = map;

			// Auto-select root task and load its deps
			selectedTask = tree;
			try {
				selectedDeps = await taskApi.getTaskDeps(spaceId, tree.id);
			} catch {
				selectedDeps = null;
			}
			rootDepsAttempted = true;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load task tree';
		} finally {
			isLoading = false;
		}
	});

	// ---- Save as Recipe dialog ----
	let showSaveAsRecipeDialog = $state(false);
	let recipeName = $state('');
	let recipeDescription = $state('');
	let backfillEstimates = $state(true);
	let isSavingAsRecipe = $state(false);

	/** Whether the root task is a job (save-as-recipe is only available for jobs). */
	let isJob = $derived(tree?.type === 'job');

	async function handleSaveAsRecipe(): Promise<void> {
		if (!tree || !recipeName.trim()) return;
		isSavingAsRecipe = true;
		try {
			const recipe = await jobApi.saveAsRecipe(spaceId, tree.id, {
				name: recipeName.trim(),
				description: recipeDescription.trim() || undefined,
				backfillEstimatedFromActual: backfillEstimates,
			});
			showSaveAsRecipeDialog = false;
			toast.success('Recipe created from job. Review and publish when ready.');
			goto(`/recipes/${recipe.id}`);
		} catch (e) {
			toast.error(e instanceof Error ? e.message : 'Failed to save as recipe');
		} finally {
			isSavingAsRecipe = false;
		}
	}

	/** Compute root-level progress. */
	function computeRootProgress(root: TaskTreeResponse): { done: number; total: number } {
		if (!root.children || root.children.length === 0) return { done: 0, total: 0 };
		let done = 0;
		for (const child of root.children) {
			if (child.status === 'done' || child.status === 'skipped') done++;
		}
		return { done, total: root.children.length };
	}
</script>

<svelte:head>
	<title>{tree?.title ?? 'Task'} - Nori</title>
</svelte:head>

<svelte:window onkeydown={handleKeydown} />

<div class="flex-1 overflow-hidden flex flex-col">
	<!-- Header: breadcrumbs, title, view toggle -->
	<div class="px-4 sm:px-6 pt-3 pb-2 border-b border-border shrink-0 space-y-2">
		<!-- 1. Breadcrumbs -->
		<Breadcrumb.Root>
			<Breadcrumb.List>
				<Breadcrumb.Item>
					<Breadcrumb.Link href="/spaces">Spaces</Breadcrumb.Link>
				</Breadcrumb.Item>
				<Breadcrumb.Separator />
				<Breadcrumb.Item>
					<Breadcrumb.Link href="/spaces/{slug}">{currentSpace?.name ?? slug}</Breadcrumb.Link>
				</Breadcrumb.Item>
				<Breadcrumb.Separator />
				<Breadcrumb.Item>
					<Breadcrumb.Page>{taskId}</Breadcrumb.Page>
				</Breadcrumb.Item>
			</Breadcrumb.List>
		</Breadcrumb.Root>

		<!-- 2. Title + actions -->
		{#if tree}
			<div class="flex items-center gap-3">
				<h1 class="text-lg font-bold text-foreground truncate">{tree.title}</h1>
				{#if isJob}
					<Button variant="outline" size="sm" onclick={() => (showSaveAsRecipeDialog = true)}>
						<BookOpen class="size-4 mr-1" />
						Save as Recipe
					</Button>
				{/if}
			</div>
		{:else if isLoading}
			<Skeleton class="h-6 w-48" />
		{/if}

		<!-- 3. View toggle buttons (hidden for non-job leaf tasks) -->
		{#if availableViewModes.length > 0}
			<div class="flex items-center rounded-lg border bg-muted/50 p-0.5 w-fit">
				{#each availableViewModes as mode (mode.value)}
					<Button
						variant={currentView === mode.value ? 'secondary' : 'ghost'}
						size="sm"
						class="gap-1.5 rounded-md px-3 {currentView === mode.value
							? 'bg-background shadow-sm'
							: 'hover:bg-transparent hover:text-foreground'}"
						onclick={() => setView(mode.value)}
					>
						<mode.icon class="size-4" />
						{mode.label}
					</Button>
				{/each}
			</div>
		{/if}
	</div>

	<!-- 4. View content -->
	{#if isLoading}
		<div class="flex-1 flex items-center justify-center p-8">
			<div class="space-y-4 w-full max-w-md">
				<Skeleton class="h-8 w-1/2" />
				<Skeleton class="h-4 w-1/3" />
				<div class="mt-6 space-y-3">
					{#each Array(5) as _}
						<div class="flex items-center gap-3">
							<Skeleton class="h-5 w-5 rounded-full" />
							<Skeleton class="h-5 w-2/3" />
						</div>
					{/each}
				</div>
			</div>
		</div>

	{:else if error}
		<div class="flex-1 flex items-center justify-center p-8">
			<div class="border border-destructive/30 bg-destructive/5 rounded-lg p-8 text-center max-w-md">
				<CircleAlert class="w-10 h-10 text-destructive mx-auto mb-3" />
				<h3 class="text-lg font-semibold text-foreground mb-1">Failed to load task</h3>
				<p class="text-sm text-muted-foreground mb-4">{error}</p>
				<Button variant="outline" size="sm" onclick={() => window.location.reload()}>
					Try Again
				</Button>
			</div>
		</div>

	{:else if tree}
		{#if isLeafTask}
			<!-- Leaf task: detail panel + neighborhood graph (no view toggle) -->
			<div class="flex-1 flex overflow-hidden">
				<!-- Left: Detail panel -->
				<div class="w-1/2 overflow-y-auto border-r border-border">
					{#if selectedTask}
						<TaskDetailPanel
							task={selectedTask}
							{stationMap}
							deps={selectedDeps}
							onaction={handleTaskAction}
							oncomplete={handleCompletion}
						/>
					{/if}
				</div>

				<!-- Right: Neighborhood graph -->
				<div class="w-1/2 overflow-hidden">
					{#if neighborhoodLoaded}
						<GraphView tasks={neighborhoodTasks} deps={neighborhoodDeps} {stationMap} focusTaskId={tree.id} />
					{:else}
						<div class="flex items-center justify-center h-full text-sm text-muted-foreground">
							Loading neighborhood...
						</div>
					{/if}
				</div>
			</div>

		{:else if currentView === 'tree'}
			{@const rootProgress = computeRootProgress(tree)}

			<div class="flex-1 flex overflow-hidden">
				<!-- Left: Task tree -->
				<div class="w-2/5 border-r border-border overflow-y-auto p-4">
					{#if rootProgress.total > 0}
						<div class="mb-4">
							<div class="flex items-center justify-between text-xs text-muted-foreground mb-1">
								<span>{rootProgress.done}/{rootProgress.total} done</span>
								<span>{Math.round((rootProgress.done / rootProgress.total) * 100)}%</span>
							</div>
							<div class="h-1.5 bg-muted rounded-full overflow-hidden">
								<div
									class="h-full bg-green-500 transition-all duration-300"
									style="width: {(rootProgress.done / rootProgress.total) * 100}%"
								></div>
							</div>
						</div>
						<Separator class="mb-3" />
					{/if}

					{#if tree.children && tree.children.length > 0}
						<TaskTree
							tasks={flatTasks}
							deps={depsMap}
							{stationMap}
							selectedTaskId={selectedTask?.id}
							onselect={handleSelect}
						/>
					{:else}
						<div class="border border-border rounded-lg p-6 text-center">
							<p class="text-sm text-muted-foreground">No child tasks yet.</p>
						</div>
					{/if}
				</div>

				<!-- Right: Detail panel -->
				<div class="w-3/5 overflow-y-auto">
					{#if selectedTask}
						<TaskDetailPanel
							task={selectedTask}
							{stationMap}
							deps={selectedDeps}
							onaction={handleTaskAction}
							oncomplete={handleCompletion}
						/>
					{:else}
						<div class="flex items-center justify-center h-full text-sm text-muted-foreground">
							Select a task to view details
						</div>
					{/if}
				</div>
			</div>

		{:else if currentView === 'board'}
			<BoardView tasks={flatTasks} {stationMap} />

		{:else if currentView === 'graph'}
			<GraphView tasks={flatTasks} deps={depsMap} {stationMap} />

		{:else if currentView === 'list'}
			<ListView tasks={flatTasks} deps={depsMap} {stationMap} />

		{:else if currentView === 'cost'}
			<div class="flex-1 overflow-y-auto">
				<JobCostSummary jobId={tree.id} {tree} {stationMap} />
			</div>
		{/if}
	{/if}
</div>

<!-- Save as Recipe Dialog -->
<Dialog.Root
	bind:open={showSaveAsRecipeDialog}
	onOpenChange={(open) => {
		if (!open) {
			recipeName = '';
			recipeDescription = '';
			backfillEstimates = true;
		}
	}}
>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Save as Recipe</Dialog.Title>
			<Dialog.Description>
				Create a reusable recipe template from this job's task tree.
			</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleSaveAsRecipe();
			}}
		>
			<div class="grid gap-4 py-2">
				<div class="grid gap-2">
					<Label for="recipe-name">Recipe Name</Label>
					<Input
						id="recipe-name"
						bind:value={recipeName}
						placeholder="e.g. Custom Bookshelf"
						required
					/>
				</div>
				<div class="grid gap-2">
					<Label for="recipe-description">Description (optional)</Label>
					<Input
						id="recipe-description"
						bind:value={recipeDescription}
						placeholder="Brief description of this recipe"
					/>
				</div>
				<div class="flex items-center justify-between">
					<Label for="backfill-estimates" class="text-sm font-normal">
						Use actual times as estimated times
					</Label>
					<Switch id="backfill-estimates" bind:checked={backfillEstimates} />
				</div>
			</div>
			<Dialog.Footer class="pt-2">
				<Button
					type="button"
					variant="outline"
					onclick={() => (showSaveAsRecipeDialog = false)}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={isSavingAsRecipe || !recipeName.trim()}>
					{isSavingAsRecipe ? 'Saving...' : 'Save as Recipe'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
