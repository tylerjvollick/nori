<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { taskApi } from '$lib/api/task';
	import type { TaskTreeResponse, TaskDepsResponse } from '$lib/api/task';
	import { jobApi } from '$lib/api/job';
	import type { TaskResponse } from '$lib/types/task';
	import type { CompleteTaskResponse } from '$lib/types/task';
	import { stationApi } from '$lib/api/station';
	import type { StationResponse } from '$lib/types/station';
	import TaskDetailPanel from '$lib/components/flow/TaskDetailPanel.svelte';
	import BoardView from '$lib/components/flow/BoardView.svelte';
	import GraphView from '$lib/components/flow/GraphView.svelte';
	import ListView from '$lib/components/flow/ListView.svelte';
	import JobCostSummary from '$lib/components/flow/JobCostSummary.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb';
	import { CircleAlert, BookOpen, PanelLeftOpen, X, Briefcase, ListTodo, Trash2 } from '@lucide/svelte';
	import { isEditableTarget } from '$lib/utils/keyboard.svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Switch } from '$lib/components/ui/switch';
	import { toast } from 'svelte-sonner';

	let slug = $derived($page.params.slug);
	let taskId = $derived($page.params.taskId);
	let space = $derived($page.data.space!);
	let spaceId = $derived(space.id);

	let tree = $state<TaskTreeResponse | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let stationMap = $state<Map<string, string>>(new Map());

	// Selection state (for tree view)
	let selectedTask = $state<TaskTreeResponse | null>(null);
	let selectedDeps = $state<TaskDepsResponse | null>(null);
	/** Whether initial deps fetch for root has completed (success or failure). */
	let rootDepsAttempted = $state(false);

	// Parent task (for child task breadcrumbs: Spaces > Space > ParentJob > Task)
	let parentTask = $state<TaskResponse | null>(null);

	// ---- View mode (top-level tabs) ----
	type ViewMode = 'overview' | 'board' | 'graph' | 'cost';
	const ALL_VIEW_MODES: { value: ViewMode; label: string }[] = [
		{ value: 'overview', label: 'Overview' },
		{ value: 'board', label: 'Board' },
		{ value: 'graph', label: 'Graph' },
		{ value: 'cost', label: 'Cost' },
	];

	/** Whether the root is a non-job leaf task (no children). */
	let isLeafTask = $derived(
		tree !== null && tree.type !== 'job' && (!tree.children || tree.children.length === 0),
	);

	/** For non-job leaf tasks, hide the tabs entirely (detail + neighborhood graph only). */
	let availableViewModes = $derived(isLeafTask ? [] : ALL_VIEW_MODES);

	/** Normalize the ?view= param to a known tab; default/unknown → Overview. */
	function normalizeView(raw: string | null): ViewMode {
		switch (raw) {
			case 'board':
			case 'graph':
			case 'cost':
				return raw;
			default:
				return 'overview';
		}
	}

	let currentView = $derived<ViewMode>(normalizeView($page.url.searchParams.get('view')));

	function setView(mode: ViewMode): void {
		const url = new URL($page.url);
		if (mode === 'overview') {
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

	let taskTitleMap = $derived(new Map(flatTasks.map((t) => [t.id, t.title])));

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

	// ---- Graph detail panel ----
	let graphPanelOpen = $state(true);
	let graphPanelTask = $state<TaskTreeResponse | null>(null);
	let graphPanelDeps = $state<TaskDepsResponse | null>(null);
	let graphPanelLoading = $state(false);

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

	// Initialize graph panel with root task on load
	$effect(() => {
		if (tree && !graphPanelTask) {
			graphPanelTask = tree;
			graphPanelDeps = selectedDeps;
		}
	});

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
		} else if ((view === 'graph' || view === 'overview') && treeVal && !leaf && !dLoaded) {
			// Job/parent task: load deps for all descendants for the graph and the
			// Overview list (the list shows blocker/dependent state from deps).
			loadDepsForDescendants();
		}
	});

	// ---- Keyboard shortcuts ----

	function handleKeydown(e: KeyboardEvent): void {
		if (isEditableTarget(e)) return;
		// No view switching shortcuts for leaf tasks (no view toggle)
		if (isLeafTask) return;

		switch (e.key) {
			case 'o':
				setView('overview');
				break;
			case 'b':
				setView('board');
				break;
			case 'g':
				setView('graph');
				break;
			case 'c':
				setView('cost');
				break;
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

	/** Called when a node is clicked in the graph view — loads its details into the side panel. */
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

	// Sequence guard so a stale in-flight fetch can't overwrite a newer one
	// when the user navigates quickly between task detail pages.
	let loadSeq = 0;

	/**
	 * Load the task tree, stations, deps and parent for a given task id, fully
	 * resetting per-task state first. Driven reactively by taskId so in-app
	 * navigation between task detail pages (same [taskId] route) re-fetches
	 * instead of showing the stale, originally-mounted task.
	 */
	async function loadTask(id: string): Promise<void> {
		const seq = ++loadSeq;

		// Reset per-task state so navigation never renders stale data.
		isLoading = true;
		error = null;
		tree = null;
		selectedTask = null;
		selectedDeps = null;
		rootDepsAttempted = false;
		parentTask = null;
		graphPanelTask = null;
		graphPanelDeps = null;
		graphPanelLoading = false;
		depsLoaded = false;
		depsMap = new Map();
		neighborhoodLoaded = false;
		neighborhoodTasks = [];
		neighborhoodDeps = new Map();

		try {
			// Fetch tree and stations in parallel
			const [treeData, stations] = await Promise.all([
				taskApi.getTaskTree(spaceId, id),
				stationApi.listStations(spaceId).catch(() => [] as StationResponse[]),
			]);
			if (seq !== loadSeq) return; // superseded by a newer navigation

			tree = treeData;

			// Build station map
			const map = new Map<string, string>();
			for (const s of stations) {
				map.set(s.id, s.name);
			}
			stationMap = map;

			// Auto-select root task and load its deps
			selectedTask = treeData;
			try {
				const deps = await taskApi.getTaskDeps(spaceId, treeData.id);
				if (seq !== loadSeq) return;
				selectedDeps = deps;
			} catch {
				if (seq !== loadSeq) return;
				selectedDeps = null;
			}
			rootDepsAttempted = true;

			// Load parent task for breadcrumbs (child task under a job)
			if (treeData.parentId) {
				try {
					const p = await taskApi.getTask(spaceId, treeData.parentId);
					if (seq !== loadSeq) return;
					parentTask = p;
				} catch {
					if (seq !== loadSeq) return;
					parentTask = null;
				}
			}
		} catch (e) {
			if (seq !== loadSeq) return;
			error = e instanceof Error ? e.message : 'Failed to load task tree';
		} finally {
			if (seq === loadSeq) isLoading = false;
		}
	}

	// Re-fetch whenever the task id changes (including in-app navigation).
	$effect(() => {
		const id = taskId;
		if (!id) {
			error = 'No task ID provided';
			isLoading = false;
			return;
		}
		loadTask(id);
	});

	/** Reload the task tree and deps — called by GraphView after mutations. */
	async function handleTreeMutate(): Promise<void> {
		if (!taskId || !spaceId) return;
		try {
			const treeData = await taskApi.getTaskTree(spaceId, taskId);
			tree = treeData;
			// Force a fresh deps load. loadDepsForDescendants() early-returns when
			// depsLoaded is already true, so without resetting it here a newly added
			// serial/parallel node renders with no connecting edges until a full
			// page refresh (nori-c98.2).
			depsLoaded = false;
			await loadDepsForDescendants();
		} catch {
			// ignore — tree will be stale but not crash
		}
	}

	// ---- Save as Recipe dialog ----
	let showSaveAsRecipeDialog = $state(false);
	let recipeName = $state('');
	let recipeDescription = $state('');
	let backfillEstimates = $state(true);
	let isSavingAsRecipe = $state(false);

	/** Whether the root task is a job (save-as-recipe is only available for jobs). */
	let isJob = $derived(tree?.type === 'job');

	/** Navigate the detail panel back to the job/task root. */
	function handleNavToRoot(): void {
		if (tree) {
			graphPanelTask = tree;
			graphPanelDeps = null;
		}
	}

	/** Select a task in the graph (called from dep badge clicks in detail panel). */
	function handleSelectTaskInGraph(depTaskId: string): void {
		handleGraphNodeSelect(depTaskId);
	}

	/** Open the full detail page for the task currently shown in the graph panel. */
	function handleOpenPanelDetail(): void {
		const id = graphPanelTask?.id;
		if (id) goto(`/spaces/${slug}/${id}`);
	}

	/** Whether the currently shown panel task is the root (hide sub-tasks for root). */
	let graphPanelIsRoot = $derived(graphPanelTask?.id === tree?.id);

	// ---- Delete task ----
	let showDeleteDialog = $state(false);
	let isDeleting = $state(false);

	/** Number of descendant sub-tasks that a delete would cascade to. */
	let descendantCount = $derived(flatTasks.length);

	/**
	 * Hard-delete the task being viewed. Reconnects its upstream blockers to its
	 * downstream dependents first (so the chain isn't broken), then deletes and
	 * navigates away — to the parent task if this is a child, else the jobs list.
	 * Matches the reconnect-then-delete behavior of GraphView.handleBeforeDelete.
	 */
	async function handleDelete(): Promise<void> {
		if (!tree) return;
		isDeleting = true;
		try {
			// Reconnect upstream blockers -> downstream dependents before deleting.
			if (selectedDeps) {
				const upstreamIds = selectedDeps.blockers.map((d) => d.toTaskId);
				const downstreamIds = selectedDeps.dependents.map((d) => d.fromTaskId);
				for (const upstream of upstreamIds) {
					for (const downstream of downstreamIds) {
						try {
							await taskApi.addDep(spaceId, upstream, downstream, 'blocks');
						} catch {
							// Ignore duplicate-dep errors — the link may already exist.
						}
					}
				}
			}

			await taskApi.deleteTask(spaceId, tree.id);
			showDeleteDialog = false;
			toast.success('Task deleted');
			// Can't stay on a deleted task's page — go to the parent or the jobs list.
			if (parentTask) {
				goto(`/spaces/${slug}/${parentTask.id}`);
			} else {
				goto(`/spaces/${slug}/jobs`);
			}
		} catch (e) {
			toast.error(e instanceof Error ? e.message : 'Failed to delete task');
		} finally {
			isDeleting = false;
		}
	}

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

</script>

<svelte:head>
	<title>{tree?.title ?? 'Task'} - Nori</title>
</svelte:head>

<svelte:window onkeydown={handleKeydown} />

<div class="flex-1 overflow-hidden flex flex-col">
	<!-- Header: breadcrumbs + actions -->
	<div class="px-4 sm:px-6 pt-3 pb-2 border-b border-border shrink-0">
		<div class="flex items-center justify-between">
			<!-- Breadcrumbs -->
			<Breadcrumb.Root>
				<Breadcrumb.List>
					<Breadcrumb.Item>
						<Breadcrumb.Link href="/spaces">Spaces</Breadcrumb.Link>
					</Breadcrumb.Item>
					<Breadcrumb.Separator />
					<Breadcrumb.Item>
						<Breadcrumb.Link href="/spaces/{slug}">{space.name}</Breadcrumb.Link>
					</Breadcrumb.Item>
					{#if parentTask}
						<Breadcrumb.Separator />
						<Breadcrumb.Item>
							<Breadcrumb.Link href="/spaces/{slug}/{parentTask.id}" class="flex items-center gap-1">
								<Briefcase class="size-3.5" />
								{parentTask.title}
							</Breadcrumb.Link>
						</Breadcrumb.Item>
						<Breadcrumb.Separator />
						<Breadcrumb.Item>
							<Breadcrumb.Page class="flex items-center gap-1">
								<ListTodo class="size-3.5" />
								{tree?.title ?? taskId}
							</Breadcrumb.Page>
						</Breadcrumb.Item>
					{:else}
						<Breadcrumb.Separator />
						<Breadcrumb.Item>
							<Breadcrumb.Page class="flex items-center gap-1">
								{#if tree?.type === 'job'}
									<Briefcase class="size-3.5" />
								{:else}
									<ListTodo class="size-3.5" />
								{/if}
								{tree?.title ?? taskId}
							</Breadcrumb.Page>
						</Breadcrumb.Item>
					{/if}
				</Breadcrumb.List>
			</Breadcrumb.Root>

			<!-- Actions (right side of breadcrumb bar) -->
			<div class="flex items-center gap-1">
				{#if isJob}
					<Button variant="ghost" size="sm" onclick={() => (showSaveAsRecipeDialog = true)}>
						<BookOpen class="size-4 mr-1" />
						Save as Recipe
					</Button>
				{/if}
				<!-- Delete is offered on individual tasks only; a whole job tree is not
				     deletable in one click (delete its tasks from the graph instead). -->
				{#if tree && !isJob}
					<Button
						variant="ghost"
						size="sm"
						class="text-destructive hover:text-destructive"
						onclick={() => (showDeleteDialog = true)}
						data-testid="delete-task"
					>
						<Trash2 class="size-4 mr-1" />
						Delete
					</Button>
				{/if}
			</div>
		</div>
	</div>

	<!-- Top-level tabs: [Overview] [Board] [Graph] [Cost] (for jobs/tasks with children) -->
	{#if availableViewModes.length > 0}
		<div class="flex-shrink-0 border-b bg-background px-4 pt-0 pb-0">
			<div class="flex gap-0" role="tablist">
				{#each availableViewModes as mode (mode.value)}
					<button
						role="tab"
						aria-selected={currentView === mode.value}
						class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
							{currentView === mode.value
								? 'border-foreground text-foreground'
								: 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground/40'}"
						onclick={() => setView(mode.value)}
					>
						{mode.label}
					</button>
				{/each}
			</div>
		</div>
	{/if}

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
							{spaceId}
							{stationMap}
							{taskTitleMap}
							deps={selectedDeps}
							onaction={handleTaskAction}
							oncomplete={handleCompletion}
						/>
					{/if}
				</div>

				<!-- Right: Neighborhood graph -->
				<div class="w-1/2 overflow-hidden">
					{#if neighborhoodLoaded}
						<GraphView {spaceId} tasks={neighborhoodTasks} deps={neighborhoodDeps} {stationMap} focusTaskId={tree.id} />
					{:else}
						<div class="flex items-center justify-center h-full text-sm text-muted-foreground">
							Loading neighborhood...
						</div>
					{/if}
				</div>
			</div>

		{:else if currentView === 'cost'}
			<!-- Cost tab: full width -->
			<div class="flex-1 overflow-y-auto">
				<JobCostSummary jobId={tree.id} {spaceId} {tree} {stationMap} />
			</div>

		{:else if currentView === 'board'}
			<!-- Board tab: full-width board (no detail panel) -->
			<div class="flex-1 overflow-hidden">
				<BoardView {spaceId} tasks={flatTasks} {stationMap} />
			</div>

		{:else if currentView === 'graph'}
			<!-- Graph tab: graph (left) + read-only detail panel (right). Clicking a
			     node activates it and loads it into the panel; Enter or the panel's
			     open-in-detail button navigates to the full task page. -->
			<div class="flex-1 flex overflow-hidden">
				<!-- Left: graph -->
				<div class="flex-1 min-w-0 overflow-hidden relative">
					<GraphView
						{spaceId}
						tasks={flatTasks}
						deps={depsMap}
						{stationMap}
						rootTaskId={tree?.id}
						onselect={handleGraphNodeSelect}
						onmutate={handleTreeMutate}
					/>

					<!-- Expand button (desktop) — shown only when the panel is collapsed -->
					{#if !graphPanelOpen}
						<button
							onclick={() => (graphPanelOpen = true)}
							class="hidden lg:flex absolute top-3 right-3 z-10 items-center justify-center h-7 w-7 rounded border border-border bg-background text-muted-foreground shadow-sm hover:text-foreground transition-colors"
							title="Show details"
							aria-label="Show details"
							data-testid="graph-panel-expand"
						>
							<PanelLeftOpen class="size-4" />
						</button>
					{/if}
				</div>

				<!-- Right: read-only detail panel (desktop sidebar) -->
				{#if graphPanelOpen}
					<div class="hidden lg:flex w-[400px] flex-col border-l border-border overflow-y-auto shrink-0">
						<TaskDetailPanel
							task={graphPanelTask ?? tree ?? undefined}
							{spaceId}
							{stationMap}
							{taskTitleMap}
							deps={graphPanelDeps}
							isLoading={graphPanelLoading}
							onaction={handleTaskAction}
							oncomplete={handleCompletion}
							parentName={tree?.title}
							parentType={isJob ? 'job' : undefined}
							onnavparent={handleNavToRoot}
							onselecttask={handleSelectTaskInGraph}
							hideSubTasks={graphPanelIsRoot}
							oncollapse={() => (graphPanelOpen = false)}
							onopendetail={handleOpenPanelDetail}
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
								onaction={handleTaskAction}
								oncomplete={handleCompletion}
								parentName={tree?.title}
								parentType={isJob ? 'job' : undefined}
								onnavparent={handleNavToRoot}
								onselecttask={handleSelectTaskInGraph}
								hideSubTasks={graphPanelIsRoot}
								onopendetail={handleOpenPanelDetail}
							/>
						</div>
					</div>
				{/if}
			</div>

		{:else}
			<!-- Overview tab: detail panel (left) + child-task List (right) -->
			<div class="flex-1 flex overflow-hidden">
				<!-- Left: Detail panel (desktop sidebar) -->
				{#if graphPanelOpen}
					<div class="hidden lg:flex w-1/2 flex-col border-r border-border overflow-y-auto shrink-0">
						<TaskDetailPanel
							task={graphPanelTask ?? tree ?? undefined}
							{spaceId}
							{stationMap}
							{taskTitleMap}
							deps={graphPanelDeps}
							isLoading={graphPanelLoading}
							onaction={handleTaskAction}
							oncomplete={handleCompletion}
							parentName={tree?.title}
							parentType={isJob ? 'job' : undefined}
							onnavparent={handleNavToRoot}
							onselecttask={handleSelectTaskInGraph}
							hideSubTasks={graphPanelIsRoot}
							oncollapse={() => (graphPanelOpen = false)}
						/>
					</div>
				{/if}

				<!-- Right: child-task list (no view-switcher header) -->
				<div class="flex-1 min-w-0 overflow-hidden flex flex-col relative">
					<div class="flex-1 overflow-hidden">
						<ListView {spaceId} tasks={flatTasks} deps={depsMap} {stationMap} />
					</div>

					<!-- Expand button (desktop) — shown only when the panel is collapsed -->
					{#if !graphPanelOpen}
						<button
							onclick={() => (graphPanelOpen = true)}
							class="hidden lg:flex absolute top-3 right-3 z-10 items-center justify-center h-7 w-7 rounded border border-border bg-background text-muted-foreground shadow-sm hover:text-foreground transition-colors"
							title="Expand panel"
							aria-label="Expand panel"
							data-testid="panel-expand"
						>
							<PanelLeftOpen class="size-4" />
						</button>
					{/if}
				</div>

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
								onaction={handleTaskAction}
								oncomplete={handleCompletion}
								parentName={tree?.title}
								parentType={isJob ? 'job' : undefined}
								onnavparent={handleNavToRoot}
								onselecttask={handleSelectTaskInGraph}
								hideSubTasks={graphPanelIsRoot}
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

<!-- Delete Task Dialog -->
<Dialog.Root bind:open={showDeleteDialog}>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Delete task?</Dialog.Title>
			<Dialog.Description>
				Delete "<strong>{tree?.title ?? 'this task'}</strong>"?
				{#if descendantCount > 0}
					This will also delete its {descendantCount} sub-task{descendantCount === 1 ? '' : 's'}.
				{/if}
				{#if selectedDeps && selectedDeps.blockers.length > 0 && selectedDeps.dependents.length > 0}
					Upstream and downstream dependencies will be reconnected automatically.
				{/if}
				This action cannot be undone.
			</Dialog.Description>
		</Dialog.Header>
		<Dialog.Footer class="pt-2">
			<Button type="button" variant="outline" onclick={() => (showDeleteDialog = false)}>
				Cancel
			</Button>
			<Button
				type="button"
				variant="destructive"
				disabled={isDeleting}
				onclick={handleDelete}
				data-testid="delete-task-confirm"
			>
				<Trash2 class="size-4 mr-1.5" />
				{isDeleting ? 'Deleting...' : 'Delete'}
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
