<script lang="ts">
	import type { TaskResponse, TaskStatus, CompleteTaskResponse } from '$lib/types/task';
	import { taskApi } from '$lib/api/task';
	import { timeEntryApi } from '$lib/api/timeEntry';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import {
		CircleCheck,
		Play,
		SkipForward,
		Loader2,
	} from '@lucide/svelte';
	import CompletionModal from './CompletionModal.svelte';

	type ActionType = 'start' | 'complete' | 'skip';

	interface Props {
		task: TaskResponse;
		spaceId: string;
		/** Layout: 'bar' for full button bar, 'compact' for small icon buttons */
		layout?: 'bar' | 'compact';
		/** Called after a successful action with the updated task */
		onaction?: (updated: TaskResponse) => void;
		/** Called after completion with navigation info (next task ID). */
		oncomplete?: (response: CompleteTaskResponse) => void;
	}

	let { task, spaceId, layout = 'bar', onaction, oncomplete }: Props = $props();

	let loadingAction = $state<ActionType | null>(null);
	let showSkipDialog = $state(false);
	let showCompletionModal = $state(false);

	/** Determine available actions for the current task status. */
	function getActions(status: TaskStatus): ActionType[] {
		switch (status) {
			case 'open':
				return ['start', 'skip'];
			case 'in_progress':
				return ['complete', 'skip'];
			default:
				return [];
		}
	}

	/** Actions excluded from compact layout (board cards) to prevent accidental taps. */
	const compactExcluded: ActionType[] = ['start', 'skip'];

	let actions = $derived(
		layout === 'compact'
			? getActions(task.status).filter((a) => !compactExcluded.includes(a))
			: getActions(task.status),
	);

	const actionConfig: Record<ActionType, {
		label: string;
		variant: 'default' | 'secondary' | 'destructive' | 'outline' | 'ghost';
		class?: string;
	}> = {
		start: { label: 'Start', variant: 'default' },
		complete: { label: 'Complete', variant: 'default', class: 'bg-green-600 hover:bg-green-700 text-white' },
		skip: { label: 'Skip', variant: 'outline' },
	};

	async function executeAction(action: ActionType): Promise<void> {
		if (loadingAction) return;

		// Skip requires confirmation
		if (action === 'skip') {
			showSkipDialog = true;
			return;
		}

		// Complete opens the completion modal (time confirmation + next task nav)
		if (action === 'complete') {
			showCompletionModal = true;
			return;
		}

		await performAction(action);
	}

	/** Handle completion modal result — notify parent with both callbacks. */
	function handleCompletionDone(response: CompleteTaskResponse): void {
		// Notify parent about the task update (so tree reloads)
		onaction?.(response);
		// Notify parent about completion navigation info
		oncomplete?.(response);
	}

	async function confirmSkip(): Promise<void> {
		showSkipDialog = false;
		await performAction('skip');
	}

	async function performAction(action: ActionType): Promise<void> {
		loadingAction = action;
		try {
			let updated: TaskResponse;
			switch (action) {
				case 'start':
					updated = await taskApi.startTask(spaceId, task.id);
					// Also start a timer for tracking.
					await timeEntryApi.start(spaceId, task.id).catch((e) =>
						console.warn('Time entry start failed (non-blocking):', e));
					break;
				case 'skip':
					updated = await taskApi.skipTask(spaceId, task.id);
					break;
				default:
					return; // 'complete' handled by modal
			}
			onaction?.(updated);
		} catch (e) {
			// TODO: toast notification for errors
			console.error(`Failed to ${action} task ${task.id}:`, e);
		} finally {
			loadingAction = null;
		}
	}
</script>

{#if actions.length > 0}
	{#if layout === 'bar'}
		<!-- Full button bar for detail panel -->
		<div class="flex items-center gap-2 flex-wrap">
			{#each actions as action (action)}
				{@const cfg = actionConfig[action]}
				<Button
					variant={cfg.variant}
					size="sm"
					class={cfg.class ?? ''}
					disabled={loadingAction !== null}
					onclick={(e: MouseEvent) => { e.preventDefault(); e.stopPropagation(); executeAction(action); }}
				>
					{#if loadingAction === action}
						<Loader2 class="size-4 animate-spin" />
					{:else if action === 'start'}
						<Play class="size-4" />
					{:else if action === 'complete'}
						<CircleCheck class="size-4" />
					{:else if action === 'skip'}
						<SkipForward class="size-4" />
					{/if}
					{cfg.label}
				</Button>
			{/each}
		</div>
	{:else}
		<!-- Compact icon buttons for task cards -->
		<div class="flex items-center gap-1">
			{#each actions as action (action)}
				{@const cfg = actionConfig[action]}
				<Button
					variant="ghost"
					size="icon-sm"
					class="size-7 {action === 'skip' ? 'text-muted-foreground hover:text-destructive' : ''} {action === 'complete' ? 'text-green-600 hover:text-green-700 hover:bg-green-50 dark:hover:bg-green-950' : ''}"
					disabled={loadingAction !== null}
					title={cfg.label}
					onclick={(e: MouseEvent) => { e.preventDefault(); e.stopPropagation(); executeAction(action); }}
				>
					{#if loadingAction === action}
						<Loader2 class="size-3.5 animate-spin" />
					{:else if action === 'start'}
						<Play class="size-3.5" />
					{:else if action === 'complete'}
						<CircleCheck class="size-3.5" />
					{:else if action === 'skip'}
						<SkipForward class="size-3.5" />
					{/if}
				</Button>
			{/each}
		</div>
	{/if}
{/if}

<!-- Skip confirmation dialog -->
<Dialog.Root bind:open={showSkipDialog}>
	<Dialog.Content>
		<Dialog.Header>
			<Dialog.Title>Skip Task</Dialog.Title>
			<Dialog.Description>
				Are you sure you want to skip "{task.title}"? This action marks the task as skipped and cannot be easily undone.
			</Dialog.Description>
		</Dialog.Header>
		<Dialog.Footer>
			<Button variant="outline" onclick={() => (showSkipDialog = false)}>
				Cancel
			</Button>
			<Button
				variant="destructive"
				disabled={loadingAction !== null}
				onclick={confirmSkip}
			>
				{#if loadingAction === 'skip'}
					<Loader2 class="size-4 animate-spin" />
				{/if}
				Skip Task
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>

<!-- Completion flow modal (time confirmation + next task navigation) -->
<CompletionModal
	{task}
	{spaceId}
	bind:open={showCompletionModal}
	oncomplete={handleCompletionDone}
/>
