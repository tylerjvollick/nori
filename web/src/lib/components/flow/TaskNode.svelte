<script lang="ts">
	import { Handle, Position } from '@xyflow/svelte';
	import type { TaskStatus, TaskType } from '$lib/types/task';
	import {
		Circle,
		CircleDot,
		CircleCheck,
		CirclePause,
		CircleMinus,
		CircleX,
	} from 'lucide-svelte';

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
		};
		selected: boolean;
		[key: string]: unknown;
	} = $props();

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

	let config = $derived(STATUS_CONFIG[data.status] ?? STATUS_CONFIG.open);
	let StatusIcon = $derived(config.icon);
	let isJob = $derived(data.type === 'job');
</script>

<!-- Target handle (incoming deps — this task depends on something) -->
<Handle type="target" position={Position.Left} class="!bg-muted-foreground !border-background !w-2 !h-2" />

<div
	class="rounded-lg border-2 px-3 py-2 shadow-sm transition-shadow {config.bg} {config.border} {selected ? 'ring-2 ring-ring shadow-md' : ''} {isJob ? 'min-w-[180px]' : 'min-w-[160px]'}"
>
	<!-- Header: status icon + title -->
	<div class="flex items-center gap-1.5">
		<StatusIcon class="size-3.5 shrink-0 {config.text}" />
		<span
			class="truncate text-xs font-medium text-foreground {data.status === 'skipped' ? 'line-through text-muted-foreground' : ''}"
			title={data.title}
		>
			{data.title}
		</span>
	</div>

	<!-- Footer: ID + badges -->
	<div class="mt-1 flex items-center gap-1.5">
		<span class="font-mono text-[10px] text-muted-foreground">{data.taskId}</span>
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
</div>

<!-- Source handle (outgoing deps — something depends on this task) -->
<Handle type="source" position={Position.Right} class="!bg-muted-foreground !border-background !w-2 !h-2" />
