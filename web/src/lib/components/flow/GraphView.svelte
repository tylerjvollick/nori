<script lang="ts">
	import { onMount, onDestroy, untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { SvelteFlow, Background, Controls, MiniMap } from '@xyflow/svelte';
	import type { Node, Edge, NodeTypes, Connection } from '@xyflow/svelte';
	import dagre from '@dagrejs/dagre';
	import { taskApi } from '$lib/api/task';
	import { stationApi } from '$lib/api/station';
	import { spaceStore } from '$lib/stores/space';
	import type { TaskResponse, TaskStatus } from '$lib/types/task';
	import type { TaskDepsResponse } from '$lib/api/task';
	import type { StationResponse } from '$lib/types/station';
	import { Button } from '$lib/components/ui/button';
	import { RefreshCw, CircleAlert, Maximize2, ArrowRight, ArrowDown, ArrowLeft, ArrowUp } from '@lucide/svelte';
	import { isEditableTarget } from '$lib/utils/keyboard.svelte';
	import { graphDirection, type GraphDirection } from '$lib/stores/graph';
	import TaskNode from './TaskNode.svelte';

	import '@xyflow/svelte/dist/style.css';

	/** Optional pre-loaded tasks and deps. When provided, the graph uses these instead of fetching. */
	interface Props {
		tasks?: TaskResponse[];
		deps?: Map<string, TaskDepsResponse>;
		stationMap?: Map<string, string>;
		/** When set, this task gets a highlighted ring in the graph (used for neighborhood view). */
		focusTaskId?: string;
	}

	let { tasks: externalTasks, deps: externalDeps, stationMap: externalStationMap, focusTaskId }: Props = $props();

	/** Whether we're in scoped mode (tasks provided externally). */
	let isScoped = $derived(!!externalTasks);

	let spaceId = $derived($spaceStore.currentSpace?.id ?? '');

	// ---- Constants ----
	const POLL_INTERVAL_MS = 30_000;
	const TASK_LIMIT = 500;
	const NODE_WIDTH = 180;
	const NODE_HEIGHT = 56;

	// ---- Custom node types ----
	const nodeTypes: NodeTypes = {
		task: TaskNode as unknown as NodeTypes[string],
	};

	// ---- State ----
	let nodes = $state<Node[]>([]);
	let edges = $state<Edge[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let isRefreshing = $state(false);
	let lastRefreshed = $state<Date | null>(null);
	let isConnecting = $state(false);
	let connectionError = $state<string | null>(null);
	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let _internalStationMap = $state<Map<string, string>>(new Map());
	let stationMap = $derived(externalStationMap ?? _internalStationMap);
	let taskCount = $state(0);
	let edgeCount = $state(0);

	// For tracking node click timing (single vs double click)
	let lastClickTime = $state(0);
	let lastClickedNodeId = $state('');

	// ---- Derived filters from URL ----
	let slug = $derived($page.params.slug);
	let stationFilter = $derived($page.url.searchParams.get('station') || '');
	let statusFilter = $derived($page.url.searchParams.get('status') || '');
	let priorityFilter = $derived($page.url.searchParams.get('priority') || '');

	// ---- Status → edge color (matches TaskNode) ----
	const STATUS_EDGE_COLORS: Record<TaskStatus, string> = {
		open: '#9ca3af', // gray-400
		active: '#3b82f6', // blue-500
		paused: '#eab308', // yellow-500
		done: '#22c55e', // green-500
		skipped: '#9ca3af', // gray-400
		cancelled: '#ef4444', // red-500
	};

	// ---- Dagre layout ----

	// Direction label for the toggle button tooltip
	const DIRECTION_LABELS: Record<GraphDirection, string> = {
		LR: 'Left → Right',
		RL: 'Right → Left',
		TB: 'Top → Bottom',
		BT: 'Bottom → Top',
	};

	const DIRECTION_ICONS: Record<GraphDirection, typeof ArrowRight> = {
		LR: ArrowRight,
		RL: ArrowLeft,
		TB: ArrowDown,
		BT: ArrowUp,
	};

	let currentDirection = $state<GraphDirection>('LR');
	const unsubDirection = graphDirection.subscribe((d) => (currentDirection = d));

	function getLayoutedElements(
		inputNodes: Node[],
		inputEdges: Edge[],
		direction: GraphDirection,
	): { nodes: Node[]; edges: Edge[] } {
		const g = new dagre.graphlib.Graph();
		g.setDefaultEdgeLabel(() => ({}));
		g.setGraph({ rankdir: direction, nodesep: 40, ranksep: 80, marginx: 20, marginy: 20 });

		for (const node of inputNodes) {
			const isJob = (node.data as { type?: string }).type === 'job';
			g.setNode(node.id, {
				width: isJob ? NODE_WIDTH + 20 : NODE_WIDTH,
				height: NODE_HEIGHT,
			});
		}

		for (const edge of inputEdges) {
			g.setEdge(edge.source, edge.target);
		}

		dagre.layout(g);

		const layoutedNodes = inputNodes.map((node) => {
			const nodeWithPosition = g.node(node.id);
			if (!nodeWithPosition) return node;

			const isJob = (node.data as { type?: string }).type === 'job';
			const w = isJob ? NODE_WIDTH + 20 : NODE_WIDTH;

			return {
				...node,
				position: {
					x: nodeWithPosition.x - w / 2,
					y: nodeWithPosition.y - NODE_HEIGHT / 2,
				},
			};
		});

		return { nodes: layoutedNodes, edges: inputEdges };
	}

	// ---- Data fetching ----

	async function fetchStations(): Promise<void> {
		try {
			const stations: StationResponse[] = await stationApi.listStations(spaceId);
			const map = new Map<string, string>();
			for (const s of stations) {
				map.set(s.id, s.name);
			}
			_internalStationMap = map;
		} catch {
			// Stations endpoint may not exist yet — gracefully degrade.
		}
	}

	async function fetchGraph(opts?: { silent?: boolean }): Promise<void> {
		if (!opts?.silent) {
			isLoading = true;
		}
		isRefreshing = true;
		error = null;

		try {
			// Build query params from filters
			const params: Record<string, string | number> = { limit: TASK_LIMIT };
			if (stationFilter) params.stationId = stationFilter;
			if (statusFilter) params.status = statusFilter;

			// Fetch all tasks
			const result = await taskApi.listTasks(spaceId, params as Parameters<typeof taskApi.listTasks>[1]);
			let tasks = result.items;

			// Apply priority filter client-side
			if (priorityFilter) {
				const p = Number(priorityFilter);
				tasks = tasks.filter((t) => t.priority === p);
			}

			if (tasks.length === 0) {
				nodes = [];
				edges = [];
				taskCount = 0;
				edgeCount = 0;
				lastRefreshed = new Date();
				return;
			}

			// Build a lookup of tasks by ID for quick access
			const taskMap = new Map<string, TaskResponse>();
			for (const t of tasks) {
				taskMap.set(t.id, t);
			}

			// Fetch deps for all tasks in parallel (batched to avoid overwhelming the server)
			const BATCH_SIZE = 20;
			const allDeps: { taskId: string; deps: TaskDepsResponse }[] = [];

			for (let i = 0; i < tasks.length; i += BATCH_SIZE) {
				const batch = tasks.slice(i, i + BATCH_SIZE);
				const batchResults = await Promise.all(
					batch.map(async (t) => {
						try {
							const deps = await taskApi.getTaskDeps(spaceId, t.id);
							return { taskId: t.id, deps };
						} catch {
							return { taskId: t.id, deps: { blockers: [], dependents: [] } };
						}
					}),
				);
				allDeps.push(...batchResults);
			}

		// Compute which tasks are blocked (have at least one unresolved dependency)
		const blockedTaskIds = new Set<string>();
		for (const { taskId, deps } of allDeps) {
			for (const dep of deps.blockers) {
				// In blockers: fromTaskId = this task (blocked), toTaskId = the blocker
				const blockerTask = taskMap.get(dep.toTaskId);
				const blockerStatus = blockerTask?.status ?? 'open';
				if (blockerStatus !== 'done' && blockerStatus !== 'skipped') {
					blockedTaskIds.add(taskId);
					break;
				}
			}
		}

		// Build nodes
		const newNodes: Node[] = tasks.map((task) => ({
			id: task.id,
			type: 'task',
			position: { x: 0, y: 0 }, // will be set by dagre
			data: {
				title: task.title,
				taskId: task.id,
				status: task.status,
				type: task.type,
				priority: task.priority,
				stationName: task.stationId ? stationMap.get(task.stationId) : undefined,
				isBlocked: blockedTaskIds.has(task.id),
				direction: currentDirection,
			},
		}));

			// Build edges from dependencies
			// In blockers: fromTaskId = blocked/downstream, toTaskId = blocker/upstream
			// In dependents: fromTaskId = downstream task, toTaskId = this task (upstream)
			// Edge direction: blocker → dependent (arrow from upstream to downstream)
			const edgeSet = new Set<string>(); // dedupe
			const newEdges: Edge[] = [];

			for (const { deps } of allDeps) {
				// Use blockers: fromTaskId = this task (blocked), toTaskId = blocker
				for (const dep of deps.blockers) {
					const edgeId = `${dep.toTaskId}->${dep.fromTaskId}`;
					if (edgeSet.has(edgeId)) continue;
					edgeSet.add(edgeId);

					// Only include edge if both nodes are in our task set
					if (!taskMap.has(dep.fromTaskId) || !taskMap.has(dep.toTaskId)) continue;

					const blockerTask = taskMap.get(dep.toTaskId);
					const blockerStatus = blockerTask?.status ?? 'open';
					const isResolved = blockerStatus === 'done' || blockerStatus === 'skipped';

					newEdges.push({
						id: edgeId,
						source: dep.toTaskId,
						target: dep.fromTaskId,
						type: 'smoothstep',
						animated: !isResolved,
						style: `stroke: ${STATUS_EDGE_COLORS[blockerStatus]}; stroke-width: 2px; ${!isResolved ? 'stroke-dasharray: 5 5;' : ''}`,
						markerEnd: {
							type: 'arrowclosed' as unknown as import('@xyflow/system').MarkerType,
							color: STATUS_EDGE_COLORS[blockerStatus],
						},
						data: { depId: dep.id },
					});
				}

				// Also use dependents to catch edges not covered by blockers
				// In dependents: fromTaskId = downstream task, toTaskId = this task (upstream/blocker)
				for (const dep of deps.dependents) {
					const edgeId = `${dep.toTaskId}->${dep.fromTaskId}`;
					if (edgeSet.has(edgeId)) continue;
					edgeSet.add(edgeId);

					if (!taskMap.has(dep.fromTaskId) || !taskMap.has(dep.toTaskId)) continue;

					const blockerTask = taskMap.get(dep.toTaskId);
					const blockerStatus = blockerTask?.status ?? 'open';
					const isResolved = blockerStatus === 'done' || blockerStatus === 'skipped';

					newEdges.push({
						id: edgeId,
						source: dep.toTaskId,
						target: dep.fromTaskId,
						type: 'smoothstep',
						animated: !isResolved,
						style: `stroke: ${STATUS_EDGE_COLORS[blockerStatus]}; stroke-width: 2px; ${!isResolved ? 'stroke-dasharray: 5 5;' : ''}`,
						markerEnd: {
							type: 'arrowclosed' as unknown as import('@xyflow/system').MarkerType,
							color: STATUS_EDGE_COLORS[blockerStatus],
						},
						data: { depId: dep.id },
					});
				}
			}

			// Apply dagre layout
			const layouted = getLayoutedElements(newNodes, newEdges, currentDirection);
			nodes = layouted.nodes;
			edges = layouted.edges;
			taskCount = tasks.length;
			edgeCount = newEdges.length;
			lastRefreshed = new Date();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load graph data';
		} finally {
			isLoading = false;
			isRefreshing = false;
		}
	}

	// ---- Polling ----

	function startPolling(): void {
		stopPolling();
		pollTimer = setInterval(() => fetchGraph({ silent: true }), POLL_INTERVAL_MS);
	}

	function stopPolling(): void {
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	function handleManualRefresh(): void {
		fetchGraph({ silent: true });
	}

	// ---- Node click handling ----

	function handleNodeClick({ node }: { node: Node; event: MouseEvent | TouchEvent }): void {
		const nodeId = node.id;
		const now = Date.now();

		// Double-click detection (within 400ms on the same node)
		if (lastClickedNodeId === nodeId && now - lastClickTime < 400) {
			// Navigate to task detail
			goto(`/spaces/${slug}/${nodeId}`);
			lastClickTime = 0;
			lastClickedNodeId = '';
			return;
		}

		lastClickTime = now;
		lastClickedNodeId = nodeId;
	}

	// ---- Re-fetch when filters change ----
	let prevStation = $state('');
	let prevStatus = $state('');
	let prevPriority = $state('');

	$effect(() => {
		const s = stationFilter;
		const st = statusFilter;
		const p = priorityFilter;
		if (s !== prevStation || st !== prevStatus || p !== prevPriority) {
			prevStation = s;
			prevStatus = st;
			prevPriority = p;
			if (lastRefreshed) {
				if (isScoped && externalTasks) {
					buildFromExternalData(externalTasks, externalDeps);
				} else {
					fetchGraph({ silent: true });
				}
			}
		}
	});

	// ---- Lifecycle ----

	/** Build graph from externally-provided tasks and deps. */
	function buildFromExternalData(tasks: TaskResponse[], depsMap?: Map<string, TaskDepsResponse>): void {
		// Apply filters
		let filtered = tasks;
		if (stationFilter) {
			filtered = filtered.filter((t) => t.stationId === stationFilter);
		}
		if (statusFilter) {
			filtered = filtered.filter((t) => t.status === statusFilter);
		}
		if (priorityFilter) {
			const p = Number(priorityFilter);
			filtered = filtered.filter((t) => t.priority === p);
		}

		if (filtered.length === 0) {
			nodes = [];
			edges = [];
			taskCount = 0;
			edgeCount = 0;
			isLoading = false;
			lastRefreshed = new Date();
			return;
		}

		const taskMap = new Map<string, TaskResponse>();
		for (const t of filtered) {
			taskMap.set(t.id, t);
		}

		// Compute which tasks are blocked (have at least one unresolved dependency)
		const blockedTaskIds = new Set<string>();
		if (depsMap) {
			for (const [taskId, deps] of depsMap) {
				for (const dep of deps.blockers) {
					// In blockers: fromTaskId = this task (blocked), toTaskId = the blocker
					const blockerTask = taskMap.get(dep.toTaskId);
					const blockerStatus = blockerTask?.status ?? 'open';
					if (blockerStatus !== 'done' && blockerStatus !== 'skipped') {
						blockedTaskIds.add(taskId);
						break;
					}
				}
			}
		}

		// Build nodes
		const newNodes: Node[] = filtered.map((task) => ({
			id: task.id,
			type: 'task',
			position: { x: 0, y: 0 },
			data: {
				title: task.title,
				taskId: task.id,
				status: task.status,
				type: task.type,
				priority: task.priority,
				stationName: task.stationId ? stationMap.get(task.stationId) : undefined,
				isFocus: focusTaskId === task.id,
				isBlocked: blockedTaskIds.has(task.id),
				direction: currentDirection,
			},
		}));

		// Build edges from provided deps
		// In both blockers and dependents: toTaskId = upstream/blocker, fromTaskId = downstream/blocked
		// Edge direction: upstream (source) → downstream (target)
		const edgeSet = new Set<string>();
		const newEdges: Edge[] = [];

		if (depsMap) {
			for (const [, deps] of depsMap) {
				for (const dep of [...deps.blockers, ...deps.dependents]) {
					const edgeId = `${dep.toTaskId}->${dep.fromTaskId}`;
					if (edgeSet.has(edgeId)) continue;
					edgeSet.add(edgeId);

					if (!taskMap.has(dep.fromTaskId) || !taskMap.has(dep.toTaskId)) continue;

					const blockerTask = taskMap.get(dep.toTaskId);
					const blockerStatus = blockerTask?.status ?? 'open';
					const isResolved = blockerStatus === 'done' || blockerStatus === 'skipped';

					newEdges.push({
						id: edgeId,
						source: dep.toTaskId,
						target: dep.fromTaskId,
						type: 'smoothstep',
						animated: !isResolved,
						style: `stroke: ${STATUS_EDGE_COLORS[blockerStatus]}; stroke-width: 2px; ${!isResolved ? 'stroke-dasharray: 5 5;' : ''}`,
						markerEnd: {
							type: 'arrowclosed' as unknown as import('@xyflow/system').MarkerType,
							color: STATUS_EDGE_COLORS[blockerStatus],
						},
						data: { depId: dep.id },
					});
				}
			}
		}

		const layouted = getLayoutedElements(newNodes, newEdges, currentDirection);
		nodes = layouted.nodes;
		edges = layouted.edges;
		taskCount = filtered.length;
		edgeCount = newEdges.length;
		isLoading = false;
		lastRefreshed = new Date();
	}

	// When external tasks or deps change, rebuild the graph.
	// Both externalTasks and externalDeps must be read directly in the $effect
	// body so Svelte 5 tracks them as dependencies. Previously only externalTasks
	// was tracked, so the graph never re-rendered when deps loaded asynchronously.
	$effect(() => {
		const tasks = externalTasks;
		const deps = externalDeps;
		if (tasks) {
			buildFromExternalData(tasks, deps);
		}
	});

	// Re-layout when graph direction changes.
	// dirTracker.prev starts empty; on first effect run, it learns the current direction
	// without re-laying out (avoiding a redundant initial layout).
	const dirTracker = { prev: '' };
	$effect(() => {
		const dir = currentDirection;
		if (dirTracker.prev && dir !== dirTracker.prev) {
			untrack(() => relayoutForDirection(dir));
		}
		dirTracker.prev = dir;
	});

	function relayoutForDirection(dir: GraphDirection): void {
		if (nodes.length === 0) return;
		// Update direction in all node data so TaskNode can adjust handle positions
		const updatedNodes = nodes.map((n) => ({
			...n,
			data: { ...n.data, direction: dir },
		}));
		const layouted = getLayoutedElements(updatedNodes, edges, dir);
		nodes = layouted.nodes;
		edges = layouted.edges;
	}

	onMount(async () => {
		if (isScoped) {
			// In scoped mode, tasks are provided externally. No fetching needed.
			return;
		}
		await Promise.all([fetchGraph(), fetchStations()]);
		startPolling();
	});

	onDestroy(() => {
		stopPolling();
		unsubDirection();
	});

	function formatLastRefreshed(date: Date | null): string {
		if (!date) return '';
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	// ---- Cycle detection ----

	function hasCycle(sourceId: string, targetId: string): boolean {
		// Check if adding source → target would create a cycle.
		// A cycle exists iff there is already a path from targetId to sourceId.
		const visited = new Set<string>();
		const queue: string[] = [targetId];
		while (queue.length > 0) {
			const current = queue.shift()!;
			if (current === sourceId) return true;
			if (visited.has(current)) continue;
			visited.add(current);
			for (const edge of edges) {
				if (edge.source === current) {
					queue.push(edge.target);
				}
			}
		}
		return false;
	}

	// ---- Edge creation/deletion ----

	async function handleConnect(connection: Connection): Promise<void> {
		const source = connection.source;
		const target = connection.target;
		if (!source || !target || source === target) return;

		// Duplicate check
		if (edges.some((e) => e.source === source && e.target === target)) return;

		// Cycle guard
		if (hasCycle(source, target)) {
			connectionError = 'Cannot create dependency: this would create a cycle';
			setTimeout(() => { connectionError = null; }, 4000);
			return;
		}

		isConnecting = true;
		connectionError = null;
		try {
			await taskApi.addDep(spaceId, source, target, 'finish_to_start');
			if (isScoped && externalTasks) {
				buildFromExternalData(externalTasks, externalDeps);
			} else {
				await fetchGraph({ silent: true });
			}
		} catch (e) {
			connectionError = e instanceof Error ? e.message : 'Failed to create dependency';
			setTimeout(() => { connectionError = null; }, 4000);
		} finally {
			isConnecting = false;
		}
	}

	async function handleBeforeDelete({ nodes: _deletedNodes, edges: deletedEdges }: { nodes: Node[]; edges: Edge[] }): Promise<boolean> {
		if (deletedEdges.length === 0) return true;

		const failures: string[] = [];
		await Promise.all(
			deletedEdges.map(async (edge) => {
				const depId = (edge.data as { depId?: string } | undefined)?.depId;
				if (!depId) return;
				try {
					await taskApi.removeDep(spaceId, edge.source, depId);
				} catch (e) {
					failures.push(e instanceof Error ? e.message : 'Failed to remove dependency');
				}
			}),
		);

		if (failures.length > 0) {
			connectionError = failures[0];
			setTimeout(() => { connectionError = null; }, 4000);
			return false; // Cancel xyflow deletion so edges stay visible
		}

		// Update our controlled edges state to match xyflow's pending deletion
		const deletedIds = new Set(deletedEdges.map((e) => e.id));
		edges = edges.filter((e) => !deletedIds.has(e.id));
		return true;
	}

	// ---- Keyboard navigation ----
	// useSvelteFlow provides zoomIn/zoomOut/fitView on the flow instance.
	// It must be called inside a component that is a child of <SvelteFlow>,
	// but since we render <SvelteFlow> in this same component, we'll use
	// a direct approach with the nodes state and store refs.

	let selectedNodeId = $derived.by(() => {
		const sel = nodes.find((n) => n.selected);
		return sel?.id ?? null;
	});

	function handleKeydown(e: KeyboardEvent): void {
		if (isEditableTarget(e)) return;

		switch (e.key) {
			case 'Enter': {
				if (selectedNodeId) {
					e.preventDefault();
					goto(`/spaces/${slug}/${selectedNodeId}`);
				}
				break;
			}
			case '+':
			case '=': {
				e.preventDefault();
				// Zoom in — dispatch custom event handled by flow controls
				// We can't use useSvelteFlow here since we're not inside the
				// SvelteFlow context. Use the xyflow controls button directly.
				const zoomInBtn = document.querySelector(
					'.svelte-flow__controls button[title="zoom in"]',
				) as HTMLButtonElement;
				zoomInBtn?.click();
				break;
			}
			case '-': {
				e.preventDefault();
				const zoomOutBtn = document.querySelector(
					'.svelte-flow__controls button[title="zoom out"]',
				) as HTMLButtonElement;
				zoomOutBtn?.click();
				break;
			}
			case '0': {
				e.preventDefault();
				const fitViewBtn = document.querySelector(
					'.svelte-flow__controls button[title="fit view"]',
				) as HTMLButtonElement;
				fitViewBtn?.click();
				break;
			}
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="flex h-full flex-col overflow-hidden">
	<!-- Graph header -->
	<div class="flex-shrink-0 flex items-center justify-between px-4 py-2">
		<div class="flex items-center gap-3">
			<h1 class="text-lg font-semibold text-foreground">{focusTaskId ? 'Neighborhood' : 'Graph'}</h1>
			{#if !isLoading && taskCount > 0}
				<span class="text-xs text-muted-foreground">
					{taskCount} tasks, {edgeCount} dependencies
				</span>
			{/if}
		</div>
		<div class="flex items-center gap-3">
			{#if lastRefreshed}
				<span class="text-xs text-muted-foreground">
					Updated {formatLastRefreshed(lastRefreshed)}
				</span>
			{/if}
			<Button
				variant="outline"
				size="sm"
				onclick={() => graphDirection.cycle()}
				title="Graph direction: {DIRECTION_LABELS[currentDirection]}"
			>
				{@const DirIcon = DIRECTION_ICONS[currentDirection]}
				<DirIcon class="size-4" />
				<span class="ml-1.5">{DIRECTION_LABELS[currentDirection]}</span>
			</Button>
			<Button variant="outline" size="sm" onclick={handleManualRefresh} disabled={isRefreshing}>
				<RefreshCw class="size-4 {isRefreshing ? 'animate-spin' : ''}" />
				<span class="ml-1.5">Refresh</span>
			</Button>
		</div>
	</div>

	<!-- Error banner -->
	{#if error}
		<div class="mx-4 mb-2 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm">
			<CircleAlert class="size-4 text-destructive shrink-0" />
			<span class="text-destructive">{error}</span>
			<Button variant="outline" size="sm" class="ml-auto" onclick={() => fetchGraph()}>
				Retry
			</Button>
		</div>
	{/if}

	<!-- Connection error toast -->
	{#if connectionError}
		<div class="mx-4 mb-2 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm">
			<CircleAlert class="size-4 text-destructive shrink-0" />
			<span class="text-destructive">{connectionError}</span>
		</div>
	{/if}

	<!-- Graph area -->
	{#if isLoading}
		<div class="flex flex-1 items-center justify-center">
			<div class="flex flex-col items-center gap-3">
				<RefreshCw class="size-6 animate-spin text-muted-foreground" />
				<p class="text-sm text-muted-foreground">Loading dependency graph...</p>
			</div>
		</div>
	{:else if taskCount === 0}
		<div class="flex flex-1 items-center justify-center text-muted-foreground">
			<div class="flex flex-col items-center gap-2">
				<Maximize2 class="size-8 text-muted-foreground/50" />
				<p class="text-sm">No tasks to display</p>
				<p class="text-xs text-muted-foreground/70">
					{#if stationFilter || statusFilter || priorityFilter}
						Try adjusting your filters
					{:else}
						Create some tasks to see the dependency graph
					{/if}
				</p>
			</div>
		</div>
	{:else}
		<div class="flex-1 min-h-0">
			<SvelteFlow
				{nodes}
				{edges}
				{nodeTypes}
				fitView
				fitViewOptions={{ padding: 0.2 }}
				nodesDraggable={true}
				nodesConnectable={true}
				elementsSelectable={true}
				minZoom={0.1}
				maxZoom={2}
				colorMode="system"
				onnodeclick={handleNodeClick}
				onconnect={handleConnect}
				onbeforedelete={handleBeforeDelete}
			>
				<Background />
				<Controls />
				<MiniMap
					nodeColor={(node) => {
						const nodeData = node.data as { status?: string; isBlocked?: boolean };
						const status = nodeData.status ?? 'open';
						// Blocked open tasks show red in the minimap
						if (nodeData.isBlocked && status === 'open') return '#ef4444';
						const colors: Record<string, string> = {
							open: '#9ca3af',
							active: '#3b82f6',
							paused: '#eab308',
							done: '#22c55e',
							skipped: '#9ca3af',
							cancelled: '#ef4444',
						};
						return colors[status] ?? '#9ca3af';
					}}
				/>
			</SvelteFlow>
		</div>
	{/if}
</div>
