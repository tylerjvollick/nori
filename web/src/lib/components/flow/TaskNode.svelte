<script lang="ts">
	import { Handle, Position } from '@xyflow/svelte';
	import type { TaskStatus, TaskType } from '$lib/types/task';
	import type { GraphDirection } from '$lib/stores/graph';
	import {
		Circle,
		CircleDot,
		CircleCheck,
		CirclePause,
		CircleMinus,
		CircleX,
		Ban,
		Plus,
	} from '@lucide/svelte';

	// Props from xyflow NodeProps — we receive these automatically
	let {
		data,
		selected,
	}: {
		data: {
			title: string;
			taskId: string;
			status: TaskStatus;
			type: TaskType;
			priority: number;
			stationName?: string;
			isFocus?: boolean;
			isBlocked?: boolean;
			direction?: GraphDirection;
			mode?: 'task' | 'recipe';
			estimatedTimeSeconds?: number | null;
			/** When true, shows an inline title input focused for editing. */
			editing?: boolean;
			/** Called when the inline title input is committed (Enter or blur). */
			onTitleCommit?: (title: string) => void;
			/** When set, shows a [+] button below the node for serial node insertion. */
			onAddSerial?: () => void;
		};
		selected: boolean;
		[key: string]: unknown;
	} = $props();

	let editTitle = $state(data.title || '');
	let inputEl: HTMLInputElement | undefined = $state(undefined);

	$effect(() => {
		if (data.editing && inputEl) {
			inputEl.focus();
			inputEl.select();
		}
	});

	// Keep editTitle in sync if title changes externally (e.g. on task refresh)
	$effect(() => {
		if (!data.editing) {
			editTitle = data.title;
		}
	});

	function commitTitle(): void {
		const trimmed = editTitle.trim() || 'New Task';
		data.onTitleCommit?.(trimmed);
	}

	function handleInputKeydown(e: KeyboardEvent): void {
		if (e.key === 'Enter') {
			e.preventDefault();
			commitTitle();
		} else if (e.key === 'Escape') {
			e.preventDefault();
			// Revert to original title
			editTitle = data.title || '';
			data.onTitleCommit?.(data.title || 'New Task');
		}
		// Prevent xyflow from processing keydown (delete, etc.) while editing
		e.stopPropagation();
	}

	// Handle positions based on graph layout direction
	// Target = incoming (predecessors/blockers), Source = outgoing (dependents/successors)
	const TARGET_POSITIONS: Record<GraphDirection, Position> = {
		LR: Position.Left,
		RL: Position.Right,
		TB: Position.Top,
		BT: Position.Bottom,
	};
	const SOURCE_POSITIONS: Record<GraphDirection, Position> = {
		LR: Position.Right,
		RL: Position.Left,
		TB: Position.Bottom,
		BT: Position.Top,
	};

	let targetPosition = $derived(TARGET_POSITIONS[data.direction ?? 'LR']);
	let sourcePosition = $derived(SOURCE_POSITIONS[data.direction ?? 'LR']);

	// Status → color mapping (matches TaskCard/TaskTree patterns)
	const STATUS_CONFIG: Record<
		TaskStatus,
		{ bg: string; border: string; text: string; icon: typeof Circle }
	> = {
		open: {
			bg: 'bg-gray-50 dark:bg-gray-900/50',
			border: 'border-gray-300 dark:border-gray-700',
			text: 'text-gray-500',
			icon: Circle,
		},
		active: {
			bg: 'bg-blue-50 dark:bg-blue-950/50',
			border: 'border-blue-400 dark:border-blue-600',
			text: 'text-blue-500',
			icon: CircleDot,
		},
		paused: {
			bg: 'bg-yellow-50 dark:bg-yellow-950/50',
			border: 'border-yellow-400 dark:border-yellow-600',
			text: 'text-yellow-500',
			icon: CirclePause,
		},
		done: {
			bg: 'bg-green-50 dark:bg-green-950/50',
			border: 'border-green-400 dark:border-green-600',
			text: 'text-green-500',
			icon: CircleCheck,
		},
		skipped: {
			bg: 'bg-gray-50 dark:bg-gray-900/50',
			border: 'border-gray-300 dark:border-gray-600',
			text: 'text-gray-400 line-through',
			icon: CircleMinus,
		},
		cancelled: {
			bg: 'bg-red-50 dark:bg-red-950/50',
			border: 'border-red-400 dark:border-red-600',
			text: 'text-red-500',
			icon: CircleX,
		},
	};

	// Priority → badge color
	const PRIORITY_COLORS: Record<number, string> = {
		0: 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300',
		1: 'bg-orange-100 text-orange-700 dark:bg-orange-900/50 dark:text-orange-300',
		2: 'bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300',
		3: 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400',
		4: 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-500',
	};

	// Blocked tasks get red styling to make blockers immediately visible
	const BLOCKED_CONFIG = {
		bg: 'bg-red-50 dark:bg-red-950/50',
		border: 'border-red-400 dark:border-red-600',
		text: 'text-red-500',
		icon: Ban,
	};

	// Priority → node styling for recipe mode (no status, color by priority)
	const PRIORITY_NODE_CONFIG: Record<number, { bg: string; border: string }> = {
		0: { bg: 'bg-red-50 dark:bg-red-950/50', border: 'border-red-400 dark:border-red-600' },
		1: { bg: 'bg-orange-50 dark:bg-orange-950/50', border: 'border-orange-400 dark:border-orange-600' },
		2: { bg: 'bg-blue-50 dark:bg-blue-950/50', border: 'border-blue-400 dark:border-blue-600' },
		3: { bg: 'bg-gray-50 dark:bg-gray-900/50', border: 'border-gray-300 dark:border-gray-700' },
		4: { bg: 'bg-gray-50 dark:bg-gray-900/50', border: 'border-gray-300 dark:border-gray-700' },
	};

	let isRecipe = $derived(data.mode === 'recipe');

	let config = $derived(
		isRecipe
			? { ...(PRIORITY_NODE_CONFIG[data.priority] ?? PRIORITY_NODE_CONFIG[2]), text: 'text-muted-foreground', icon: Circle }
			: data.isBlocked && data.status === 'open'
				? BLOCKED_CONFIG
				: (STATUS_CONFIG[data.status] ?? STATUS_CONFIG.open),
	);
	let StatusIcon = $derived(config.icon);
	let isJob = $derived(data.type === 'job');

	function formatEstimate(seconds: number): string {
		if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
		const hours = seconds / 3600;
		return hours % 1 === 0 ? `${hours}h` : `${hours.toFixed(1)}h`;
	}
