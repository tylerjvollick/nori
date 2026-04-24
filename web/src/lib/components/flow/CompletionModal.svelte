<script lang="ts">
	import type { TaskResponse, CompleteTaskResponse } from '$lib/types/task';
	import { taskApi } from '$lib/api/task';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Dialog from '$lib/components/ui/dialog';
	import { computeTimeSpent, formatDuration } from '$lib/utils/time';
	import {
		Loader2,
		CircleCheck,
		ArrowRight,
	} from '@lucide/svelte';

	interface Props {
		task: TaskResponse;
		spaceId: string;
		open: boolean;
		/** Called after successful completion with the API response. */
		oncomplete?: (response: CompleteTaskResponse) => void;
		/** Called when the dialog is cancelled / closed without completing. */
		oncancel?: () => void;
	}

	let { task, spaceId, open = $bindable(), oncomplete, oncancel }: Props = $props();

	// --- Time editing state ---

	/** Compute total time spent at the moment the modal opens. */
	let initialTotalSeconds = $derived(
		computeTimeSpent(task.actualTimeSeconds, task.status, task.startedAt),
	);

	let hours = $state(0);
	let minutes = $state(0);

	// Reset hours/minutes when modal opens or task changes
	$effect(() => {
		if (open) {
			const total = computeTimeSpent(task.actualTimeSeconds, task.status, task.startedAt);
			hours = Math.floor(total / 3600);
			minutes = Math.floor((total % 3600) / 60);
		}
	});

	let editedTotalSeconds = $derived((hours * 3600) + (minutes * 60));
	let timeWasEdited = $derived(editedTotalSeconds !== initialTotalSeconds);

	// --- Submission state ---

	type ModalPhase = 'confirm-time' | 'submitting' | 'completed';

	let phase = $state<ModalPhase>('confirm-time');
	let completionResponse = $state<CompleteTaskResponse | null>(null);
	let errorMessage = $state<string | null>(null);

	// Reset phase when modal opens
	$effect(() => {
		if (open) {
			phase = 'confirm-time';
			completionResponse = null;
			errorMessage = null;
		}
	});

	async function confirmAndComplete(): Promise<void> {
		phase = 'submitting';
		errorMessage = null;
		try {
			const response = await taskApi.completeTask(spaceId, task.id, {
				actualTimeSeconds: editedTotalSeconds,
			});
			completionResponse = response;
			phase = 'completed';
		} catch (e) {
			errorMessage = e instanceof Error ? e.message : 'Failed to complete task';
			phase = 'confirm-time';
		}
	}

	function handleSaveAndNext(): void {
		if (completionResponse) {
			oncomplete?.(completionResponse);
		}
		open = false;
	}

	function handleClose(): void {
		if (phase === 'completed' && completionResponse) {
			// Even on close after completion, notify parent
			oncomplete?.(completionResponse);
		} else {
			oncancel?.();
		}
		open = false;
	}

	function handleOpenChange(isOpen: boolean): void {
		if (!isOpen) {
			handleClose();
		}
	}
</script>

<Dialog.Root bind:open onOpenChange={handleOpenChange}>
	<Dialog.Content class="sm:max-w-md">
		{#if phase === 'confirm-time' || phase === 'submitting'}
			<!-- Phase 1: Confirm time -->
			<Dialog.Header>
				<Dialog.Title class="flex items-center gap-2">
					<CircleCheck class="size-5 text-green-600" />
					Complete Task
				</Dialog.Title>
				<Dialog.Description>
					Review the time logged for "{task.title}" before completing.
				</Dialog.Description>
			</Dialog.Header>

			<div class="space-y-4 py-4">
				<!-- Current computed time (read-only context) -->
				<div class="text-sm text-muted-foreground">
					Computed time: <span class="font-medium text-foreground">{formatDuration(initialTotalSeconds)}</span>
				</div>

				<!-- Editable time inputs -->
				<div class="flex items-end gap-3">
					<div class="space-y-1.5">
						<Label for="hours">Hours</Label>
						<Input
							id="hours"
							type="number"
							min={0}
							bind:value={hours}
							class="w-20"
							disabled={phase === 'submitting'}
						/>
					</div>
					<div class="space-y-1.5">
						<Label for="minutes">Minutes</Label>
						<Input
							id="minutes"
							type="number"
							min={0}
							max={59}
							bind:value={minutes}
							class="w-20"
							disabled={phase === 'submitting'}
						/>
					</div>
					{#if timeWasEdited}
						<span class="text-xs text-muted-foreground pb-1.5">(edited)</span>
					{/if}
				</div>

				{#if errorMessage}
					<p class="text-sm text-destructive">{errorMessage}</p>
				{/if}
			</div>

			<Dialog.Footer>
				<Button
					variant="outline"
					onclick={() => handleClose()}
					disabled={phase === 'submitting'}
				>
					Cancel
				</Button>
				<Button
					class="bg-green-600 hover:bg-green-700 text-white"
					onclick={confirmAndComplete}
					disabled={phase === 'submitting'}
				>
					{#if phase === 'submitting'}
						<Loader2 class="size-4 animate-spin" />
					{/if}
					Confirm & Complete
				</Button>
			</Dialog.Footer>

		{:else if phase === 'completed' && completionResponse}
			<!-- Phase 2: Completed — show next task navigation -->
			<Dialog.Header>
				<Dialog.Title class="flex items-center gap-2">
					<CircleCheck class="size-5 text-green-600" />
					Task Completed
				</Dialog.Title>
				<Dialog.Description>
					"{task.title}" has been marked as done.
					{#if timeWasEdited}
						Time recorded: {formatDuration(editedTotalSeconds)}.
					{/if}
				</Dialog.Description>
			</Dialog.Header>

			<div class="py-4">
				{#if completionResponse.nextTaskId}
					<div class="rounded-lg border bg-muted/50 p-4 space-y-2">
						<p class="text-sm font-medium text-foreground">Up next</p>
						<div class="flex items-center gap-2">
							<ArrowRight class="size-4 text-muted-foreground shrink-0" />
							<span class="font-mono text-sm">{completionResponse.nextTaskId}</span>
							{#if completionResponse.nextTaskTitle}
								<span class="text-sm text-muted-foreground truncate">
									{completionResponse.nextTaskTitle}
								</span>
							{/if}
						</div>
					</div>
				{:else}
					<p class="text-sm text-muted-foreground">
						No downstream tasks are waiting. You're all caught up.
					</p>
				{/if}
			</div>

			<Dialog.Footer>
				{#if completionResponse.nextTaskId}
					<Button variant="outline" onclick={() => handleClose()}>
						Close
					</Button>
					<Button onclick={handleSaveAndNext} class="gap-2">
						Save & Next
						<ArrowRight class="size-4" />
					</Button>
				{:else}
					<Button onclick={() => handleClose()}>
						Done
					</Button>
				{/if}
			</Dialog.Footer>
		{/if}
	</Dialog.Content>
</Dialog.Root>
