<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { taskApi } from '$lib/api/task';
	import type { TaskTreeResponse, TaskDepsResponse } from '$lib/api/task';
	import { stationApi } from '$lib/api/station';
	import type { StationResponse } from '$lib/types/station';
	import TaskTree from '$lib/components/flow/TaskTree.svelte';
	import TaskDetailPanel from '$lib/components/flow/TaskDetailPanel.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Separator } from '$lib/components/ui/separator';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb';
	import { ArrowLeft, AlertCircle } from 'lucide-svelte';

	let taskId = $derived($page.params.taskId);

	let tree = $state<TaskTreeResponse | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let stationMap = $state<Map<string, string>>(new Map());

	// Selection state
	let selectedTask = $state<TaskTreeResponse | null>(null);
	let selectedDeps = $state<TaskDepsResponse | null>(null);

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
		tree && selectedTask ? buildBreadcrumb(tree, selectedTask.id) : [],
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

<div class="flex-1 overflow-hidden flex flex-col">
	<!-- Top bar: back link + breadcrumb -->
	<div class="px-4 sm:px-6 py-3 border-b border-border shrink-0">
		<div class="flex items-center gap-4">
			<a
				href="/flow"
				class="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors shrink-0"
			>
				<ArrowLeft class="w-4 h-4" />
				Back
			</a>

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

	<!-- Main split view: tree (left 40%) | detail (right 60%) -->
	{:else if tree}
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
					/>
				{:else}
					<div class="flex items-center justify-center h-full text-sm text-muted-foreground">
						Select a task to view details
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>
