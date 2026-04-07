<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { SvelteFlow, Background, Controls, MiniMap } from '@xyflow/svelte';
	import type { Node, Edge, NodeTypes } from '@xyflow/svelte';
	import dagre from '@dagrejs/dagre';
	import { taskApi } from '$lib/api/task';
	import { stationApi } from '$lib/api/station';
	import type { TaskResponse, TaskStatus } from '$lib/types/task';
	import type { TaskDepsResponse } from '$lib/api/task';
	import type { StationResponse } from '$lib/types/station';
	import { Button } from '$lib/components/ui/button';
	import { RefreshCw, AlertCircle, Maximize2 } from 'lucide-svelte';
	import { isEditableTarget } from '$lib/utils/keyboard.svelte';
	import TaskNode from './TaskNode.svelte';

	import '@xyflow/svelte/dist/style.css';

	/** Optional pre-loaded tasks and deps. When provided, the graph uses these instead of fetching. */
	interface Props {
		tasks?: TaskResponse[];
		deps?: Map<string, TaskDepsResponse>;
		stationMap?: Map<string, string>;
	}

	let { tasks: externalTasks, deps: externalDeps, stationMap: externalStationMap }: Props = $props();

	/** Whether we're in scoped mode (tasks provided externally). */
	let isScoped = $derived(!!externalTasks);

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
	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let _internalStationMap = $state<Map<string, string>>(new Map());
	let stationMap = $derived(externalStationMap ?? _internalStationMap);
	let taskCount = $state(0);
	let edgeCount = $state(0);

	// For tracking node click timing (single vs double click)
	let lastClickTime = $state(0);
	let lastClickedNodeId = $state('');

	// ---- Derived filters from URL ----
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

	function getLayoutedElements(
		inputNodes: Node[],
		inputEdges: Edge[],
	): { nodes: Node[]; edges: Edge[] } {
		const g = new dagre.graphlib.Graph();
		g.setDefaultEdgeLabel(() => ({}));
		g.setGraph({ rankdir: 'LR', nodesep: 40, ranksep: 80, marginx: 20, marginy: 20 });

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
			const stations: StationResponse[] = await stationApi.listStations();
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
			const result = await taskApi.listTasks(params as Parameters<typeof taskApi.listTasks>[0]);
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
							const deps = await taskApi.getTaskDeps(t.id);
							return { taskId: t.id, deps };
						} catch {
							return { taskId: t.id, deps: { blockers: [], dependents: [] } };
						}
					}),
				);
				allDeps.push(...batchResults);
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
				},
			}));

			// Build edges from dependencies
			// Each dep has fromTaskId (blocker) → toTaskId (dependent)
			// Edge direction: blocker → dependent (arrow from blocker to dependent)
			const edgeSet = new Set<string>(); // dedupe
			const newEdges: Edge[] = [];

			for (const { deps } of allDeps) {
				// Use blockers: from = blocker, to = this task
				for (const dep of deps.blockers) {
					const edgeId = `${dep.fromTaskId}->${dep.toTaskId}`;
					if (edgeSet.has(edgeId)) continue;
					edgeSet.add(edgeId);

					// Only include edge if both nodes are in our task set
					if (!taskMap.has(dep.fromTaskId) || !taskMap.has(dep.toTaskId)) continue;

					const blockerTask = taskMap.get(dep.fromTaskId);
					const blockerStatus = blockerTask?.status ?? 'open';
					const isResolved = blockerStatus === 'done' || blockerStatus === 'skipped';

					newEdges.push({
						id: edgeId,
						source: dep.fromTaskId,
						target: dep.toTaskId,
						type: 'smoothstep',
						animated: !isResolved,
						style: `stroke: ${STATUS_EDGE_COLORS[blockerStatus]}; stroke-width: 2px; ${!isResolved ? 'stroke-dasharray: 5 5;' : ''}`,
						markerEnd: {
							type: 'arrowclosed' as unknown as import('@xyflow/system').MarkerType,
							color: STATUS_EDGE_COLORS[blockerStatus],
						},
					});
				}

				// Also use dependents to catch edges not covered by blockers
				for (const dep of deps.dependents) {
					const edgeId = `${dep.fromTaskId}->${dep.toTaskId}`;
					if (edgeSet.has(edgeId)) continue;
					edgeSet.add(edgeId);

					if (!taskMap.has(dep.fromTaskId) || !taskMap.has(dep.toTaskId)) continue;

					const blockerTask = taskMap.get(dep.fromTaskId);
					const blockerStatus = blockerTask?.status ?? 'open';
					const isResolved = blockerStatus === 'done' || blockerStatus === 'skipped';

					newEdges.push({
						id: edgeId,
						source: dep.fromTaskId,
						target: dep.toTaskId,
						type: 'smoothstep',
						animated: !isResolved,
						style: `stroke: ${STATUS_EDGE_COLORS[blockerStatus]}; stroke-width: 2px; ${!isResolved ? 'stroke-dasharray: 5 5;' : ''}`,
						markerEnd: {
							type: 'arrowclosed' as unknown as import('@xyflow/system').MarkerType,
							color: STATUS_EDGE_COLORS[blockerStatus],
						},
					});
				}
			}

			// Apply dagre layout
			const layouted = getLayoutedElements(newNodes, newEdges);
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
			goto(`/flow/${nodeId}`);
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
			},
		}));

		// Build edges from provided deps
		const edgeSet = new Set<string>();
		const newEdges: Edge[] = [];

		if (depsMap) {
			for (const [, deps] of depsMap) {
				for (const dep of [...deps.blockers, ...deps.dependents]) {
					const edgeId = `${dep.fromTaskId}->${dep.toTaskId}`;
					if (edgeSet.has(edgeId)) continue;
					edgeSet.add(edgeId);

					if (!taskMap.has(dep.fromTaskId) || !taskMap.has(dep.toTaskId)) continue;

					const blockerTask = taskMap.get(dep.fromTaskId);
					const blockerStatus = blockerTask?.status ?? 'open';
					const isResolved = blockerStatus === 'done' || blockerStatus === 'skipped';

					newEdges.push({
						id: edgeId,
						source: dep.fromTaskId,
						target: dep.toTaskId,
						type: 'smoothstep',
						animated: !isResolved,
						style: `stroke: ${STATUS_EDGE_COLORS[blockerStatus]}; stroke-width: 2px; ${!isResolved ? 'stroke-dasharray: 5 5;' : ''}`,
						markerEnd: {
							type: 'arrowclosed' as unknown as import('@xyflow/system').MarkerType,
							color: STATUS_EDGE_COLORS[blockerStatus],
						},
					});
				}
			}
		}

		const layouted = getLayoutedElements(newNodes, newEdges);
		nodes = layouted.nodes;
		edges = layouted.edges;
		taskCount = filtered.length;
		edgeCount = newEdges.length;
		isLoading = false;
		lastRefreshed = new Date();
	}

	// When external tasks change, rebuild the graph
	$effect(() => {
		if (externalTasks) {
			buildFromExternalData(externalTasks, externalDeps);
		}
	});

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
	});

	function formatLastRefreshed(date: Date | null): string {
		if (!date) return '';
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
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
					goto(`/flow/${selectedNodeId}`);
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
			<h1 class="text-lg font-semibold text-foreground">Graph</h1>
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
			<Button variant="outline" size="sm" onclick={handleManualRefresh} disabled={isRefreshing}>
				<RefreshCw class="size-4 {isRefreshing ? 'animate-spin' : ''}" />
				<span class="ml-1.5">Refresh</span>
			</Button>
		</div>
	</div>

	<!-- Error banner -->
	{#if error}
		<div class="mx-4 mb-2 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm">
			<AlertCircle class="size-4 text-destructive shrink-0" />
			<span class="text-destructive">{error}</span>
			<Button variant="outline" size="sm" class="ml-auto" onclick={() => fetchGraph()}>
				Retry
			</Button>
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
		<div class="flex-1">
			<SvelteFlow
				{nodes}
				{edges}
				{nodeTypes}
				fitView
				fitViewOptions={{ padding: 0.2 }}
				nodesDraggable={true}
				nodesConnectable={false}
				elementsSelectable={true}
				minZoom={0.1}
				maxZoom={2}
				colorMode="system"
				onnodeclick={handleNodeClick}
			>
				<Background />
				<Controls />
				<MiniMap
					nodeColor={(node) => {
						const status = (node.data as { status?: string }).status ?? 'open';
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
