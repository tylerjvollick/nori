<script lang="ts">
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { onMount, onDestroy } from "svelte";
  import { taskApi, type ListTasksParams, type TaskDepsResponse } from "$lib/api/task";
  import { stationApi } from "$lib/api/station";
  import type { TaskResponse, TaskStatus, TaskType } from "$lib/types/task";
  import type { StationResponse } from "$lib/types/station";
  import { Badge } from "$lib/components/ui/badge";
  import { Button } from "$lib/components/ui/button";
  import { Skeleton } from "$lib/components/ui/skeleton";
  import {
    ArrowUpDown,
    ArrowUp,
    ArrowDown,
    RefreshCw,
    CircleAlert,
    Inbox,
    ChevronRight,
    ChevronDown,
  } from "@lucide/svelte";
  import { isEditableTarget, showToast } from "$lib/utils/keyboard.svelte";
  import { formatDuration } from "$lib/utils/time";

  /** Optional pre-loaded tasks. When provided, the list uses these instead of fetching. */
  interface Props {
    spaceId: string;
    tasks?: TaskResponse[];
    deps?: Map<string, TaskDepsResponse>;
    stationMap?: Map<string, string>;
    /** When provided, clicking a row calls this instead of navigating. */
    onselect?: (taskId: string) => void;
  }

  let {
    spaceId,
    tasks: externalTasks,
    deps: externalDeps,
    stationMap: externalStationMap,
    onselect,
  }: Props = $props();

  /** Whether we're in scoped mode (tasks provided externally). */
  let isScoped = $derived(!!externalTasks);


  const POLL_INTERVAL_MS = 30_000;
  const PAGE_SIZE = 200; // Fetch more to build complete chains

  // ---- Column definitions ----

  type SortKey =
    | "id"
    | "title"
    | "status"
    | "type"
    | "station"
    | "assignee"
    | "priority"
    | "quantity"
    | "time"
    | "dueDate";
  type SortDir = "asc" | "desc";

  interface ColumnDef {
    key: SortKey;
    label: string;
    class?: string;
  }

  const COLUMNS: ColumnDef[] = [
    { key: "id", label: "ID", class: "w-[140px]" },
    { key: "title", label: "Title" },
    { key: "status", label: "Status", class: "w-[100px]" },
    { key: "type", label: "Type", class: "w-[90px]" },
    { key: "station", label: "Station", class: "w-[120px]" },
    { key: "assignee", label: "Assignee", class: "w-[120px]" },
    { key: "priority", label: "Priority", class: "w-[80px]" },
    { key: "quantity", label: "Qty", class: "w-[60px]" },
    { key: "time", label: "Time", class: "w-[100px]" },
    { key: "dueDate", label: "Due", class: "w-[100px]" },
  ];

  // ---- State ----

  let allTasks = $state<TaskResponse[]>([]);
  let totalCount = $state(0);
  let isLoading = $state(true);
  let isLoadingMore = $state(false);
  let error = $state<string | null>(null);
  let isRefreshing = $state(false);
  let lastRefreshed = $state<Date | null>(null);
  let pollTimer: ReturnType<typeof setInterval> | null = null;

  let _internalStationMap = $state<Map<string, string>>(new Map());
  let stationMap = $derived(externalStationMap ?? _internalStationMap);

  let sortKey = $state<SortKey>("priority");
  let sortDir = $state<SortDir>("asc");

  // ---- Dependency chain state ----

  /** Internal deps map — built from fetched data or external prop. */
  let _internalDepsMap = $state<Map<string, TaskDepsResponse>>(new Map());
  let depsMap = $derived(externalDeps ?? _internalDepsMap);
  let depsLoaded = $state(false);
  let isLoadingDeps = $state(false);

  /** Set of collapsed chain root IDs. */
  let collapsedChains = $state<Set<string>>(new Set());

  // ---- Derived filters from URL ----

  let stationFilter = $derived($page.url.searchParams.get("station") || "");
  let statusFilter = $derived($page.url.searchParams.get("status") || "");
  let priorityFilter = $derived($page.url.searchParams.get("priority") || "");

  // ---- Dependency chain tree building ----

  /** A row in the flattened tree display. */
  interface TreeRow {
    task: TaskResponse;
    depth: number;
    chainRootId: string;
    isChainRoot: boolean;
    hasChildren: boolean;
  }

  /**
   * Build dependency chains from tasks and their deps.
   * Returns a flat array of TreeRow entries in display order.
   *
   * Algorithm:
   * 1. Build adjacency: for each task, find which tasks it blocks (successors)
   *    and which tasks block it (predecessors), scoped to the current task set.
   * 2. Root tasks = those with no predecessors in the set.
   * 3. DFS from each root, building the tree. Tasks that converge (multiple
   *    predecessors) appear under their last-visited predecessor to avoid
   *    duplication.
   */
  function buildDependencyChains(
    tasks: TaskResponse[],
    deps: Map<string, TaskDepsResponse>,
  ): TreeRow[] {
    if (tasks.length === 0) return [];

    const taskSet = new Set(tasks.map((t) => t.id));
    const taskMap = new Map(tasks.map((t) => [t.id, t]));

    // Build adjacency lists scoped to visible tasks
    const successors = new Map<string, string[]>(); // task -> tasks it blocks (outgoing)
    const predecessors = new Map<string, string[]>(); // task -> tasks that block it (incoming)

    for (const id of taskSet) {
      successors.set(id, []);
      predecessors.set(id, []);
    }

    for (const [taskId, taskDeps] of deps) {
      if (!taskSet.has(taskId)) continue;

      // dependents: tasks that depend on taskId (downstream)
      // dep.fromTaskId = downstream task, dep.toTaskId = taskId (upstream/blocker)
      for (const dep of taskDeps.dependents) {
        if (taskSet.has(dep.fromTaskId) && dep.toTaskId === taskId) {
          const succs = successors.get(taskId);
          if (succs && !succs.includes(dep.fromTaskId)) {
            succs.push(dep.fromTaskId);
          }
          const preds = predecessors.get(dep.fromTaskId);
          if (preds && !preds.includes(taskId)) {
            preds.push(taskId);
          }
        }
      }

      // blockers: tasks that block taskId (upstream)
      // dep.fromTaskId = taskId (blocked), dep.toTaskId = blocker/upstream
      for (const dep of taskDeps.blockers) {
        if (taskSet.has(dep.toTaskId) && dep.fromTaskId === taskId) {
          const succs = successors.get(dep.toTaskId);
          if (succs && !succs.includes(taskId)) {
            succs.push(taskId);
          }
          const preds = predecessors.get(taskId);
          if (preds && !preds.includes(dep.toTaskId)) {
            preds.push(dep.toTaskId);
          }
        }
      }
    }

    // Find root tasks: no predecessors within the set
    const roots = tasks.filter((t) => {
      const preds = predecessors.get(t.id);
      return !preds || preds.length === 0;
    });

    // Sort roots by priority (ascending), then by ID for stability
    roots.sort((a, b) => {
      const cmp = a.priority - b.priority;
      if (cmp !== 0) return cmp;
      return a.id.localeCompare(b.id);
    });

    // DFS to build the flat row list
    const rows: TreeRow[] = [];
    const visited = new Set<string>();

    function dfs(taskId: string, depth: number, chainRootId: string): void {
      if (visited.has(taskId)) return;
      visited.add(taskId);

      const task = taskMap.get(taskId);
      if (!task) return;

      const succs = successors.get(taskId) ?? [];
      // Filter to unvisited successors
      const unvisitedSuccs = succs.filter((s) => !visited.has(s));

      rows.push({
        task,
        depth,
        chainRootId,
        isChainRoot: depth === 0,
        hasChildren: unvisitedSuccs.length > 0,
      });

      // Sort successors by priority then ID for consistent ordering
      const sortedSuccs = [...unvisitedSuccs].sort((a, b) => {
        const tA = taskMap.get(a);
        const tB = taskMap.get(b);
        if (!tA || !tB) return 0;
        const cmp = tA.priority - tB.priority;
        if (cmp !== 0) return cmp;
        return tA.id.localeCompare(tB.id);
      });

      for (const succId of sortedSuccs) {
        // Only visit if all predecessors have been visited (convergence point)
        const succPreds = predecessors.get(succId) ?? [];
        const allPredsVisited = succPreds.every((p) => visited.has(p));
        if (allPredsVisited) {
          dfs(succId, depth + 1, chainRootId);
        }
      }
    }

    // Walk each root's chain
    for (const root of roots) {
      dfs(root.id, 0, root.id);
    }

    // Any tasks not yet visited (e.g., in cycles or orphaned from convergence)
    // get added as standalone roots
    for (const task of tasks) {
      if (!visited.has(task.id)) {
        visited.add(task.id);
        rows.push({
          task,
          depth: 0,
          chainRootId: task.id,
          isChainRoot: true,
          hasChildren: false,
        });
      }
    }

    return rows;
  }

  // ---- Build tree rows from current data ----

  let treeRows = $derived.by((): TreeRow[] => {
    if (allTasks.length === 0) return [];

    // If we have no deps data, fall back to flat list (all depth 0)
    if (depsMap.size === 0) {
      return sortedTasksFlat.map((t) => ({
        task: t,
        depth: 0,
        chainRootId: t.id,
        isChainRoot: true,
        hasChildren: false,
      }));
    }

    return buildDependencyChains(allTasks, depsMap);
  });

  /** Visible rows after applying chain collapse. */
  let visibleRows = $derived.by((): TreeRow[] => {
    if (collapsedChains.size === 0) return treeRows;

    return treeRows.filter((row) => {
      // Always show chain roots
      if (row.isChainRoot) return true;
      // Hide children of collapsed chains
      return !collapsedChains.has(row.chainRootId);
    });
  });

  // ---- Sorting (flat fallback when no deps) ----

  let sortedTasksFlat = $derived.by(() => {
    const tasks = [...allTasks];
    const dir = sortDir === "asc" ? 1 : -1;

    tasks.sort((a, b) => {
      let cmp = 0;
      switch (sortKey) {
        case "id":
          cmp = a.id.localeCompare(b.id);
          break;
        case "title":
          cmp = a.title.localeCompare(b.title);
          break;
        case "status":
          cmp = a.status.localeCompare(b.status);
          break;
        case "type":
          cmp = a.type.localeCompare(b.type);
          break;
        case "station": {
          const sA = a.stationId
            ? (stationMap.get(a.stationId) ?? a.stationId)
            : "";
          const sB = b.stationId
            ? (stationMap.get(b.stationId) ?? b.stationId)
            : "";
          cmp = sA.localeCompare(sB);
          break;
        }
        case "assignee": {
          const aA = a.assignedToId ?? "";
          const aB = b.assignedToId ?? "";
          cmp = aA.localeCompare(aB);
          break;
        }
        case "priority":
          cmp = a.priority - b.priority;
          break;
        case "quantity":
          cmp = a.quantity - b.quantity;
          break;
        case "time":
          cmp = a.actualTimeSeconds - b.actualTimeSeconds;
          break;
        case "dueDate": {
          const dA = a.dueDate ? new Date(a.dueDate).getTime() : Infinity;
          const dB = b.dueDate ? new Date(b.dueDate).getTime() : Infinity;
          cmp = dA - dB;
          break;
        }
      }
      return cmp * dir;
    });

    return tasks;
  });

  let hasMore = $derived(allTasks.length < totalCount);

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

  function buildParams(offset: number): ListTasksParams {
    const params: ListTasksParams = {
      offset,
      limit: PAGE_SIZE,
    };
    if (stationFilter) params.stationId = stationFilter;
    if (statusFilter) params.status = statusFilter;
    if (priorityFilter) {
      // Priority filter: the API doesn't support priority filtering,
      // so we fetch all and filter client-side. But if a status is set,
      // we still pass it server-side.
    }
    return params;
  }

  async function fetchTasks(opts?: {
    silent?: boolean;
    append?: boolean;
  }): Promise<void> {
    if (!opts?.silent && !opts?.append) {
      isLoading = true;
    }
    if (opts?.append) {
      isLoadingMore = true;
    }
    isRefreshing = true;
    error = null;

    try {
      const offset = opts?.append ? allTasks.length : 0;
      const result = await taskApi.listTasks(spaceId, buildParams(offset));

      let items = result.items;

      // Exclude jobs — they have their own board mode
      items = items.filter((t) => t.type !== "job");

      // Apply client-side priority filter if set (API doesn't support it)
      if (priorityFilter) {
        const pNum = Number(priorityFilter);
        items = items.filter((t) => t.priority === pNum);
      }

      if (opts?.append) {
        allTasks = [...allTasks, ...items];
      } else {
        allTasks = items;
      }
      totalCount = result.total;
      lastRefreshed = new Date();

      // Fetch deps for chain building (non-blocking)
      if (!opts?.append) {
        fetchDepsForTasks(allTasks);
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load tasks";
    } finally {
      isLoading = false;
      isLoadingMore = false;
      isRefreshing = false;
    }
  }

  /** Fetch deps for all tasks in batches (for self-fetching mode). */
  async function fetchDepsForTasks(tasks: TaskResponse[]): Promise<void> {
    if (tasks.length === 0) return;
    isLoadingDeps = true;

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
            return {
              id: t.id,
              deps: { blockers: [], dependents: [] } as TaskDepsResponse,
            };
          }
        }),
      );
      for (const r of results) {
        map.set(r.id, r.deps);
      }
    }

    _internalDepsMap = map;
    depsLoaded = true;
    isLoadingDeps = false;
  }

  function loadMore(): void {
    fetchTasks({ append: true });
  }

  function handleManualRefresh(): void {
    fetchTasks({ silent: true });
  }

  function startPolling(): void {
    stopPolling();
    pollTimer = setInterval(
      () => fetchTasks({ silent: true }),
      POLL_INTERVAL_MS,
    );
  }

  function stopPolling(): void {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }

  // Re-fetch when filters change
  let prevStation = $state("");
  let prevStatus = $state("");
  let prevPriority = $state("");

  /** Load tasks from external data, applying filters client-side. */
  function loadFromExternalTasks(tasks: TaskResponse[]): void {
    let filtered = tasks;
    if (stationFilter) {
      filtered = filtered.filter((t) => t.stationId === stationFilter);
    }
    if (statusFilter) {
      filtered = filtered.filter((t) => t.status === statusFilter);
    }
    if (priorityFilter) {
      const pNum = Number(priorityFilter);
      filtered = filtered.filter((t) => t.priority === pNum);
    }
    allTasks = filtered;
    totalCount = filtered.length;
    isLoading = false;
    lastRefreshed = new Date();
  }

  // When external tasks change, reload
  $effect(() => {
    if (externalTasks) {
      loadFromExternalTasks(externalTasks);
    }
  });

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
          loadFromExternalTasks(externalTasks);
        } else {
          fetchTasks({ silent: true });
        }
      }
    }
  });

  onMount(async () => {
    if (isScoped) {
      // In scoped mode, tasks are provided externally. No fetching needed.
      return;
    }
    await Promise.all([fetchTasks(), fetchStations()]);
    startPolling();
  });

  onDestroy(() => {
    stopPolling();
  });

  // ---- Sort handling ----

  function handleSort(key: SortKey): void {
    if (sortKey === key) {
      sortDir = sortDir === "asc" ? "desc" : "asc";
    } else {
      sortKey = key;
      sortDir = "asc";
    }
  }

  // ---- Chain collapse ----

  function toggleChainCollapse(chainRootId: string): void {
    const next = new Set(collapsedChains);
    if (next.has(chainRootId)) {
      next.delete(chainRootId);
    } else {
      next.add(chainRootId);
    }
    collapsedChains = next;
  }

  // ---- Row click ----

  let slug = $derived($page.params.slug);

  function navigateToTask(taskId: string): void {
    if (onselect) {
      onselect(taskId);
    } else {
      goto(`/spaces/${slug}/${taskId}`);
    }
  }

  // ---- Keyboard selection ----

  let selectedRow = $state(-1);

  /** Get the currently selected row, or null. */
  let selectedTreeRow = $derived.by(() => {
    if (selectedRow < 0 || selectedRow >= visibleRows.length) return null;
    return visibleRows[selectedRow] ?? null;
  });

  /** Get the currently selected task, or null. */
  let selectedTask = $derived(selectedTreeRow?.task ?? null);

  function clearSelection(): void {
    selectedRow = -1;
  }

  function scrollSelectedRowIntoView(): void {
    requestAnimationFrame(() => {
      const el = document.querySelector(
        '[data-kb-selected="true"]',
      ) as HTMLElement;
      el?.scrollIntoView({ block: "nearest", behavior: "smooth" });
    });
  }

  function handleKeydown(e: KeyboardEvent): void {
    if (isEditableTarget(e)) return;

    switch (e.key) {
      case "j": {
        e.preventDefault();
        if (visibleRows.length === 0) break;
        if (selectedRow < 0) {
          selectedRow = 0;
        } else if (selectedRow < visibleRows.length - 1) {
          selectedRow++;
        }
        scrollSelectedRowIntoView();
        break;
      }
      case "k": {
        e.preventDefault();
        if (visibleRows.length === 0) break;
        if (selectedRow < 0) {
          selectedRow = 0;
        } else if (selectedRow > 0) {
          selectedRow--;
        }
        scrollSelectedRowIntoView();
        break;
      }
      case "Enter": {
        const task = selectedTask;
        if (task) {
          e.preventDefault();
          navigateToTask(task.id);
        }
        break;
      }
      case "c": {
        const task = selectedTask;
        if (task && task.status === "open") {
          e.preventDefault();
          startSelectedTask(task);
        }
        break;
      }
      case "d": {
        const task = selectedTask;
        if (task && task.status === "in_progress") {
          e.preventDefault();
          completeSelectedTask(task);
        }
        break;
      }
      case "h": {
        // Toggle collapse for the chain the selected row belongs to
        const row = selectedTreeRow;
        if (row) {
          e.preventDefault();
          toggleChainCollapse(row.chainRootId);
          // If we collapsed and the selected row is now hidden, move to the chain root
          if (collapsedChains.has(row.chainRootId) && !row.isChainRoot) {
            const rootIdx = visibleRows.findIndex(
              (r) => r.task.id === row.chainRootId,
            );
            if (rootIdx >= 0) {
              selectedRow = rootIdx;
            }
          }
        }
        break;
      }
      case "Escape": {
        if (selectedRow >= 0) {
          clearSelection();
        }
        break;
      }
    }
  }

  async function startSelectedTask(task: TaskResponse): Promise<void> {
    try {
      await taskApi.startTask(spaceId, task.id);
      showToast(`Started: ${task.title}`);
      fetchTasks({ silent: true });
    } catch (err) {
      console.error("Failed to start task:", err);
      showToast("Failed to start task");
    }
  }

  async function completeSelectedTask(task: TaskResponse): Promise<void> {
    try {
      await taskApi.completeTask(spaceId, task.id);
      showToast(`Completed: ${task.title}`);
      fetchTasks({ silent: true });
    } catch (err) {
      console.error("Failed to complete task:", err);
      showToast("Failed to complete task");
    }
  }

  // ---- Helpers ----

  const STATUS_COLORS: Record<string, string> = {
    open: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
    active:
      "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300",
    paused:
      "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300",
    done: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
    skipped: "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300",
    cancelled: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300",
  };

  const TYPE_COLORS: Record<string, string> = {
    job: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
    task: "bg-sky-100 text-sky-800 dark:bg-sky-900/30 dark:text-sky-300",
    milestone:
      "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300",
    gate: "bg-rose-100 text-rose-800 dark:bg-rose-900/30 dark:text-rose-300",
  };

  const PRIORITY_COLORS: Record<number, string> = {
    0: "text-red-600 dark:text-red-400 font-semibold",
    1: "text-orange-600 dark:text-orange-400 font-semibold",
    2: "text-blue-600 dark:text-blue-400",
    3: "text-gray-500 dark:text-gray-400",
    4: "text-gray-400 dark:text-gray-500",
  };

  function getStationName(stationId: string | null | undefined): string {
    if (!stationId) return "\u2014";
    return stationMap.get(stationId) ?? stationId.slice(0, 8);
  }

  function getAssigneeName(assigneeId: string | null | undefined): string {
    if (!assigneeId) return "\u2014";
    return assigneeId.slice(0, 8);
  }

  function formatTimeAgo(dateStr: string | null | undefined): string {
    if (!dateStr) return "\u2014";
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60_000);
    if (diffMins < 1) return "Just now";
    if (diffMins < 60) return `${diffMins}m ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 30) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  }

  function formatLastRefreshed(date: Date | null): string {
    if (!date) return "";
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }

  /** Count children in a chain (for the collapse indicator). */
  function chainChildCount(chainRootId: string): number {
    return treeRows.filter(
      (r) => r.chainRootId === chainRootId && !r.isChainRoot,
    ).length;
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="flex h-full flex-col overflow-hidden">
  <!-- Header -->
  <div class="flex-shrink-0 flex items-center justify-between px-4 py-2">
    <div class="flex items-center gap-3">
      <h1 class="text-lg font-semibold text-foreground">Tasks</h1>
      {#if !isLoading}
        <span class="text-xs text-muted-foreground">
          {allTasks.length} of {totalCount}
        </span>
      {/if}
      {#if isLoadingDeps}
        <span class="text-xs text-muted-foreground animate-pulse">
          Loading chains...
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
        onclick={handleManualRefresh}
        disabled={isRefreshing}
      >
        <RefreshCw class="size-4 {isRefreshing ? 'animate-spin' : ''}" />
        <span class="ml-1.5">Refresh</span>
      </Button>
    </div>
  </div>

  <!-- Error banner -->
  {#if error}
    <div
      class="mx-4 mb-2 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm"
    >
      <CircleAlert class="size-4 shrink-0 text-destructive" />
      <span class="text-destructive">{error}</span>
      <Button
        variant="outline"
        size="sm"
        class="ml-auto"
        onclick={() => fetchTasks()}
      >
        Retry
      </Button>
    </div>
  {/if}

  <!-- Table container -->
  <div class="flex-1 overflow-auto px-4 pb-4">
    {#if isLoading}
      <!-- Loading skeleton -->
      <div class="rounded-lg border">
        <div class="border-b bg-muted/50 px-4 py-3">
          <Skeleton class="h-4 w-full" />
        </div>
        {#each Array(10) as _}
          <div
            class="flex items-center gap-4 border-b px-4 py-3 last:border-b-0"
          >
            <Skeleton class="h-4 w-[120px]" />
            <Skeleton class="h-4 flex-1" />
            <Skeleton class="h-5 w-[70px] rounded-full" />
            <Skeleton class="h-5 w-[60px] rounded-full" />
            <Skeleton class="h-4 w-[100px]" />
            <Skeleton class="h-4 w-[100px]" />
            <Skeleton class="h-4 w-[50px]" />
            <Skeleton class="h-4 w-[40px]" />
            <Skeleton class="h-4 w-[80px]" />
            <Skeleton class="h-4 w-[80px]" />
          </div>
        {/each}
      </div>
    {:else if allTasks.length === 0}
      <!-- Empty state -->
      <div class="rounded-lg border border-border p-12 text-center">
        <Inbox class="mx-auto mb-4 size-12 text-muted-foreground" />
        <h3 class="mb-1 text-lg font-semibold text-foreground">
          No tasks found
        </h3>
        <p class="mx-auto max-w-md text-sm text-muted-foreground">
          {#if stationFilter || statusFilter || priorityFilter}
            No tasks match the current filters. Try adjusting or clearing them.
          {:else}
            There are no tasks yet. Create a job or task to get started.
          {/if}
        </p>
      </div>
    {:else}
      <!-- Table -->
      <div class="rounded-lg border">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b bg-muted/50">
              {#each COLUMNS as col (col.key)}
                <th
                  class="px-3 py-2 text-left font-medium text-muted-foreground {col.class ??
                    ''}"
                >
                  <button
                    class="inline-flex items-center gap-1 hover:text-foreground transition-colors"
                    onclick={() => handleSort(col.key)}
                  >
                    {col.label}
                    {#if sortKey === col.key}
                      {#if sortDir === "asc"}
                        <ArrowUp class="size-3" />
                      {:else}
                        <ArrowDown class="size-3" />
                      {/if}
                    {:else}
                      <ArrowUpDown class="size-3 opacity-30" />
                    {/if}
                  </button>
                </th>
              {/each}
            </tr>
          </thead>
          <tbody>
            {#each visibleRows as row, i (row.task.id)}
              {@const task = row.task}
              {@const isCollapsed = collapsedChains.has(row.chainRootId)}
              {@const childCount = row.isChainRoot
                ? chainChildCount(row.chainRootId)
                : 0}
              <tr
                class="border-b last:border-b-0 cursor-pointer transition-colors {selectedRow ===
                i
                  ? 'bg-primary/10 ring-1 ring-inset ring-primary/30'
                  : row.isChainRoot && row.hasChildren
                    ? 'bg-muted/30 hover:bg-accent/50'
                    : 'hover:bg-accent/50'}"
                onclick={() => navigateToTask(task.id)}
                role="link"
                tabindex="0"
                data-kb-selected={selectedRow === i}
                onkeydown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    navigateToTask(task.id);
                  }
                }}
              >
                <!-- ID -->
                <td
                  class="px-3 py-2.5 font-mono text-xs text-muted-foreground truncate max-w-[140px]"
                  title={task.id}
                >
                  {task.id}
                </td>

                <!-- Title (with tree indentation) -->
                <td
                  class="px-3 py-2.5 truncate max-w-[300px]"
                  title={task.title}
                >
                  <div class="flex items-center gap-1">
                    {#if row.depth > 0}
                      <!-- Tree indent -->
                      <span
                        class="inline-flex shrink-0"
                        style="width: {row.depth * 20}px"
                      >
                        {#each Array(row.depth - 1) as _}
                          <span
                            class="inline-block w-5 border-l border-border"
                          ></span>
                        {/each}
                        <span
                          class="inline-flex w-5 items-center text-muted-foreground/50"
                          >&#x2514;</span
                        >
                      </span>
                    {/if}
                    {#if row.isChainRoot && row.hasChildren}
                      <!-- Collapse toggle for chain roots -->
                      <button
                        class="shrink-0 p-0.5 -ml-0.5 rounded hover:bg-accent text-muted-foreground"
                        onclick={(e) => {
                          e.stopPropagation();
                          toggleChainCollapse(row.chainRootId);
                        }}
                        title={isCollapsed
                          ? `Expand chain (${childCount} tasks)`
                          : `Collapse chain (${childCount} tasks)`}
                      >
                        {#if isCollapsed}
                          <ChevronRight class="size-3.5" />
                        {:else}
                          <ChevronDown class="size-3.5" />
                        {/if}
                      </button>
                    {/if}
                    <span class="truncate">{task.title}</span>
                    {#if row.isChainRoot && row.hasChildren && isCollapsed}
                      <span
                        class="ml-1 text-xs text-muted-foreground/60 shrink-0"
                      >
                        +{childCount}
                      </span>
                    {/if}
                  </div>
                </td>

                <!-- Status -->
                <td class="px-3 py-2.5">
                  <span
                    class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {STATUS_COLORS[
                      task.status
                    ] ?? ''}"
                  >
                    {task.status}
                  </span>
                </td>

                <!-- Type -->
                <td class="px-3 py-2.5">
                  <span
                    class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {TYPE_COLORS[
                      task.type
                    ] ?? ''}"
                  >
                    {task.type}
                  </span>
                </td>

                <!-- Station -->
                <td
                  class="px-3 py-2.5 text-muted-foreground truncate max-w-[120px]"
                >
                  {getStationName(task.stationId)}
                </td>

                <!-- Assignee -->
                <td
                  class="px-3 py-2.5 text-muted-foreground truncate max-w-[120px]"
                >
                  {getAssigneeName(task.assignedToId)}
                </td>

                <!-- Priority -->
                <td class="px-3 py-2.5 {PRIORITY_COLORS[task.priority] ?? ''}">
                  P{task.priority}
                </td>

                <!-- Quantity -->
                <td class="px-3 py-2.5 text-muted-foreground">
                  {#if task.quantity > 1}
                    {task.quantity}
                  {/if}
                </td>

                <!-- Time Spent -->
                <td
                  class="px-3 py-2.5 text-xs text-muted-foreground {task.status ===
                  'in_progress'
                    ? 'text-blue-500 dark:text-blue-400'
                    : ''}"
                >
                  <span class="inline-flex items-center gap-1">
                    {#if task.status === 'in_progress'}
                      <span class="relative flex size-2 shrink-0">
                        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                        <span class="relative inline-flex rounded-full size-2 bg-green-500"></span>
                      </span>
                    {/if}
                    {formatDuration(task.actualTimeSeconds)}
                  </span>
                </td>

                <!-- Due Date -->
                <td
                  class="px-3 py-2.5 text-xs text-muted-foreground"
                  title={task.dueDate
                    ? new Date(task.dueDate).toLocaleString()
                    : ""}
                >
                  {formatTimeAgo(task.dueDate)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <!-- Load more -->
      {#if hasMore}
        <div class="mt-4 flex justify-center">
          <Button
            variant="outline"
            size="sm"
            onclick={loadMore}
            disabled={isLoadingMore}
          >
            {#if isLoadingMore}
              <RefreshCw class="size-4 animate-spin mr-1.5" />
              Loading...
            {:else}
              Load more ({totalCount - allTasks.length} remaining)
            {/if}
          </Button>
        </div>
      {/if}
    {/if}
  </div>
</div>