</script>

<!-- Wrapper for relative positioning of the [+] button -->
<div class="relative group/tasknode">
	<!-- Target handle (incoming deps — this task depends on something) -->
	<Handle type="target" position={targetPosition} class="!bg-muted-foreground !border-background !w-2 !h-2" />

	<div
		class="rounded-lg border-2 px-3 py-2 shadow-sm transition-shadow {config.bg} {config.border} {selected ? 'ring-2 ring-ring shadow-md' : ''} {data.isFocus ? 'ring-2 ring-primary shadow-md' : ''} {isJob ? 'min-w-[140px]' : 'min-w-[120px]'}"
	>
		<!-- Header: status icon + title -->
		<div class="flex items-center gap-1.5">
			{#if !isRecipe}
				<StatusIcon class="size-3.5 shrink-0 {config.text}" data-testid="status-icon" />
			{/if}
			{#if data.editing}
				<input
					bind:this={inputEl}
					bind:value={editTitle}
					onkeydown={handleInputKeydown}
					onblur={commitTitle}
					class="w-full min-w-0 bg-transparent text-xs font-medium text-foreground outline-none border-b border-primary placeholder:text-muted-foreground"
					placeholder="Task title..."
				/>
			{:else}
				<span
					class="truncate text-xs font-medium text-foreground {!isRecipe && data.status === 'skipped' ? 'line-through text-muted-foreground' : ''}"
					title={data.title}
				>
					{data.title}
				</span>
			{/if}
		</div>

		<!-- Footer: estimated time (recipe) or badges (task/job) -->
		{#if !data.editing}
			{#if isRecipe && data.estimatedTimeSeconds}
				<div class="mt-1">
					<span class="text-[10px] text-muted-foreground">{formatEstimate(data.estimatedTimeSeconds)}</span>
				</div>
			{:else if !isRecipe && (data.priority <= 1 || data.stationName)}
				<div class="mt-1 flex items-center gap-1.5">
					{#if data.priority <= 1}
						<span class="rounded px-1 py-0.5 text-[9px] font-medium leading-none {PRIORITY_COLORS[data.priority] ?? ''}">
							P{data.priority}
						</span>
					{/if}
					{#if data.stationName}
						<span class="rounded bg-secondary px-1 py-0.5 text-[9px] text-secondary-foreground leading-none">
							{data.stationName}
						</span>
					{/if}
				</div>
			{/if}
		{/if}
	</div>

	<!-- Source handle (outgoing deps — something depends on this task) -->
	<Handle type="source" position={sourcePosition} class="!bg-muted-foreground !border-background !w-2 !h-2" />

	<!-- [+] Serial node button — visible on hover or when selected -->
	{#if data.onAddSerial}
		<button
			class="absolute -bottom-5 left-1/2 -translate-x-1/2 z-10 flex size-4 items-center justify-center rounded-full border border-border bg-background text-muted-foreground opacity-0 shadow-sm transition-opacity hover:text-foreground group-hover/tasknode:opacity-100 {selected ? 'opacity-100' : ''}"
			title="Add serial node (Alt+S)"
			onmousedown={(e) => { e.stopPropagation(); e.preventDefault(); }}
			onclick={(e) => { e.stopPropagation(); e.preventDefault(); data.onAddSerial?.(); }}
		>
			<Plus class="size-2.5" />
		</button>
	{/if}
</div>
