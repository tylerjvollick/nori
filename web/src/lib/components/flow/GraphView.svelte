<script lang="ts">
	import { onMount, onDestroy, untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { SvelteFlow, Background, Controls, MiniMap } from '@xyflow/svelte';
	import type { Node, Edge, NodeTypes, Connection } from '@xyflow/svelte';
	import dagre from '@dagrejs/dagre';
	import { taskApi } from '$lib/api/task';
	import { stationApi } from '$lib/api/station';
	import type { TaskResponse, TaskStatus } from '$lib/types/task';
	import type { TaskDepsResponse } from '$lib/api/task';
	import type { StationResponse } from '$lib/types/station';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { RefreshCw, CircleAlert, Maximize2, ArrowRight, ArrowDown, ArrowLeft, ArrowUp, Plus, Trash2, LayoutDashboard } from '@lucide/svelte';
	import { isEditableTarget } from '$lib/utils/keyboard.svelte';
	import { graphDirection, type GraphDirection } from '$lib/stores/graph';
	import TaskNode from './TaskNode.svelte';

	import '@xyflow/svelte/dist/style.css';

	/** Optional pre-loaded tasks and deps. When provided, the graph uses these instead of fetching. */
	interface Props {
		spaceId: string;
		tasks?: TaskResponse[];
		deps?: Map<string, TaskDepsResponse>;
		stationMap?: Map<string, string>;
		/** When set, this task gets a highlighted ring in the graph (used for neighborhood view). */
		focusTaskId?: string;
		/** Called when a node is single-clicked. Receives the task ID. */
		onselect?: (taskId: string) => void;
		/**
		 * 'recipe': disables double-click navigation; new tasks are created as children of rootTaskId.
		 * 'task' (default): standard behavior.
		 */
		mode?: 'task' | 'recipe';
		/** When mode='recipe', new nodes are created as children of this task ID. */
		rootTaskId?: string;
		/** Called after a graph mutation (node/edge add/delete) so the parent can re-fetch data. */
		onmutate?: () => Promise<void> | void;
	}

	let { spaceId, tasks: externalTasks, deps: externalDeps, stationMap: externalStationMap, focusTaskId, onselect, mode = 'task', rootTaskId, onmutate }: Props = $props();

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
	let isConnecting = $state(false);
	let connectionError = $state<string | null>(null);
	let editingNodeId = $state<string | null>(null);
	let deleteConfirmState = $state<{
		nodeId: string;
		nodeTitle: string;
		hasConnections: boolean;
		resolve: (ok: boolean) => void;
	} | null>(null);
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
				onAddSerial: () => handleAddSerial(task.id),
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
			if (mode !== 'recipe') {
				// Navigate to task detail (task mode only)
				goto(`/spaces/${slug}/${nodeId}`);
			}
			lastClickTime = 0;
			lastClickedNodeId = '';
			return;
		}

		lastClickTime = now;
		lastClickedNodeId = nodeId;
		onselect?.(nodeId);
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
				onAddSerial: () => handleAddSerial(task.id),
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

	function handleRelayout(): void {
		relayoutForDirection(currentDirection);
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
			await taskApi.addDep(spaceId, source, target, 'blocks');
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

	async function handleBeforeDelete({ nodes: deletedNodes, edges: deletedEdges }: { nodes: Node[]; edges: Edge[] }): Promise<boolean> {
		// Handle node deletion
		if (deletedNodes.length > 0) {
			for (const node of deletedNodes) {
				const upstreamEdges = edges.filter((e) => e.target === node.id);
				const downstreamEdges = edges.filter((e) => e.source === node.id);
				const hasConnections = upstreamEdges.length > 0 || downstreamEdges.length > 0;
				const nodeTitle = (node.data as { title?: string }).title ?? node.id;

				const confirmed = await new Promise<boolean>((resolve) => {
					deleteConfirmState = { nodeId: node.id, nodeTitle, hasConnections, resolve };
				});

				if (!confirmed) return false;

				// Reconnect: add a dep from each upstream to each downstream
				for (const upEdge of upstreamEdges) {
					for (const downEdge of downstreamEdges) {
						try {
							await taskApi.addDep(spaceId, upEdge.source, downEdge.target, 'blocks');
						} catch {
							// Ignore duplicate dep errors
						}
					}
				}

				// Delete the task
				try {
					await taskApi.deleteTask(spaceId, node.id);
				} catch (e) {
					connectionError = e instanceof Error ? e.message : 'Failed to delete task';
					setTimeout(() => { connectionError = null; }, 4000);
					return false;
				}
			}

			// Refresh graph after node deletion
			if (isScoped && externalTasks) {
				buildFromExternalData(externalTasks, externalDeps);
			} else {
				await fetchGraph({ silent: true });
			}
			return false; // We've already updated state via fetch
		}

		// Handle edge deletion
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

	// ---- Node creation ----

	/** After graph refresh, patch a specific node to show inline title editing. */
	function activateNodeEditing(taskId: string): void {
		editingNodeId = taskId;
		nodes = nodes.map((n) =>
			n.id === taskId
				? {
						...n,
						data: {
							...n.data,
							editing: true,
							onTitleCommit: (title: string) => handleTitleCommit(taskId, title),
						},
					}
				: n,
		);
	}

	async function refreshGraph(): Promise<void> {
		if (isScoped && onmutate) {
			await onmutate();
		} else if (isScoped && externalTasks) {
			buildFromExternalData(externalTasks, externalDeps);
		} else {
			await fetchGraph({ silent: true });
		}
	}

	async function handleAddUnconnected(): Promise<void> {
		if (!spaceId) return;
		try {
			const newTask = await taskApi.createTask(spaceId, {
				title: 'New Task',
				type: 'task',
				priority: 2,
				...(mode === 'recipe' && rootTaskId ? { parentId: rootTaskId } : {}),
			});
			await refreshGraph();
			activateNodeEditing(newTask.id);
		} catch (e) {
			connectionError = e instanceof Error ? e.message : 'Failed to create task';
			setTimeout(() => { connectionError = null; }, 4000);
		}
	}

	async function handleAddSerial(sourceNodeId: string): Promise<void> {
		if (!spaceId) return;
		// Capture downstream edges before we modify state
		const downstreamEdges = edges.filter((e) => e.source === sourceNodeId);
		try {
			const newTask = await taskApi.createTask(spaceId, {
				title: 'New Task',
				type: 'task',
				priority: 2,
				...(mode === 'recipe' && rootTaskId ? { parentId: rootTaskId } : {}),
			});

			// sourceNode → newTask
			await taskApi.addDep(spaceId, sourceNodeId, newTask.id, 'blocks');

			// Reconnect downstream: replace sourceNode → X with newTask → X
			for (const downEdge of downstreamEdges) {
				const depId = (downEdge.data as { depId?: string })?.depId;
				if (depId) {
					try {
						await taskApi.removeDep(spaceId, sourceNodeId, depId);
					} catch { /* ignore */ }
					await taskApi.addDep(spaceId, newTask.id, downEdge.target, 'blocks');
				}
			}

			await refreshGraph();
			activateNodeEditing(newTask.id);
		} catch (e) {
			connectionError = e instanceof Error ? e.message : 'Failed to create serial task';
			setTimeout(() => { connectionError = null; }, 4000);
		}
	}

	async function handleAddParallel(sourceNodeId: string): Promise<void> {
		if (!spaceId || !sourceNodeId) return;
		// Find upstream blockers of source node (edges where sourceNode is target)
		const upstreamEdges = edges.filter((e) => e.target === sourceNodeId);
		try {
			const newTask = await taskApi.createTask(spaceId, {
				title: 'New Task',
				type: 'task',
				priority: 2,
				...(mode === 'recipe' && rootTaskId ? { parentId: rootTaskId } : {}),
			});

			// Give newTask the same upstream deps as sourceNode
			for (const upEdge of upstreamEdges) {
				await taskApi.addDep(spaceId, upEdge.source, newTask.id, 'blocks');
			}

			await refreshGraph();
			activateNodeEditing(newTask.id);
		} catch (e) {
			connectionError = e instanceof Error ? e.message : 'Failed to create parallel task';
			setTimeout(() => { connectionError = null; }, 4000);
		}
	}

	async function handleTitleCommit(taskId: string, title: string): Promise<void> {
		editingNodeId = null;
		// Clear editing state on the node immediately
		nodes = nodes.map((n) =>
			n.id === taskId
				? { ...n, data: { ...n.data, editing: false, onTitleCommit: undefined } }
				: n,
		);
		if (!title || title === 'New Task') return;
		try {
			await taskApi.updateTask(spaceId, taskId, { title });
			// Update the node title locally
			nodes = nodes.map((n) =>
				n.id === taskId ? { ...n, data: { ...n.data, title } } : n,
			);
		} catch (e) {
			connectionError = e instanceof Error ? e.message : 'Failed to update task title';
			setTimeout(() => { connectionError = null; }, 4000);
		}
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

	// Reference to the flow container div, used to restore focus on Escape.
	let flowContainer: HTMLDivElement | undefined = $state(undefined);

	/** Programmatically select a node and notify the detail panel. */
	function selectNode(nodeId: string): void {
		nodes = nodes.map((n) => ({ ...n, selected: n.id === nodeId }));
		onselect?.(nodeId);
	}

	/**
	 * Vim-style graph navigation.
	 * - upstream / downstream: walk along dependency edges (k=upstream, j=downstream).
	 * - prev-sibling / next-sibling: move among nodes that share the same upstream
	 *   parent (h=prev, l=next), sorted by visual position.
	 */
	function navigateVim(dir: 'upstream' | 'downstream' | 'prev-sibling' | 'next-sibling'): void {
		if (nodes.length === 0) return;

		// No selection → select the first visible node.
		if (!selectedNodeId) {
			const sorted = [...nodes].sort(
				(a, b) => a.position.x - b.position.x || a.position.y - b.position.y,
			);
			selectNode(sorted[0].id);
			return;
		}

		// Helper: sort sibling/fork candidates by visual position.
		// In LR/RL layouts siblings are stacked vertically; in TB/BT they're horizontal.
		function sortByPosition(ns: Node[]): Node[] {
			if (currentDirection === 'LR' || currentDirection === 'RL') {
				return [...ns].sort((a, b) => a.position.y - b.position.y || a.position.x - b.position.x);
			}
			return [...ns].sort((a, b) => a.position.x - b.position.x || a.position.y - b.position.y);
		}

		switch (dir) {
			case 'upstream': {
				// k — go to upstream node (edge where selected is target)
				const upEdges = edges.filter((e) => e.target === selectedNodeId);
				if (upEdges.length === 0) return; // at root, do nothing
				const candidates = upEdges
					.map((e) => nodes.find((n) => n.id === e.source))
					.filter(Boolean) as Node[];
				if (candidates.length === 0) return;
				const sorted = sortByPosition(candidates);
				selectNode(sorted[0].id);
				break;
			}
			case 'downstream': {
				// j — go to downstream node (edge where selected is source), prefer left branch
				const downEdges = edges.filter((e) => e.source === selectedNodeId);
				if (downEdges.length === 0) return; // at leaf, do nothing
				const candidates = downEdges
					.map((e) => nodes.find((n) => n.id === e.target))
					.filter(Boolean) as Node[];
				if (candidates.length === 0) return;
				// "prefer left branch at forks" = sort by X asc (leftmost first)
				const sorted = [...candidates].sort(
					(a, b) => a.position.x - b.position.x || a.position.y - b.position.y,
				);
				selectNode(sorted[0].id);
				break;
			}
			case 'prev-sibling':
			case 'next-sibling': {
				// h/l — move among nodes that share the same upstream parent
				const upEdges = edges.filter((e) => e.target === selectedNodeId);
				let siblings: Node[];

				if (upEdges.length > 0) {
					// All nodes downstream of the same parent
					const parentId = upEdges[0].source;
					const sibEdges = edges.filter((e) => e.source === parentId);
					siblings = sibEdges
						.map((e) => nodes.find((n) => n.id === e.target))
						.filter(Boolean) as Node[];
				} else {
					// Root nodes: all nodes with no incoming edges
					const targetIds = new Set(edges.map((e) => e.target));
					siblings = nodes.filter((n) => !targetIds.has(n.id));
				}

				if (siblings.length <= 1) return;
				const sorted = sortByPosition(siblings);
				const idx = sorted.findIndex((n) => n.id === selectedNodeId);
				if (idx === -1) return;

				const nextIdx =
					dir === 'prev-sibling'
						? (idx - 1 + sorted.length) % sorted.length
						: (idx + 1) % sorted.length;
				selectNode(sorted[nextIdx].id);
				break;
			}
		}
	}

	function handleKeydown(e: KeyboardEvent): void {
		if (isEditableTarget(e)) return;

		// Alt+S: Add serial node on selected node
		if ((e.altKey || e.metaKey) && e.key === 's') {
			if (selectedNodeId) {
				e.preventDefault();
				handleAddSerial(selectedNodeId);
			}
			return;
		}

		// Alt+P: Add parallel node on selected node
		if ((e.altKey || e.metaKey) && e.key === 'p') {
			if (selectedNodeId) {
				e.preventDefault();
				handleAddParallel(selectedNodeId);
			}
			return;
		}

		// Alt+L: Re-run dagre layout
		if ((e.altKey || e.metaKey) && e.key === 'l') {
			e.preventDefault();
			handleRelayout();
			return;
		}

		// Vim-style navigation — skip if any modifier is held (avoid clashing with Alt+L etc.)
		if (!e.altKey && !e.metaKey && !e.ctrlKey) {
			switch (e.key) {
				case 'h':
					e.preventDefault();
					navigateVim('prev-sibling');
					return;
				case 'j':
					e.preventDefault();
					navigateVim('downstream');
					return;
				case 'k':
					e.preventDefault();
					navigateVim('upstream');
					return;
				case 'l':
					e.preventDefault();
					navigateVim('next-sibling');
					return;
			}
		}

		switch (e.key) {
			case 'Enter': {
				if (selectedNodeId) {
					e.preventDefault();
					// Open / focus the detail panel for the selected node.
					onselect?.(selectedNodeId);
				}
				break;
			}
			case 'Escape': {
				// Return focus to the graph canvas.
				e.preventDefault();
				flowContainer?.focus();
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
			<h1 class="text-lg font-semibold text-foreground">{mode === 'recipe' ? 'Recipe Steps' : (focusTaskId ? 'Neighborhood' : 'Graph')}</h1>
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
			<Button variant="outline" size="sm" onclick={handleAddUnconnected} title="Add unconnected node">
				<Plus class="size-4" />
				<span class="ml-1.5">Add Node</span>
			</Button>
			<Button variant="outline" size="sm" onclick={handleRelayout} title="Re-run dagre layout (Alt+L)" disabled={taskCount === 0}>
				<LayoutDashboard class="size-4" />
				<span class="ml-1.5">Re-layout</span>
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

	<!-- Delete confirmation dialog -->
	{#if deleteConfirmState}
		{@const dcs = deleteConfirmState}
		<Dialog.Root
			open={true}
			onOpenChange={(open) => {
				if (!open) {
					dcs.resolve(false);
					deleteConfirmState = null;
				}
			}}
		>
			<Dialog.Content>
				<Dialog.Header>
					<Dialog.Title>Delete task?</Dialog.Title>
					<Dialog.Description>
						Delete "<strong>{dcs.nodeTitle}</strong>"?
						{#if dcs.hasConnections}
							Upstream and downstream dependencies will be reconnected automatically.
						{/if}
						This action cannot be undone.
					</Dialog.Description>
				</Dialog.Header>
				<Dialog.Footer>
					<Button
						variant="outline"
						onclick={() => { dcs.resolve(false); deleteConfirmState = null; }}
					>
						Cancel
					</Button>
					<Button
						variant="destructive"
						onclick={() => { dcs.resolve(true); deleteConfirmState = null; }}
					>
						<Trash2 class="size-4 mr-1.5" />
						Delete
					</Button>
				</Dialog.Footer>
			</Dialog.Content>
		</Dialog.Root>
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
		<div bind:this={flowContainer} class="flex-1 min-h-0" tabindex="-1">
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
