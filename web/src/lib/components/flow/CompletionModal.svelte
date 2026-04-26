<script lang="ts">
	import type { TaskResponse, CompleteTaskResponse } from '$lib/types/task';
	import { taskApi } from '$lib/api/task';
	import { timeEntryApi, type TimeEntryResponse, type UpdateTimeEntryRequest, type CreateTimeEntryRequest } from '$lib/api/timeEntry';
	import { formatDuration } from '$lib/utils/time';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import * as Dialog from '$lib/components/ui/dialog';
	import {
		Loader2,
		CircleCheck,
		ArrowRight,
		Plus,
		Trash2,
		Clock,
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

	// --- Time entry state ---

	let entries = $state<TimeEntryResponse[]>([]);
	let loading = $state(false);
	let totalDurationSecs = $state(0);

	// Track inline edits keyed by entry ID → field
	let editingEntry = $state<string | null>(null);

	// --- Submission state ---

	type ModalPhase = 'confirm-time' | 'submitting' | 'cascade' | 'completed';

	let phase = $state<ModalPhase>('confirm-time');
	let completionResponse = $state<CompleteTaskResponse | null>(null);
	let errorMessage = $state<string | null>(null);
	let addingEntry = $state(false);

	// --- Cascade state ---
	let cascadeSelected = $state<Set<string>>(new Set());

	// Load time entries when modal opens
	$effect(() => {
		if (open) {
			phase = 'confirm-time';
			completionResponse = null;
			errorMessage = null;
			editingEntry = null;
			loadEntries();
		}
	});

	async function loadEntries(): Promise<void> {
		loading = true;
		try {
			const result = await timeEntryApi.list(spaceId, task.id);
			entries = result.items;
			totalDurationSecs = result.totalDurationSecs;
		} catch {
			entries = [];
			totalDurationSecs = 0;
		} finally {
			loading = false;
		}
	}

	// --- Computed total ---

	/** Total from all entries, including any running one's live elapsed. */
	let computedTotal = $derived.by(() => {
		let total = 0;
		for (const entry of entries) {
			total += entry.elapsedSecs;
			if (!entry.endedAt && !entry.isPaused) {
				// Running entry — add live delta
				const activeStart = entry.resumedAt || entry.startedAt;
				total += Math.max(0, Math.floor((Date.now() - new Date(activeStart).getTime()) / 1000));
			}
		}
		return total;
	});

	// --- Entry formatting helpers ---

	function formatTime(dateStr: string | null | undefined): string {
		if (!dateStr) return '--';
		return new Date(dateStr).toLocaleTimeString(undefined, {
			hour: '2-digit',
			minute: '2-digit',
		});
	}

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString(undefined, {
			month: 'short',
			day: 'numeric',
		});
	}

	function entryDuration(entry: TimeEntryResponse): number {
		let elapsed = entry.elapsedSecs;
		if (!entry.endedAt && !entry.isPaused) {
			const activeStart = entry.resumedAt || entry.startedAt;
			elapsed += Math.max(0, Math.floor((Date.now() - new Date(activeStart).getTime()) / 1000));
		}
		return elapsed;
	}

	function isRunning(entry: TimeEntryResponse): boolean {
		return !entry.endedAt;
	}

	// --- Entry actions ---

	async function updateEntry(entryId: string, data: UpdateTimeEntryRequest): Promise<void> {
		try {
			await timeEntryApi.update(spaceId, task.id, entryId, data);
			await loadEntries();
		} catch (e) {
			console.error('Failed to update time entry:', e);
		}
		editingEntry = null;
	}

	async function deleteEntry(entryId: string): Promise<void> {
		try {
			await timeEntryApi.remove(spaceId, task.id, entryId);
			await loadEntries();
		} catch (e) {
			console.error('Failed to delete time entry:', e);
		}
	}

	async function addManualEntry(): Promise<void> {
		addingEntry = true;
		try {
			const now = new Date();
			const thirtyMinAgo = new Date(now.getTime() - 30 * 60 * 1000);
			const data: CreateTimeEntryRequest = {
				startedAt: thirtyMinAgo.toISOString(),
				endedAt: now.toISOString(),
			};
			await timeEntryApi.create(spaceId, task.id, data);
			await loadEntries();
		} catch (e) {
			console.error('Failed to add time entry:', e);
		} finally {
			addingEntry = false;
		}
	}

	// --- Duration editing ---

	let durationEditValue = $state('');

	function startDurationEdit(entry: TimeEntryResponse): void {
		editingEntry = entry.id;
		const dur = entryDuration(entry);
		const h = Math.floor(dur / 3600);
		const m = Math.floor((dur % 3600) / 60);
		durationEditValue = h > 0 ? `${h}h ${m}m` : `${m}m`;
	}

	function parseDurationInput(input: string): number | null {
		const trimmed = input.trim().toLowerCase();
		if (!trimmed) return null;

		// Try "Xh Ym" or "Xh" or "Ym" or plain minutes number
		let totalSecs = 0;
		const hMatch = trimmed.match(/(\d+)\s*h/);
		const mMatch = trimmed.match(/(\d+)\s*m/);
		if (hMatch) totalSecs += parseInt(hMatch[1]) * 3600;
		if (mMatch) totalSecs += parseInt(mMatch[1]) * 60;
		if (!hMatch && !mMatch) {
			// Try as plain number (minutes)
			const num = parseInt(trimmed);
			if (isNaN(num)) return null;
			totalSecs = num * 60;
		}
		return totalSecs;
	}

	async function saveDurationEdit(entry: TimeEntryResponse): Promise<void> {
		const secs = parseDurationInput(durationEditValue);
		if (secs == null || secs <= 0) {
			editingEntry = null;
			return;
		}
		await updateEntry(entry.id, { elapsedSecs: secs });
	}

	// --- Completion flow ---

	async function confirmAndComplete(): Promise<void> {
		phase = 'submitting';
		errorMessage = null;
		try {
			// Auto-stop running timer if one exists
			const running = entries.find((e) => !e.endedAt);
			if (running) {
				await timeEntryApi.stop(spaceId, task.id).catch(() => {});
			}

			const response = await taskApi.completeTask(spaceId, task.id, {
				actualTimeSeconds: computedTotal,
			});
			completionResponse = response;

			// If there are unresolved blockers, show cascade prompt
			if (response.unresolvedBlockers && response.unresolvedBlockers.length > 0) {
				cascadeSelected = new Set(response.unresolvedBlockers.map((b) => b.id));
				phase = 'cascade';
			} else {
				phase = 'completed';
			}
		} catch (e) {
			errorMessage = e instanceof Error ? e.message : 'Failed to complete task';
			phase = 'confirm-time';
		}
	}

	/** Complete selected blockers in cascade, then show completed phase. */
	async function completeCascade(): Promise<void> {
		if (!completionResponse) return;
		phase = 'submitting';
		try {
			const promises = Array.from(cascadeSelected).map((blockerId) =>
				taskApi.completeTask(spaceId, blockerId, {}).catch(() => {}),
			);
			await Promise.all(promises);
		} catch {
			// Best-effort — don't block the flow
		}
		phase = 'completed';
	}

	/** Skip cascade and go straight to completed. */
	function skipCascade(): void {
		phase = 'completed';
	}

	function toggleCascadeItem(id: string): void {
		const next = new Set(cascadeSelected);
		if (next.has(id)) {
			next.delete(id);
		} else {
			next.add(id);
		}
		cascadeSelected = next;
	}

	function handleSaveAndNext(): void {
		if (completionResponse) {
			oncomplete?.(completionResponse);
		}
		open = false;
	}

	function handleClose(): void {
		if (phase === 'completed' && completionResponse) {
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
	<Dialog.Content class="sm:max-w-lg">
		{#if phase === 'confirm-time' || phase === 'submitting'}
			<!-- Phase 1: Review time entries -->
			<Dialog.Header>
				<Dialog.Title class="flex items-center gap-2">
					<CircleCheck class="size-5 text-green-600" />
					Complete Task
				</Dialog.Title>
				<Dialog.Description>
					Review time entries for "{task.title}" before completing.
				</Dialog.Description>
			</Dialog.Header>

			<div class="space-y-3 py-4">
				{#if loading}
					<div class="flex items-center justify-center py-6 text-muted-foreground">
						<Loader2 class="size-4 animate-spin mr-2" />
						Loading time entries...
					</div>
				{:else if entries.length === 0}
					<p class="text-sm text-muted-foreground text-center py-4">
						No time entries recorded for this task.
					</p>
				{:else}
					<!-- Time entry list -->
					<div class="max-h-60 overflow-y-auto space-y-1" data-testid="time-entry-list">
						{#each entries as entry (entry.id)}
							<div class="flex items-center gap-2 rounded-md border px-3 py-2 text-sm {isRunning(entry) ? 'border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-950/30' : ''}">
								<!-- Date + time range -->
								<div class="flex-1 min-w-0">
									<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
										<span>{formatDate(entry.startedAt)}</span>
										<span>{formatTime(entry.startedAt)}</span>
										<span>–</span>
										{#if isRunning(entry)}
											<span class="text-green-600 dark:text-green-400 font-medium">running</span>
										{:else}
											<span>{formatTime(entry.endedAt)}</span>
										{/if}
									</div>
									{#if entry.notes}
										<div class="text-xs text-muted-foreground truncate mt-0.5">{entry.notes}</div>
									{/if}
								</div>

								<!-- Duration (editable on click) -->
								{#if editingEntry === entry.id}
									<Input
										type="text"
										bind:value={durationEditValue}
										class="w-20 h-7 text-xs text-right"
										autofocus
										onblur={() => saveDurationEdit(entry)}
										onkeydown={(e: KeyboardEvent) => {
											if (e.key === 'Enter') (e.target as HTMLInputElement).blur();
											if (e.key === 'Escape') { editingEntry = null; }
										}}
									/>
								{:else}
									<button
										class="text-xs font-mono text-right min-w-[4rem] hover:text-foreground text-muted-foreground cursor-pointer"
										onclick={() => startDurationEdit(entry)}
										title="Click to edit duration"
									>
										{formatDuration(entryDuration(entry))}
									</button>
								{/if}

								<!-- Delete button -->
								<Button
									variant="ghost"
									size="icon-sm"
									class="size-6 text-muted-foreground hover:text-destructive shrink-0"
									onclick={() => deleteEntry(entry.id)}
									title="Delete entry"
								>
									<Trash2 class="size-3" />
								</Button>
							</div>
						{/each}
					</div>
				{/if}

				<!-- Add manual entry -->
				<Button
					variant="outline"
					size="sm"
					class="w-full gap-1.5"
					onclick={addManualEntry}
					disabled={addingEntry || phase === 'submitting'}
				>
					{#if addingEntry}
						<Loader2 class="size-3.5 animate-spin" />
					{:else}
						<Plus class="size-3.5" />
					{/if}
					Add Time Entry
				</Button>

				<!-- Total -->
				<div class="flex items-center justify-between pt-2 border-t" data-testid="completion-total">
					<span class="text-sm font-medium flex items-center gap-1.5">
						<Clock class="size-4 text-muted-foreground" />
						Total
					</span>
					<span class="text-sm font-mono font-medium">{formatDuration(computedTotal)}</span>
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

		{:else if phase === 'cascade' && completionResponse?.unresolvedBlockers}
			<!-- Phase 2: Cascade — unresolved blockers -->
			<Dialog.Header>
				<Dialog.Title class="flex items-center gap-2">
					<CircleCheck class="size-5 text-green-600" />
					Task Completed — Blockers Remaining
				</Dialog.Title>
				<Dialog.Description>
					"{task.title}" is done, but these blocking tasks are still unresolved.
					Complete them now or handle later.
				</Dialog.Description>
			</Dialog.Header>

			<div class="py-4 space-y-2">
				<div class="max-h-60 overflow-y-auto space-y-1">
					{#each completionResponse.unresolvedBlockers as blocker (blocker.id)}
						<label class="flex items-center gap-3 rounded-md border px-3 py-2 cursor-pointer hover:bg-muted/50 transition-colors">
							<input
								type="checkbox"
								checked={cascadeSelected.has(blocker.id)}
								onchange={() => toggleCascadeItem(blocker.id)}
								class="size-4 rounded border-border"
							/>
							<div class="flex-1 min-w-0">
								<span class="text-sm font-medium text-foreground truncate block">{blocker.title}</span>
								<span class="text-xs text-muted-foreground">{blocker.id} &middot; {blocker.status}</span>
							</div>
						</label>
					{/each}
				</div>
			</div>

			<Dialog.Footer>
				<Button variant="outline" onclick={skipCascade}>
					Later
				</Button>
				<Button
					class="bg-green-600 hover:bg-green-700 text-white"
					onclick={completeCascade}
					disabled={cascadeSelected.size === 0}
				>
					Complete Selected ({cascadeSelected.size})
				</Button>
			</Dialog.Footer>

		{:else if phase === 'completed' && completionResponse}
			<!-- Phase 3: Completed — show next task navigation -->
			<Dialog.Header>
				<Dialog.Title class="flex items-center gap-2">
					<CircleCheck class="size-5 text-green-600" />
					Task Completed
				</Dialog.Title>
				<Dialog.Description>
					"{task.title}" has been marked as done.
					Time recorded: {formatDuration(computedTotal)}.
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
