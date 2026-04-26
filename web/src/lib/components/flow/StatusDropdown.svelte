<script lang="ts">
	import type { TaskStatus } from '$lib/types/task';
	import { taskApi } from '$lib/api/task';
	import { Badge } from '$lib/components/ui/badge';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import {
		Circle,
		CircleDot,
		CircleCheck,
		CircleX,
		CircleMinus,
		Loader2,
		ChevronDown,
	} from '@lucide/svelte';
	import type { Component } from 'svelte';

	interface Props {
		status: TaskStatus;
		spaceId: string;
		taskId: string;
		/** Whether the task has unresolved blockers (shows derived "Blocked" indicator). */
		isBlocked?: boolean;
		/** Called after a successful status change. */
		onchange?: (newStatus: TaskStatus) => void;
	}

	let { status, spaceId, taskId, isBlocked = false, onchange }: Props = $props();

	let isUpdating = $state(false);

	interface StatusOption {
		value: TaskStatus;
		label: string;
		icon: Component;
		colorClass: string;
		bgClass: string;
	}

	const STATUS_OPTIONS: StatusOption[] = [
		{ value: 'open', label: 'Open', icon: Circle, colorClass: 'text-muted-foreground', bgClass: 'bg-muted' },
		{ value: 'in_progress', label: 'In Progress', icon: CircleDot, colorClass: 'text-blue-500', bgClass: 'bg-blue-500/10' },
		{ value: 'done', label: 'Done', icon: CircleCheck, colorClass: 'text-green-500', bgClass: 'bg-green-500/10' },
		{ value: 'skipped', label: 'Skipped', icon: CircleMinus, colorClass: 'text-muted-foreground', bgClass: 'bg-muted' },
		{ value: 'cancelled', label: 'Cancelled', icon: CircleX, colorClass: 'text-red-500', bgClass: 'bg-red-500/10' },
	];

	const TERMINAL_STATUSES: TaskStatus[] = ['done', 'skipped', 'cancelled'];

	let currentOption = $derived(STATUS_OPTIONS.find((o) => o.value === status) ?? STATUS_OPTIONS[0]);
	let isTerminal = $derived(TERMINAL_STATUSES.includes(status));

	async function setStatus(newStatus: TaskStatus): Promise<void> {
		if (newStatus === status || isUpdating) return;
		isUpdating = true;
		try {
			await taskApi.setStatus(spaceId, taskId, newStatus);
			onchange?.(newStatus);
		} catch (e) {
			console.error('Failed to set status:', e);
		} finally {
			isUpdating = false;
		}
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger disabled={isTerminal || isUpdating}>
		{#snippet child({ props })}
			<button
				{...props}
				class="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-sm font-medium transition-colors {currentOption.bgClass} {currentOption.colorClass} {isTerminal ? 'opacity-70 cursor-not-allowed' : 'hover:opacity-80 cursor-pointer'} focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1"
			>
				{#if isUpdating}
					<Loader2 class="size-3.5 animate-spin" />
				{:else}
					<currentOption.icon class="size-3.5" />
				{/if}
				{currentOption.label}
				{#if isBlocked && status === 'open'}
					<span class="ml-1 text-[10px] font-semibold text-orange-500">BLOCKED</span>
				{/if}
				{#if !isTerminal}
					<ChevronDown class="size-3 opacity-60" />
				{/if}
			</button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="min-w-[160px]">
		{#each STATUS_OPTIONS as option (option.value)}
			<DropdownMenu.Item
				class="gap-2 {option.value === status ? 'bg-accent' : ''}"
				disabled={option.value === status}
				onclick={() => setStatus(option.value)}
			>
				<option.icon class="size-4 {option.colorClass}" />
				<span>{option.label}</span>
			</DropdownMenu.Item>
		{/each}
	</DropdownMenu.Content>
</DropdownMenu.Root>
