<script lang="ts">
	import { Handle, Position } from '@xyflow/svelte';
	import type { TaskStatus, TaskType } from '$lib/types/task';
	import type { GraphDirection } from '$lib/stores/graph';
	import {
		Circle,
		CircleDashed,
		CircleDot,
		CircleCheck,
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
			/** Called when Tab is pressed during inline editing — commits title and creates serial node. */
			onTabCommit?: (title: string) => void;
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
		if (e.key === 'Enter' && e.metaKey) {
			// Cmd+Enter: commit title and create serial downstream node
			e.preventDefault();
			const trimmed = editTitle.trim() || 'New Task';
			data.onTabCommit?.(trimmed);
		} else if (e.key === 'Enter') {
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

	// Status → color mapping (matches TaskCard/TaskTree patterns).
	// `border` is the muted/inactive border. `activeBorder` + `ring` are used
	// when the node is highlighted (keyboard-selected or the in-focus task) so
	// the highlight matches the node's status color instead of always green.
	const STATUS_CONFIG: Record<
		TaskStatus,
		{ bg: string; border: string; activeBorder: string; ring: string; text: string; icon: typeof Circle }
	> = {
		backlog: {
			bg: 'bg-slate-50 dark:bg-slate-900/50',
			border: 'border-slate-200/40 border-dashed dark:border-slate-700/30',
			activeBorder: 'border-slate-400 dark:border-slate-500',
			ring: 'ring-slate-400/40',
			text: 'text-slate-400',
			icon: CircleDashed,
		},
		open: {
			bg: 'bg-gray-50 dark:bg-gray-900/50',
			border: 'border-gray-200/50 dark:border-gray-800/40',
			activeBorder: 'border-gray-400 dark:border-gray-500',
			ring: 'ring-gray-400/40',
			text: 'text-gray-500',
			icon: Circle,
		},
		in_progress: {
			bg: 'bg-blue-50 dark:bg-blue-950/50',
			border: 'border-blue-200/50 dark:border-blue-900/40',
			activeBorder: 'border-blue-500 dark:border-blue-400',
			ring: 'ring-blue-500/40',
			text: 'text-blue-500',
			icon: CircleDot,
		},
		done: {
			bg: 'bg-green-50 dark:bg-green-950/50',
			border: 'border-green-200/50 dark:border-green-900/40',
			activeBorder: 'border-green-500 dark:border-green-400',
			ring: 'ring-green-500/40',
			text: 'text-green-500',
			icon: CircleCheck,
		},
		skipped: {
			bg: 'bg-gray-50 dark:bg-gray-900/50',
			border: 'border-gray-200/40 dark:border-gray-800/30',
			activeBorder: 'border-gray-400 dark:border-gray-500',
			ring: 'ring-gray-400/40',
			text: 'text-gray-400 line-through',
			icon: CircleMinus,
		},
		cancelled: {
			bg: 'bg-red-50 dark:bg-red-950/50',
			border: 'border-red-200/50 dark:border-red-900/40',
			activeBorder: 'border-red-500 dark:border-red-400',
			ring: 'ring-red-500/40',
			text: 'text-red-500',
			icon: CircleX,
		},
	};

	// Blocked tasks get red styling to make blockers immediately visible
	const BLOCKED_CONFIG = {
		bg: 'bg-red-50 dark:bg-red-950/50',
		border: 'border-red-200/50 dark:border-red-900/40',
		activeBorder: 'border-red-500 dark:border-red-400',
		ring: 'ring-red-500/40',
		text: 'text-red-500',
		icon: Ban,
	};

	// Recipe mode: all nodes use the same neutral styling (no priority coloring)
	const RECIPE_NODE_CONFIG = {
		bg: 'bg-background',
		border: 'border-border/40',
		activeBorder: 'border-primary',
		ring: 'ring-primary/40',
	};

	let isRecipe = $derived(data.mode === 'recipe');

	let config = $derived(
		isRecipe
			? { ...RECIPE_NODE_CONFIG, text: 'text-muted-foreground', icon: Circle }
			: data.isBlocked && data.status === 'open'
				? BLOCKED_CONFIG
				: (STATUS_CONFIG[data.status] ?? STATUS_CONFIG.open),
	);
	let StatusIcon = $derived(config.icon);

	// Border/ring for the node based on highlight state. Only the keyboard-
	// selected node is highlighted (vivid status border + status-colored ring);
	// every other node — including the in-focus task once the cursor moves off
	// it — uses the muted status border. The current task starts selected, so it
	// is highlighted on load, but hjkl moves the single highlight with the cursor.
	let highlightClass = $derived(
		selected
			? `${config.activeBorder} ring-2 ${config.ring} shadow-md`
			: `${config.border} shadow-sm`,
	);

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
		class="rounded-lg border-2 px-3 py-2 w-[180px] h-[72px] flex flex-col transition-shadow {config.bg} {highlightClass}"
	>
		<!-- Title area: 2-line clamp -->
		<div class="flex items-start gap-1.5 flex-1 min-h-0">
			{#if !isRecipe}
				<StatusIcon class="size-3.5 shrink-0 mt-0.5 {config.text}" data-testid="status-icon" />
			{/if}
			{#if data.editing}
				<input
					bind:this={inputEl}
					bind:value={editTitle}
					onkeydown={handleInputKeydown}
					onblur={commitTitle}
					class="w-full min-w-0 bg-transparent text-xs font-light text-foreground outline-none border-b border-primary placeholder:text-muted-foreground"
					placeholder="Task title..."
				/>
			{:else}
				<span
					class="text-xs font-light text-foreground leading-tight line-clamp-2 {!isRecipe && data.status === 'skipped' ? 'line-through text-muted-foreground' : ''}"
					title={data.title}
				>
					{data.title}
				</span>
			{/if}
		</div>

		<!-- Footer: station (left) + estimated time (right) -->
		{#if !data.editing && (data.stationName || data.estimatedTimeSeconds)}
			<div class="flex items-center justify-between mt-auto pt-0.5">
				<span class="text-[9px] text-muted-foreground truncate max-w-[60%]">
					{data.stationName ?? ''}
				</span>
				{#if data.estimatedTimeSeconds}
					<span class="text-[9px] text-muted-foreground shrink-0">
						{formatEstimate(data.estimatedTimeSeconds)}
					</span>
				{/if}
			</div>
		{/if}
	</div>

	<!-- Source handle (outgoing deps — something depends on this task) -->
	<Handle type="source" position={sourcePosition} class="!bg-muted-foreground !border-background !w-2 !h-2" />

	<!-- [+] Serial node button — visible on hover or when selected -->
	{#if data.onAddSerial}
		<button
			class="absolute -bottom-5 left-1/2 -translate-x-1/2 z-10 flex size-4 items-center justify-center rounded-full border border-border bg-background text-muted-foreground opacity-0 shadow-sm transition-opacity hover:text-foreground group-hover/tasknode:opacity-100 {selected ? 'opacity-100' : ''}"
			title="Add serial node (⌥S)"
			onmousedown={(e) => { e.stopPropagation(); e.preventDefault(); }}
			onclick={(e) => { e.stopPropagation(); e.preventDefault(); data.onAddSerial?.(); }}
		>
			<Plus class="size-2.5" />
		</button>
	{/if}
</div>
