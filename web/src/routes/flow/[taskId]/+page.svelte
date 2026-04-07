<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { taskApi } from '$lib/api/task';
	import type { TaskTreeResponse, TaskDepsResponse } from '$lib/api/task';
	import type { TaskResponse } from '$lib/types/task';
	import { stationApi } from '$lib/api/station';
	import type { StationResponse } from '$lib/types/station';
	import TaskTree from '$lib/components/flow/TaskTree.svelte';
	import TaskDetailPanel from '$lib/components/flow/TaskDetailPanel.svelte';
	import BoardView from '$lib/components/flow/BoardView.svelte';
	import GraphView from '$lib/components/flow/GraphView.svelte';
	import ListView from '$lib/components/flow/ListView.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Separator } from '$lib/components/ui/separator';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb';
	import { ArrowLeft, AlertCircle, TreePine, LayoutGrid, GitBranch, List } from 'lucide-svelte';
	import { isEditableTarget } from '$lib/utils/keyboard.svelte';

	let taskId = $derived($page.params.taskId);

	let tree = $state<TaskTreeResponse | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let stationMap = $state<Map<string, string>>(new Map());

	// Selection state (for tree view)
	let selectedTask = $state<TaskTreeResponse | null>(null);
	let selectedDeps = $state<TaskDepsResponse | null>(null);

	// ---- View mode ----
	type ViewMode = 'tree' | 'board' | 'graph' | 'list';
	const VIEW_MODES: { value: ViewMode; label: string; icon: typeof LayoutGrid }[] = [
		{ value: 'tree', label: 'Tree', icon: TreePine },
		{ value: 'board', label: 'Board', icon: LayoutGrid },
		{ value: 'graph', label: 'Graph', icon: GitBranch },
		{ value: 'list', label: 'List', icon: List },
	];

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
		depsLoaded = true;
	}

	// Load deps when switching to graph view
	$effect(() => {
		if (currentView === 'graph' && !depsLoaded && tree) {
			loadDepsForDescendants();
		}
	});

	// ---- Keyboard shortcuts ----

	function handleKeydown(e: KeyboardEvent): void {
		if (isEditableTarget(e)) return;

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
		}
	}

	/** Build breadcrumb path from root to the selected task by walking the tree. */
	function buildBreadcrumb(
		root: TaskTreeResponse,
		targetId: string,
	): TaskTreeResponse[] {
		const path: TaskTreeResponse[] = [];

		function walk(node: TaskTreeResponse): boolean {
			path.push(node);
			if (node.id === targetId) return true;
			if (node.children) {
				for (const child of node.children) {
					if (walk(child)) return true;
				}
			}
			path.pop();
			return false;
		}

		walk(root);
		return path;
	}

	let breadcrumbPath = $derived(
		tree && selectedTask && currentView === 'tree'
			? buildBreadcrumb(tree, selectedTask.id)
			: [],
	);

	async function handleSelect(task: TaskTreeResponse): Promise<void> {
		selectedTask = task;
		// Load deps for the selected task
		selectedDeps = null;
		try {
			selectedDeps = await taskApi.getTaskDeps(task.id);
		} catch {
			// Deps endpoint may fail — degrade gracefully
			selectedDeps = null;
		}
	}

	/** After a task action, reload the tree to reflect the new state. */
	async function handleTaskAction(updated: TaskResponse): Promise<void> {
		if (!taskId) return;
		try {
			const newTree = await taskApi.getTaskTree(taskId);
			tree = newTree;
			// Reset deps cache so graph view re-fetches on next switch
			depsLoaded = false;
			depsMap = new Map();
			// Re-select the same task within the new tree
			if (selectedTask) {
				const found = findNode(newTree, selectedTask.id);
				if (found) {
					selectedTask = found;
					try {
						selectedDeps = await taskApi.getTaskDeps(found.id);
					} catch {
						selectedDeps = null;
					}
				}
			}
		} catch {
			// Silently fail — user can refresh manually
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
				taskApi.getTaskTree(taskId),
				stationApi.listStations().catch(() => [] as StationResponse[]),
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
				selectedDeps = await taskApi.getTaskDeps(tree.id);
			} catch {
				selectedDeps = null;
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load task tree';
		} finally {
			isLoading = false;
		}
	});

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
	<!-- Top bar: back link + view switcher + breadcrumb -->
	<div class="px-4 sm:px-6 py-3 border-b border-border shrink-0">
		<div class="flex items-center gap-4">
			<a
				href="/flow"
				class="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors shrink-0"
			>
				<ArrowLeft class="w-4 h-4" />
				Back
			</a>

			<!-- View mode switcher (segmented control) -->
			<div class="flex items-center rounded-lg border bg-muted/50 p-0.5">
				{#each VIEW_MODES as mode (mode.value)}
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

			{#if breadcrumbPath.length > 0}
				<Breadcrumb.Root>
					<Breadcrumb.List>
						{#each breadcrumbPath as crumb, i (crumb.id)}
							{#if i > 0}
								<Breadcrumb.Separator />
							{/if}
							<Breadcrumb.Item>
								{#if i < breadcrumbPath.length - 1}
									<Breadcrumb.Link
										href="#{crumb.id}"
										onclick={(e: MouseEvent) => { e.preventDefault(); handleSelect(crumb); }}
									>
										{crumb.title}
									</Breadcrumb.Link>
								{:else}
									<Breadcrumb.Page>{crumb.title}</Breadcrumb.Page>
								{/if}
							</Breadcrumb.Item>
						{/each}
					</Breadcrumb.List>
				</Breadcrumb.Root>
			{/if}
		</div>
	</div>

	<!-- Loading state -->
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

	<!-- Error state -->
	{:else if error}
		<div class="flex-1 flex items-center justify-center p-8">
			<div class="border border-destructive/30 bg-destructive/5 rounded-lg p-8 text-center max-w-md">
				<AlertCircle class="w-10 h-10 text-destructive mx-auto mb-3" />
				<h3 class="text-lg font-semibold text-foreground mb-1">Failed to load task</h3>
				<p class="text-sm text-muted-foreground mb-4">{error}</p>
				<Button variant="outline" size="sm" onclick={() => window.location.reload()}>
					Try Again
				</Button>
			</div>
		</div>

	<!-- Content views -->
	{:else if tree}
		{#if currentView === 'tree'}
			<!-- Tree + Detail split view (original) -->
			{@const rootProgress = computeRootProgress(tree)}

			<div class="flex-1 flex overflow-hidden">
				<!-- Left: Task tree -->
				<div class="w-2/5 border-r border-border overflow-y-auto p-4">
					<div class="mb-4">
						<h1 class="text-lg font-bold text-foreground truncate">{tree.title}</h1>
						{#if rootProgress.total > 0}
							<div class="mt-2">
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
						{/if}
					</div>

					<Separator class="mb-3" />

					{#if tree.children && tree.children.length > 0}
						<TaskTree
							nodes={tree.children}
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
			<ListView tasks={flatTasks} {stationMap} />
		{/if}
	{/if}
</div>
