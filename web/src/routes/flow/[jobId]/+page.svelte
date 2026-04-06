<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { apiClient } from '$lib/api/client';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Separator } from '$lib/components/ui/separator';
	import * as Sheet from '$lib/components/ui/sheet';
	import {
		ArrowLeft,
		AlertCircle,
		ChevronRight,
		ChevronDown,
		Circle,
		CircleDot,
		CircleCheck,
		CirclePause,
		CircleX,
		CircleMinus,
		Clock,
		User,
		Calendar,
		ArrowRight
	} from 'lucide-svelte';

	/** Matches the backend TaskResponse DTO. */
	interface TaskNode {
		id: string;
		parentId?: string | null;
		stationId?: string | null;
		assignedToId?: string | null;
		type: string;
		status: string;
		title: string;
		description?: string | null;
		priority: number;
		displayOrder: number;
		dueDate?: string | null;
		startedAt?: string | null;
		completedAt?: string | null;
		actualTimeSeconds: number;
		deviationNotes?: string | null;
		createdAt: string;
		updatedAt: string;
		// Client-side tree properties
		children?: TaskNode[];
		isExpanded?: boolean;
		isLoadingChildren?: boolean;
		childrenLoaded?: boolean;
		depth?: number;
	}

	interface TaskListResponse {
		items: TaskNode[];
		total: number;
		offset: number;
		limit: number;
	}

	interface StationInfo {
		id: string;
		name: string;
	}

	let jobId = $derived($page.params.jobId);

	let rootTask = $state<TaskNode | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let stationMap = $state<Map<string, string>>(new Map());

	// Detail panel state
	let selectedTask = $state<TaskNode | null>(null);
	let detailOpen = $state(false);

	// Track dependency arrows: maps "fromTaskId" → "toTaskId" for siblings
	// We'll display arrows for parent-child ordering based on displayOrder,
	// since the dependency API isn't exposed yet. In the future this can
	// use actual TaskDep data.

	async function fetchStations(): Promise<void> {
		try {
			const resp = await apiClient.get<{ items: StationInfo[] }>('/api/v1/stations');
			const map = new Map<string, string>();
			for (const s of resp.items) {
				map.set(s.id, s.name);
			}
			stationMap = map;
		} catch {
			// Stations endpoint may not exist yet — gracefully degrade.
		}
	}

	async function fetchTask(id: string): Promise<TaskNode> {
		return apiClient.get<TaskNode>(`/api/v1/tasks/${encodeURIComponent(id)}`);
	}

	async function fetchChildren(parentId: string): Promise<TaskNode[]> {
		const resp = await apiClient.get<TaskListResponse>(
			`/api/v1/tasks?parentId=${encodeURIComponent(parentId)}&limit=200`
		);
		return resp.items ?? [];
	}

	/**
	 * Recursively loads a task and all its descendants.
	 * Fetches children level-by-level to build the tree.
	 */
	async function loadTaskTree(taskId: string, depth: number = 0): Promise<TaskNode> {
		const task = depth === 0 ? await fetchTask(taskId) : null;
		const node: TaskNode = task ?? ({ id: taskId } as TaskNode);
		node.depth = depth;
		node.isExpanded = depth < 2; // Auto-expand first 2 levels
		node.childrenLoaded = false;

		const children = await fetchChildren(taskId);
		node.children = await Promise.all(
			children.map(async (child) => {
				const childNode: TaskNode = {
					...child,
					depth: depth + 1,
					isExpanded: depth + 1 < 2,
					childrenLoaded: false,
					children: []
				};
				// Recursively load grandchildren
				const grandchildren = await fetchChildren(child.id);
				if (grandchildren.length > 0) {
					childNode.children = grandchildren.map((gc) => ({
						...gc,
						depth: depth + 2,
						isExpanded: false,
						childrenLoaded: false,
						children: []
					}));
					childNode.childrenLoaded = true;
				} else {
					childNode.childrenLoaded = true;
				}
				return childNode;
			})
		);
		node.childrenLoaded = true;

		return node;
	}

	/**
	 * Lazy-load children for a node that hasn't been expanded yet.
	 */
	async function expandNode(node: TaskNode): Promise<void> {
		if (node.childrenLoaded && node.children && node.children.length > 0) {
			// Children already loaded, check if grandchildren need loading
			for (const child of node.children) {
				if (!child.childrenLoaded) {
					child.isLoadingChildren = true;
					const grandchildren = await fetchChildren(child.id);
					child.children = grandchildren.map((gc) => ({
						...gc,
						depth: (child.depth ?? 0) + 1,
						isExpanded: false,
						childrenLoaded: false,
						children: []
					}));
					child.childrenLoaded = true;
					child.isLoadingChildren = false;
				}
			}
			return;
		}

		node.isLoadingChildren = true;
		const children = await fetchChildren(node.id);
		node.children = children.map((child) => ({
			...child,
			depth: (node.depth ?? 0) + 1,
			isExpanded: false,
			childrenLoaded: false,
			children: []
		}));
		node.childrenLoaded = true;
		node.isLoadingChildren = false;

		// Trigger reactivity
		rootTask = rootTask;
	}

	function toggleNode(node: TaskNode): void {
		node.isExpanded = !node.isExpanded;
		if (node.isExpanded && !node.childrenLoaded) {
			expandNode(node);
		}
		// Trigger reactivity
		rootTask = rootTask;
	}

	function selectTask(task: TaskNode): void {
		selectedTask = task;
		detailOpen = true;
	}

	onMount(async () => {
		try {
			await fetchStations();
			rootTask = await loadTaskTree(jobId!);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load job';
		} finally {
			isLoading = false;
		}
	});

	// --- Helpers ---

	type StatusConfig = {
		label: string;
		colorClass: string;
		bgClass: string;
	};

	function getStatusConfig(status: string): StatusConfig {
		switch (status) {
			case 'active':
				return {
					label: 'Active',
					colorClass: 'text-blue-500',
					bgClass: 'bg-blue-500/10'
				};
			case 'done':
				return {
					label: 'Done',
					colorClass: 'text-green-500',
					bgClass: 'bg-green-500/10'
				};
			case 'paused':
				return {
					label: 'Paused',
					colorClass: 'text-yellow-500',
					bgClass: 'bg-yellow-500/10'
				};
			case 'skipped':
				return {
					label: 'Skipped',
					colorClass: 'text-muted-foreground',
					bgClass: 'bg-muted'
				};
			case 'cancelled':
				return {
					label: 'Cancelled',
					colorClass: 'text-red-500',
					bgClass: 'bg-red-500/10'
				};
			default:
				// 'open' or unknown
				return {
					label: 'Open',
					colorClass: 'text-muted-foreground',
					bgClass: 'bg-muted'
				};
		}
	}

	function priorityLabel(priority: number): string {
		switch (priority) {
			case 0:
				return 'Critical';
			case 1:
				return 'High';
			case 2:
				return 'Medium';
			case 3:
				return 'Low';
			default:
				return `P${priority}`;
		}
	}

	function priorityBadgeVariant(
		priority: number
	): 'default' | 'secondary' | 'destructive' | 'outline' {
		switch (priority) {
			case 0:
				return 'destructive';
			case 1:
				return 'default';
			case 2:
				return 'secondary';
			default:
				return 'outline';
		}
	}

	function getStationName(stationId: string | null | undefined): string | null {
		if (!stationId) return null;
		return stationMap.get(stationId) ?? stationId.slice(0, 8);
	}

	function formatDuration(seconds: number): string {
		if (seconds === 0) return '--';
		const h = Math.floor(seconds / 3600);
		const m = Math.floor((seconds % 3600) / 60);
		if (h > 0) return `${h}h ${m}m`;
		return `${m}m`;
	}

	function formatDate(dateStr: string | null | undefined): string {
		if (!dateStr) return '--';
		return new Date(dateStr).toLocaleDateString(undefined, {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function formatDateTime(dateStr: string | null | undefined): string {
		if (!dateStr) return '--';
		return new Date(dateStr).toLocaleString(undefined, {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	function typeLabel(type: string): string {
		switch (type) {
			case 'job':
				return 'Job';
			case 'task':
				return 'Task';
			case 'milestone':
				return 'Milestone';
			case 'gate':
				return 'Gate';
			default:
				return type;
		}
	}

	/** Compute job progress from root task's children. */
	function computeProgress(node: TaskNode): { done: number; active: number; total: number } {
		if (!node.children || node.children.length === 0) {
			return { done: 0, active: 0, total: 0 };
		}

		let done = 0;
		let active = 0;
		const total = node.children.length;

		for (const child of node.children) {
			if (child.status === 'done' || child.status === 'skipped') done++;
			else if (child.status === 'active') active++;
		}

		return { done, active, total };
	}
</script>

{#snippet statusIcon(status: string, sizeClass: string)}
	{@const cfg = getStatusConfig(status)}
	{#if status === 'active'}
		<CircleDot class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'done'}
		<CircleCheck class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'paused'}
		<CirclePause class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'cancelled'}
		<CircleX class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else if status === 'skipped'}
		<CircleMinus class="{sizeClass} {cfg.colorClass} shrink-0" />
	{:else}
		<Circle class="{sizeClass} {cfg.colorClass} shrink-0" />
	{/if}
{/snippet}

<svelte:head>
	<title>{rootTask?.title ?? 'Job'} - Nori</title>
</svelte:head>

<div class="flex-1 overflow-auto">
	<div class="max-w-5xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
		<!-- Back navigation -->
		<div class="mb-4">
			<a
				href="/flow"
				class="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
			>
				<ArrowLeft class="w-4 h-4" />
				Back to Ready Queue
			</a>
		</div>

		<!-- Loading state -->
		{#if isLoading}
			<div class="space-y-4">
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

		<!-- Error state -->
		{:else if error}
			<div class="border border-destructive/30 bg-destructive/5 rounded-lg p-8 text-center">
				<AlertCircle class="w-10 h-10 text-destructive mx-auto mb-3" />
				<h3 class="text-lg font-semibold text-foreground mb-1">Failed to load job</h3>
				<p class="text-sm text-muted-foreground mb-4">{error}</p>
				<Button variant="outline" size="sm" onclick={() => window.location.reload()}>
					Try Again
				</Button>
			</div>

		<!-- Job loaded -->
		{:else if rootTask}
			{@const progress = computeProgress(rootTask)}
			{@const statusCfg = getStatusConfig(rootTask.status)}

			<!-- Job header -->
			<div class="mb-6">
				<div class="flex items-start justify-between gap-4">
					<div class="min-w-0">
						<div class="flex items-center gap-2 mb-1">
							<Badge variant="outline" class="text-xs font-mono">
								{rootTask.id}
							</Badge>
							<Badge variant={priorityBadgeVariant(rootTask.priority)}>
								{priorityLabel(rootTask.priority)}
							</Badge>
						</div>
						<h1 class="text-2xl font-bold text-foreground">
							{rootTask.title}
						</h1>
						{#if rootTask.description}
							<p class="text-sm text-muted-foreground mt-1">{rootTask.description}</p>
						{/if}
					</div>
					<div class="flex items-center gap-2 shrink-0">
						<Badge class="{statusCfg.bgClass} {statusCfg.colorClass} border-transparent">
							{@render statusIcon(rootTask.status, 'w-3 h-3 mr-1')}
							{statusCfg.label}
						</Badge>
					</div>
				</div>

				<!-- Progress bar -->
				{#if progress.total > 0}
					<div class="mt-4">
						<div class="flex items-center justify-between text-xs text-muted-foreground mb-1.5">
							<span>
								{progress.done}/{progress.total} done{#if progress.active > 0}, {progress.active} active{/if}
							</span>
							<span>{Math.round((progress.done / progress.total) * 100)}%</span>
						</div>
						<div class="h-2 bg-muted rounded-full overflow-hidden">
							<div
								class="h-full bg-green-500 transition-all duration-300"
								style="width: {(progress.done / progress.total) * 100}%"
							></div>
						</div>
					</div>
				{/if}

				{#if rootTask.dueDate}
					<div class="flex items-center gap-1.5 text-sm text-muted-foreground mt-3">
						<Calendar class="w-4 h-4" />
						Due {formatDate(rootTask.dueDate)}
					</div>
				{/if}
			</div>

			<Separator class="mb-6" />

			<!-- Task tree -->
			<div class="space-y-0.5">
				{#if rootTask.children && rootTask.children.length > 0}
					{#each rootTask.children as child, i (child.id)}
						{@const childStatus = getStatusConfig(child.status)}
						{@const hasChildren = (child.children && child.children.length > 0) || !child.childrenLoaded}
						{@const station = getStationName(child.stationId)}

						<!-- Dependency arrow indicator between siblings -->
						{#if i > 0}
							<div class="flex items-center pl-6 py-0.5">
								<ArrowRight class="w-3 h-3 text-muted-foreground/50 rotate-90" />
							</div>
						{/if}

						<!-- Task row -->
						<div class="group">
							<div
								class="flex items-center gap-2 py-2 px-2 rounded-md hover:bg-accent/50 transition-colors cursor-pointer"
								onclick={() => selectTask(child)}
								onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') selectTask(child); }}
								role="button"
								tabindex="0"
							>
								<!-- Expand/collapse toggle -->
								{#if hasChildren}
									<button
										class="p-0.5 rounded hover:bg-accent transition-colors shrink-0"
										onclick={(e) => { e.stopPropagation(); toggleNode(child); }}
									>
										{#if child.isExpanded}
											<ChevronDown class="w-4 h-4 text-muted-foreground" />
										{:else}
											<ChevronRight class="w-4 h-4 text-muted-foreground" />
										{/if}
									</button>
								{:else}
									<div class="w-5 shrink-0"></div>
								{/if}

								<!-- Status icon -->
								{@render statusIcon(child.status, 'w-5 h-5')}

								<!-- Task info -->
								<div class="flex-1 min-w-0 flex items-center gap-2">
									<span class="text-sm font-medium text-foreground truncate">
										{child.title}
									</span>
									{#if child.type === 'gate'}
										<Badge variant="outline" class="text-xs">Gate</Badge>
									{:else if child.type === 'milestone'}
										<Badge variant="outline" class="text-xs">Milestone</Badge>
									{/if}
									{#if station}
										<Badge variant="secondary" class="text-xs">
											{station}
										</Badge>
									{/if}
								</div>

								<!-- Right side metadata -->
								<div class="flex items-center gap-3 shrink-0">
									{#if child.actualTimeSeconds > 0}
										<span class="text-xs text-muted-foreground flex items-center gap-1">
											<Clock class="w-3 h-3" />
											{formatDuration(child.actualTimeSeconds)}
										</span>
									{/if}
									{#if child.assignedToId}
										<span class="text-xs text-muted-foreground flex items-center gap-1">
											<User class="w-3 h-3" />
										</span>
									{/if}
									<span class="text-xs text-muted-foreground font-mono">
										{child.id}
									</span>
								</div>
							</div>

							<!-- Expanded children (nested) -->
							{#if child.isExpanded && child.children && child.children.length > 0}
								<div class="ml-7 border-l border-border/50 pl-3">
									{#each child.children as grandchild, gi (grandchild.id)}
										{@const gcStatus = getStatusConfig(grandchild.status)}
										{@const gcHasChildren = (grandchild.children && grandchild.children.length > 0) || !grandchild.childrenLoaded}
										{@const gcStation = getStationName(grandchild.stationId)}

										{#if gi > 0}
											<div class="flex items-center pl-6 py-0.5">
												<ArrowRight class="w-3 h-3 text-muted-foreground/30 rotate-90" />
											</div>
										{/if}

										<div
											class="flex items-center gap-2 py-1.5 px-2 rounded-md hover:bg-accent/50 transition-colors cursor-pointer"
											onclick={() => selectTask(grandchild)}
											onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') selectTask(grandchild); }}
											role="button"
											tabindex="0"
										>
											{#if gcHasChildren}
												<button
													class="p-0.5 rounded hover:bg-accent transition-colors shrink-0"
													onclick={(e) => { e.stopPropagation(); toggleNode(grandchild); }}
												>
													{#if grandchild.isExpanded}
														<ChevronDown class="w-3.5 h-3.5 text-muted-foreground" />
													{:else}
														<ChevronRight class="w-3.5 h-3.5 text-muted-foreground" />
													{/if}
												</button>
											{:else}
												<div class="w-4.5 shrink-0"></div>
											{/if}

											{@render statusIcon(grandchild.status, 'w-4 h-4')}

											<div class="flex-1 min-w-0 flex items-center gap-2">
												<span class="text-sm text-foreground truncate">
													{grandchild.title}
												</span>
												{#if grandchild.type === 'gate'}
													<Badge variant="outline" class="text-[10px] py-0">Gate</Badge>
												{/if}
												{#if gcStation}
													<Badge variant="secondary" class="text-[10px] py-0">
														{gcStation}
													</Badge>
												{/if}
											</div>

											<div class="flex items-center gap-2 shrink-0">
												{#if grandchild.actualTimeSeconds > 0}
													<span class="text-xs text-muted-foreground flex items-center gap-1">
														<Clock class="w-3 h-3" />
														{formatDuration(grandchild.actualTimeSeconds)}
													</span>
												{/if}
												<span class="text-[10px] text-muted-foreground font-mono">
													{grandchild.id}
												</span>
											</div>
										</div>

										<!-- Third-level children -->
										{#if grandchild.isExpanded && grandchild.children && grandchild.children.length > 0}
											<div class="ml-6 border-l border-border/30 pl-3">
												{#each grandchild.children as ggchild (ggchild.id)}
													{@const ggStatus = getStatusConfig(ggchild.status)}
													<div
														class="flex items-center gap-2 py-1 px-2 rounded-md hover:bg-accent/50 transition-colors cursor-pointer"
														onclick={() => selectTask(ggchild)}
														onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') selectTask(ggchild); }}
														role="button"
														tabindex="0"
													>
														{@render statusIcon(ggchild.status, 'w-3.5 h-3.5')}
														<span class="text-sm text-foreground truncate flex-1">
															{ggchild.title}
														</span>
														<span class="text-[10px] text-muted-foreground font-mono shrink-0">
															{ggchild.id}
														</span>
													</div>
												{/each}
											</div>
										{/if}
									{/each}
								</div>
							{/if}

							<!-- Loading indicator for lazy-loaded children -->
							{#if child.isExpanded && child.isLoadingChildren}
								<div class="ml-12 py-2">
									<Skeleton class="h-4 w-1/3" />
								</div>
							{/if}
						</div>
					{/each}
				{:else}
					<div class="border border-border rounded-lg p-8 text-center">
						<p class="text-sm text-muted-foreground">
							This job has no child tasks yet.
						</p>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</div>

<!-- Task detail side panel -->
<Sheet.Root bind:open={detailOpen}>
	<Sheet.Content side="right" class="w-full sm:max-w-md overflow-y-auto">
		{#if selectedTask}
			{@const statusCfg = getStatusConfig(selectedTask.status)}
			<Sheet.Header class="pb-4">
				<div class="flex items-center gap-2 mb-2">
					<Badge variant="outline" class="text-xs font-mono">
						{selectedTask.id}
					</Badge>
					<Badge variant="outline" class="text-xs">
						{typeLabel(selectedTask.type)}
					</Badge>
				</div>
				<Sheet.Title class="text-lg">
					{selectedTask.title}
				</Sheet.Title>
				{#if selectedTask.description}
					<Sheet.Description>
						{selectedTask.description}
					</Sheet.Description>
				{/if}
			</Sheet.Header>

			<Separator />

			<div class="py-4 space-y-4">
				<!-- Status -->
				<div class="flex items-center justify-between">
					<span class="text-sm text-muted-foreground">Status</span>
					<Badge class="{statusCfg.bgClass} {statusCfg.colorClass} border-transparent">
						{@render statusIcon(selectedTask.status, 'w-3 h-3 mr-1')}
						{statusCfg.label}
					</Badge>
				</div>

				<!-- Priority -->
				<div class="flex items-center justify-between">
					<span class="text-sm text-muted-foreground">Priority</span>
					<Badge variant={priorityBadgeVariant(selectedTask.priority)}>
						{priorityLabel(selectedTask.priority)}
					</Badge>
				</div>

				<!-- Station -->
				{#if getStationName(selectedTask.stationId)}
					<div class="flex items-center justify-between">
						<span class="text-sm text-muted-foreground">Station</span>
						<span class="text-sm text-foreground">{getStationName(selectedTask.stationId)}</span>
					</div>
				{/if}

				<!-- Assignee -->
				{#if selectedTask.assignedToId}
					<div class="flex items-center justify-between">
						<span class="text-sm text-muted-foreground">Assigned To</span>
						<span class="text-sm text-foreground font-mono">
							{selectedTask.assignedToId.slice(0, 8)}
						</span>
					</div>
				{/if}

				<Separator />

				<!-- Time data -->
				<div class="space-y-3">
					<h4 class="text-sm font-medium text-foreground">Time</h4>

					<div class="flex items-center justify-between">
						<span class="text-sm text-muted-foreground">Actual Time</span>
						<span class="text-sm text-foreground">
							{formatDuration(selectedTask.actualTimeSeconds)}
						</span>
					</div>

					{#if selectedTask.startedAt}
						<div class="flex items-center justify-between">
							<span class="text-sm text-muted-foreground">Started</span>
							<span class="text-sm text-foreground">
								{formatDateTime(selectedTask.startedAt)}
							</span>
						</div>
					{/if}

					{#if selectedTask.completedAt}
						<div class="flex items-center justify-between">
							<span class="text-sm text-muted-foreground">Completed</span>
							<span class="text-sm text-foreground">
								{formatDateTime(selectedTask.completedAt)}
							</span>
						</div>
					{/if}

					{#if selectedTask.dueDate}
						<div class="flex items-center justify-between">
							<span class="text-sm text-muted-foreground">Due Date</span>
							<span class="text-sm text-foreground">
								{formatDate(selectedTask.dueDate)}
							</span>
						</div>
					{/if}
				</div>

				<!-- Dates -->
				<Separator />

				<div class="space-y-3">
					<div class="flex items-center justify-between">
						<span class="text-sm text-muted-foreground">Created</span>
						<span class="text-sm text-foreground">
							{formatDateTime(selectedTask.createdAt)}
						</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-sm text-muted-foreground">Updated</span>
						<span class="text-sm text-foreground">
							{formatDateTime(selectedTask.updatedAt)}
						</span>
					</div>
				</div>

				<!-- Deviation notes -->
				{#if selectedTask.deviationNotes}
					<Separator />
					<div class="space-y-2">
						<h4 class="text-sm font-medium text-foreground">Deviation Notes</h4>
						<p class="text-sm text-muted-foreground whitespace-pre-wrap">
							{selectedTask.deviationNotes}
						</p>
					</div>
				{/if}

				<!-- Child tasks summary -->
				{#if selectedTask.children && selectedTask.children.length > 0}
					{@const childProgress = computeProgress(selectedTask)}
					<Separator />
					<div class="space-y-2">
						<h4 class="text-sm font-medium text-foreground">Sub-tasks</h4>
						<div class="text-sm text-muted-foreground">
							{childProgress.done}/{childProgress.total} done{#if childProgress.active > 0}, {childProgress.active} active{/if}
						</div>
						<div class="h-1.5 bg-muted rounded-full overflow-hidden">
							<div
								class="h-full bg-green-500 transition-all"
								style="width: {childProgress.total > 0 ? (childProgress.done / childProgress.total) * 100 : 0}%"
							></div>
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</Sheet.Content>
</Sheet.Root>
