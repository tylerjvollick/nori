<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import type { TaskTreeResponse, TaskDepsResponse } from '$lib/api/task';
	import type { TaskResponse } from '$lib/types/task';
	import type { CompleteTaskResponse } from '$lib/types/task';
	import { spaceMembersStore } from '$lib/stores/spaceMembers';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Separator } from '$lib/components/ui/separator';
	import { Progress } from '$lib/components/ui/progress';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import * as Avatar from '$lib/components/ui/avatar';
	import {
		Circle,
		CircleDot,
		CircleCheck,
		CirclePause,
		CircleX,
		CircleMinus,
		Clock,
		User,
		Calendar,
		Link as LinkIcon,
		ArrowRight,
		ArrowLeft,
		ChevronRight,
		BookOpen,
		Briefcase,
		ListTodo,
	} from '@lucide/svelte';
	import { formatDuration } from '$lib/utils/time';
	import TaskActions from './TaskActions.svelte';
	import SubTaskList from './SubTaskList.svelte';

	interface Props {
		task?: TaskTreeResponse;
		spaceId: string;
		stationMap?: Map<string, string>;
		deps?: TaskDepsResponse | null;
		/** Called after a successful task action with the updated task */
		onaction?: (updated: TaskResponse) => void;
		/** Called after completion with navigation info (next task ID). */
		oncomplete?: (response: CompleteTaskResponse) => void;
		/** When true, show skeleton placeholders instead of task data */
		isLoading?: boolean;
		/**
		 * 'recipe': hides status action buttons and actual time fields.
		 * 'task' (default): full detail view.
		 */
		mode?: 'task' | 'recipe';
		/** Map of task ID → title, used to display dependency titles instead of IDs. */
		taskTitleMap?: Map<string, string>;
		/** Name of the parent container (recipe or job name) for breadcrumb. */
		parentName?: string;
		/** Type of the parent container for breadcrumb icon. */
		parentType?: 'recipe' | 'job';
		/** Called when the user clicks the parent breadcrumb to navigate back. */
		onnavparent?: () => void;
		/** Called when a dependency badge is clicked — selects that task in the graph. */
		onselecttask?: (taskId: string) => void;
		/** Whether to hide the sub-tasks section (e.g. for recipe/job root tasks). */
		hideSubTasks?: boolean;
		/** Recipe version status (e.g. 'published', 'draft') — shown on root task only. */
		recipeStatus?: string;
		/** Recipe version number — shown on root task only. */
		recipeVersion?: number;
	}

	let {
		task, spaceId, stationMap = new Map(), deps = null,
		onaction, oncomplete, isLoading = false, mode = 'task',
		taskTitleMap = new Map(),
		parentName, parentType, onnavparent, onselecttask,
		hideSubTasks = false,
		recipeStatus, recipeVersion,
	}: Props = $props();

	let slug = $derived($page.params.slug);

	// --- Helpers ---

	type StatusConfig = {
		label: string;
		colorClass: string;
		bgClass: string;
	};

	function getStatusConfig(status: string): StatusConfig {
		switch (status) {
			case 'active':
				return { label: 'Active', colorClass: 'text-blue-500', bgClass: 'bg-blue-500/10' };
			case 'done':
				return { label: 'Done', colorClass: 'text-green-500', bgClass: 'bg-green-500/10' };
			case 'paused':
				return { label: 'Paused', colorClass: 'text-yellow-500', bgClass: 'bg-yellow-500/10' };
			case 'skipped':
				return { label: 'Skipped', colorClass: 'text-muted-foreground', bgClass: 'bg-muted' };
			case 'cancelled':
				return { label: 'Cancelled', colorClass: 'text-red-500', bgClass: 'bg-red-500/10' };
			default:
				return { label: 'Open', colorClass: 'text-muted-foreground', bgClass: 'bg-muted' };
		}
	}

	function priorityLabel(priority: number): string {
		switch (priority) {
			case 0: return 'Critical';
			case 1: return 'High';
			case 2: return 'Medium';
			case 3: return 'Low';
			default: return `P${priority}`;
		}
	}

	function priorityBadgeVariant(priority: number): 'default' | 'secondary' | 'destructive' | 'outline' {
		switch (priority) {
			case 0: return 'destructive';
			case 1: return 'default';
			case 2: return 'secondary';
			default: return 'outline';
		}
	}

	function typeLabel(type: string): string {
		switch (type) {
			case 'job': return 'Job';
			case 'task': return 'Task';
			case 'milestone': return 'Milestone';
			case 'gate': return 'Gate';
			default: return type;
		}
	}

	function getStationName(stationId: string | null | undefined): string | null {
		if (!stationId) return null;
		return stationMap.get(stationId) ?? stationId.slice(0, 8);
	}

	function formatDate(dateStr: string | null | undefined): string {
		if (!dateStr) return '--';
		return new Date(dateStr).toLocaleDateString(undefined, {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
		});
	}

	function formatDateTime(dateStr: string | null | undefined): string {
		if (!dateStr) return '--';
		return new Date(dateStr).toLocaleString(undefined, {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
		});
	}

	/** Compute completion ratio from children. */
	function computeProgress(node: TaskTreeResponse): { done: number; total: number } {
		if (!node.children || node.children.length === 0) {
			return { done: 0, total: 0 };
		}
		let done = 0;
		for (const child of node.children) {
			if (child.status === 'done' || child.status === 'skipped') done++;
		}
		return { done, total: node.children.length };
	}

	// --- Space members for assignee display ---

	let members = $derived($spaceMembersStore.members);

	/** Get initials for a user. Falls back to first char of email. */
	function getInitials(user: { firstName?: string; lastName?: string; email: string }): string {
		if (user.firstName && user.lastName) {
			return `${user.firstName[0]}${user.lastName[0]}`.toUpperCase();
		}
		if (user.firstName) {
			return user.firstName[0].toUpperCase();
		}
		return user.email[0].toUpperCase();
	}

	/** Get display name for a user. */
	function getDisplayName(user: { firstName?: string; lastName?: string; email: string }): string {
		if (user.firstName && user.lastName) {
			return `${user.firstName} ${user.lastName}`;
		}
		if (user.firstName) {
			return user.firstName;
		}
		return user.email;
	}

	/** Find the assigned member from the members list. */
	let assignedMember = $derived(
		task?.assignedToId
			? members.find((m) => m.userId === task.assignedToId)
			: null,
	);

	let assignedInitials = $derived(
		assignedMember ? getInitials(assignedMember.user) : null,
	);

	let assignedDisplayName = $derived(
		assignedMember ? getDisplayName(assignedMember.user) : null,
	);

	// --- Forward navigation ---

	let nextTaskId = $derived.by((): string | null => {
		if (!deps || deps.dependents.length === 0) return null;
		return deps.dependents[0].fromTaskId;
	});

	function navigateToNextTask(): void {
		if (nextTaskId) {
			goto(`/spaces/${slug}/${nextTaskId}`);
		}
	}

	/** Handle dependency badge click — select in graph if handler provided, else navigate. */
	function handleDepClick(e: MouseEvent, depTaskId: string): void {
		if (onselecttask) {
			e.preventDefault();
			onselecttask(depTaskId);
		}
	}

	// --- Derived state (safe when task is undefined during loading) ---

	let progress = $derived(task ? computeProgress(task) : { done: 0, total: 0 });
	let statusCfg = $derived(task ? getStatusConfig(task.status) : getStatusConfig('open'));
	let station = $derived(task ? getStationName(task.stationId) : null);
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

<div class="p-6 space-y-5">
	{#if isLoading || !task}
		<!-- Skeleton loading state -->
		<div>
			<div class="flex items-center gap-2 mb-2">
				<Skeleton class="h-5 w-20 rounded" />
				<Skeleton class="h-5 w-14 rounded" />
			</div>
			<Skeleton class="h-6 w-3/4 rounded" />
			<Skeleton class="h-4 w-1/2 mt-1 rounded" />
		</div>

		<Skeleton class="h-8 w-full rounded" />

		<Separator />

		<div class="space-y-3">
			{#each Array(4) as _}
				<div class="flex items-center justify-between">
					<Skeleton class="h-4 w-20 rounded" />
					<Skeleton class="h-5 w-16 rounded" />
				</div>
			{/each}
		</div>

		<Separator />

		<div class="space-y-3">
			<Skeleton class="h-4 w-12 rounded" />
			{#each Array(2) as _}
				<div class="flex items-center justify-between">
					<Skeleton class="h-4 w-24 rounded" />
					<Skeleton class="h-4 w-16 rounded" />
				</div>
			{/each}
		</div>

		<Separator />

		<div class="space-y-2">
			<div class="flex items-center justify-between">
				<Skeleton class="h-3 w-16 rounded" />
				<Skeleton class="h-3 w-24 rounded" />
			</div>
			<div class="flex items-center justify-between">
				<Skeleton class="h-3 w-16 rounded" />
				<Skeleton class="h-3 w-24 rounded" />
			</div>
		</div>
	{:else}
		<!-- Breadcrumb: Parent (recipe/job) > Task -->
		{#if parentName && parentType}
			<nav class="flex items-center gap-1 text-xs text-muted-foreground">
				<button
					class="flex items-center gap-1 hover:text-foreground transition-colors truncate max-w-[45%]"
					onclick={() => onnavparent?.()}
				>
					{#if parentType === 'recipe'}
						<BookOpen class="size-3 shrink-0" />
					{:else}
						<Briefcase class="size-3 shrink-0" />
					{/if}
					<span class="truncate">{parentName}</span>
				</button>
				<ChevronRight class="size-3 shrink-0 text-muted-foreground/50" />
				<span class="flex items-center gap-1 text-foreground truncate">
					<ListTodo class="size-3 shrink-0" />
					<span class="truncate">{task.title}</span>
				</span>
			</nav>
		{/if}

		<!-- Header: title, type badge, recipe status/version -->
		<div>
			<div class="flex items-center gap-2 mb-2">
				<Badge variant="outline" class="text-xs">
					{typeLabel(task.type)}
				</Badge>
				{#if recipeStatus}
					<Badge
						variant={recipeStatus === 'published' ? 'secondary' : 'outline'}
						class="text-xs {recipeStatus === 'published'
							? 'bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 border-green-200 dark:border-green-800'
							: 'bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 border-yellow-200 dark:border-yellow-800'}"
						data-testid="recipe-status-tag"
					>
						{recipeStatus === 'published' ? 'Published' : 'Draft'}
					</Badge>
				{/if}
				{#if recipeVersion}
					<Badge variant="secondary" class="text-xs" data-testid="recipe-version-tag">v{recipeVersion}</Badge>
				{/if}
			</div>
			<h2 class="text-lg font-semibold text-foreground">{task.title}</h2>
			{#if task.description}
				<p class="text-sm text-muted-foreground mt-1">{task.description}</p>
			{/if}
		</div>

		<!-- Action buttons (hidden in recipe mode — no status transitions) -->
		{#if mode !== 'recipe'}
			<TaskActions {task} {spaceId} layout="bar" {onaction} {oncomplete} />

			<!-- Forward navigation: go to next task in dependency chain -->
			{#if nextTaskId}
				<Button
					variant="outline"
					size="sm"
					class="w-full gap-2 justify-between text-muted-foreground hover:text-foreground"
					onclick={navigateToNextTask}
				>
					<span class="text-sm">Next: <span class="font-medium">{taskTitleMap.get(nextTaskId) ?? 'Task'}</span></span>
					<ChevronRight class="size-4" />
				</Button>
			{/if}
		{/if}

		<Separator />

		<!-- Core metadata -->
		<div class="space-y-3">
			<!-- Status (hidden in recipe mode) -->
			{#if mode !== 'recipe'}
				<div class="flex items-center justify-between">
					<span class="text-sm text-muted-foreground">Status</span>
					<Badge class="{statusCfg.bgClass} {statusCfg.colorClass} border-transparent">
						{@render statusIcon(task.status, 'w-3 h-3 mr-1')}
						{statusCfg.label}
					</Badge>
				</div>
			{/if}

			<!-- Priority -->
			<div class="flex items-center justify-between">
				<span class="text-sm text-muted-foreground">Priority</span>
				<Badge variant={priorityBadgeVariant(task.priority)}>
					{priorityLabel(task.priority)}
				</Badge>
			</div>

			<!-- Station -->
			{#if station}
				<div class="flex items-center justify-between">
					<span class="text-sm text-muted-foreground">Station</span>
					<span class="text-sm text-foreground">{station}</span>
				</div>
			{/if}

			<!-- Assignee -->
			{#if task.assignedToId}
				<div class="flex items-center justify-between">
					<span class="text-sm text-muted-foreground">Assigned To</span>
					<span class="text-sm text-foreground flex items-center gap-2">
						{#if assignedMember}
							<Avatar.Root size="sm" class="size-5 bg-primary/10 text-primary">
								<Avatar.Fallback class="text-[10px] font-medium bg-primary/10 text-primary">
									{assignedInitials}
								</Avatar.Fallback>
							</Avatar.Root>
							<span>{assignedDisplayName}</span>
						{:else}
							<Avatar.Root size="sm" class="size-5 bg-muted text-muted-foreground">
								<Avatar.Fallback class="text-[10px] font-medium">
									<User class="size-3" />
								</Avatar.Fallback>
							</Avatar.Root>
							<span class="font-mono">{task.assignedToId.slice(0, 8)}</span>
						{/if}
					</span>
				</div>
			{/if}

			<!-- Quantity -->
			{#if task.quantity > 1}
				<div class="flex items-center justify-between">
					<span class="text-sm text-muted-foreground">Quantity</span>
					<span class="text-sm text-foreground">{task.quantity}</span>
				</div>
			{/if}

			<!-- Recipe link -->
			{#if task.recipeId}
				<div class="flex items-center justify-between">
					<span class="text-sm text-muted-foreground">Recipe</span>
					<span class="text-sm text-foreground font-mono flex items-center gap-1">
						<LinkIcon class="w-3 h-3" />
						{task.recipeId.slice(0, 8)}
					</span>
				</div>
			{/if}
		</div>

		<Separator />

		<!-- Time data -->
		<div class="space-y-3">
			<h4 class="text-sm font-medium text-foreground">Time</h4>

			{#if task.estimatedTimeSeconds}
				<div class="flex items-center justify-between">
					<span class="text-sm text-muted-foreground">Estimated</span>
					<span class="text-sm text-foreground flex items-center gap-1">
						<Clock class="w-3 h-3 text-muted-foreground" />
						{formatDuration(task.estimatedTimeSeconds)}
					</span>
				</div>
			{/if}

			{#if mode !== 'recipe'}
				<div class="flex items-center justify-between">
					<span class="text-sm text-muted-foreground">Actual Time</span>
					<span class="text-sm text-foreground flex items-center gap-1">
						<Clock class="w-3 h-3 text-muted-foreground" />
						{formatDuration(task.actualTimeSeconds)}
					</span>
				</div>

				{#if task.startedAt}
					<div class="flex items-center justify-between">
						<span class="text-sm text-muted-foreground">Started</span>
						<span class="text-sm text-foreground">{formatDateTime(task.startedAt)}</span>
					</div>
				{/if}

				{#if task.completedAt}
					<div class="flex items-center justify-between">
						<span class="text-sm text-muted-foreground">Completed</span>
						<span class="text-sm text-foreground">{formatDateTime(task.completedAt)}</span>
					</div>
				{/if}

				{#if task.dueDate}
					<div class="flex items-center justify-between">
						<span class="text-sm text-muted-foreground">Due Date</span>
						<span class="text-sm text-foreground flex items-center gap-1">
							<Calendar class="w-3 h-3 text-muted-foreground" />
							{formatDate(task.dueDate)}
						</span>
					</div>
				{/if}
			{/if}
		</div>

		<!-- Sub-tasks (hidden for recipe/job root tasks) -->
		{#if !hideSubTasks}
			<Separator />
			<SubTaskList spaceId={spaceId} taskId={task.id} />
		{/if}

		<!-- Children progress (child tasks in the graph) -->
		{#if progress.total > 0}
			<Separator />
			<div class="space-y-2">
				<h4 class="text-sm font-medium text-foreground">Child Tasks</h4>
				<div class="flex items-center justify-between text-sm text-muted-foreground">
					<span>{progress.done} of {progress.total} complete</span>
					<span>{Math.round((progress.done / progress.total) * 100)}%</span>
				</div>
				<Progress value={progress.done} max={progress.total} class="h-2 rounded-full [&>[data-slot=progress-indicator]]:bg-green-500" />
			</div>
		{/if}

		<!-- Dependencies -->
		{#if deps && (deps.blockers.length > 0 || deps.dependents.length > 0)}
			<Separator />
			<div class="space-y-3">
				<h4 class="text-sm font-medium text-foreground">Dependencies</h4>

				{#if deps.blockers.length > 0}
					<div class="space-y-1.5">
						<span class="text-xs text-muted-foreground uppercase tracking-wide">Blocked by</span>
						<div class="flex flex-wrap gap-1.5">
							{#each deps.blockers as dep (dep.id)}
								<button
									class="inline-flex"
									onclick={(e) => handleDepClick(e, dep.toTaskId)}
								>
									<Badge variant="outline" class="cursor-pointer hover:bg-accent transition-colors gap-1">
										<ArrowLeft class="w-3 h-3 text-red-400 shrink-0" />
										<span class="text-xs">{taskTitleMap.get(dep.toTaskId) ?? dep.toTaskId}</span>
									</Badge>
								</button>
							{/each}
						</div>
					</div>
				{/if}

				{#if deps.dependents.length > 0}
					<div class="space-y-1.5">
						<span class="text-xs text-muted-foreground uppercase tracking-wide">Blocks</span>
						<div class="flex flex-wrap gap-1.5">
							{#each deps.dependents as dep (dep.id)}
								<button
									class="inline-flex"
									onclick={(e) => handleDepClick(e, dep.fromTaskId)}
								>
									<Badge variant="outline" class="cursor-pointer hover:bg-accent transition-colors gap-1">
										<ArrowRight class="w-3 h-3 text-orange-400 shrink-0" />
										<span class="text-xs">{taskTitleMap.get(dep.fromTaskId) ?? dep.fromTaskId}</span>
									</Badge>
								</button>
							{/each}
						</div>
					</div>
				{/if}
			</div>
		{/if}

		<!-- Deviation notes -->
		{#if task.deviationNotes}
			<Separator />
			<div class="space-y-2">
				<h4 class="text-sm font-medium text-foreground">Deviation Notes</h4>
				<p class="text-sm text-muted-foreground whitespace-pre-wrap">{task.deviationNotes}</p>
			</div>
		{/if}

		<!-- Timestamps -->
		<Separator />
		<div class="space-y-2 text-xs text-muted-foreground">
			<div class="flex items-center justify-between">
				<span>Created</span>
				<span>{formatDateTime(task.createdAt)}</span>
			</div>
			<div class="flex items-center justify-between">
				<span>Updated</span>
				<span>{formatDateTime(task.updatedAt)}</span>
			</div>
		</div>
	{/if}
</div>
