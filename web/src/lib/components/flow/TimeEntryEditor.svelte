<script lang="ts">
	import {
		timeEntryApi,
		type TimeEntryResponse,
		type CreateTimeEntryRequest,
		type UpdateTimeEntryRequest,
	} from '$lib/api/timeEntry';
	import { formatDuration } from '$lib/utils/time';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Loader2, Plus, Trash2, Clock, Pencil } from '@lucide/svelte';

	interface Props {
		spaceId: string;
		taskId: string;
		open: boolean;
		/** Called after any change (add/edit/delete). */
		onchange?: () => void;
	}

	let { spaceId, taskId, open = $bindable(), onchange }: Props = $props();

	let entries = $state<TimeEntryResponse[]>([]);
	let totalDurationSecs = $state(0);
	let loading = $state(false);
	let addingEntry = $state(false);
	let deletingId = $state<string | null>(null);

	// --- Duration editing ---
	let editingEntryId = $state<string | null>(null);
	let durationEditValue = $state('');

	// Load entries when modal opens
	$effect(() => {
		if (open) {
			editingEntryId = null;
			loadEntries();
		}
	});

	async function loadEntries(): Promise<void> {
		loading = true;
		try {
			const result = await timeEntryApi.list(spaceId, taskId);
			entries = result.items;
			totalDurationSecs = result.totalDurationSecs;
		} catch {
			entries = [];
			totalDurationSecs = 0;
		} finally {
			loading = false;
		}
	}

	// --- Formatting ---

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

	// --- Actions ---

	async function addManualEntry(): Promise<void> {
		addingEntry = true;
		try {
			const now = new Date();
			const thirtyMinAgo = new Date(now.getTime() - 30 * 60 * 1000);
			const data: CreateTimeEntryRequest = {
				startedAt: thirtyMinAgo.toISOString(),
				endedAt: now.toISOString(),
			};
			await timeEntryApi.create(spaceId, taskId, data);
			await loadEntries();
			onchange?.();
		} catch (e) {
			console.error('Failed to add time entry:', e);
		} finally {
			addingEntry = false;
		}
	}

	async function deleteEntry(entryId: string): Promise<void> {
		deletingId = entryId;
		try {
			await timeEntryApi.remove(spaceId, taskId, entryId);
			await loadEntries();
			onchange?.();
		} catch (e) {
			console.error('Failed to delete time entry:', e);
		} finally {
			deletingId = null;
		}
	}

	// --- Duration editing ---

	function startDurationEdit(entry: TimeEntryResponse): void {
		editingEntryId = entry.id;
		const dur = entryDuration(entry);
		const h = Math.floor(dur / 3600);
		const m = Math.floor((dur % 3600) / 60);
		durationEditValue = h > 0 ? `${h}h ${m}m` : `${m}m`;
	}

	function parseDurationInput(input: string): number | null {
		const trimmed = input.trim().toLowerCase();
		if (!trimmed) return null;

		let totalSecs = 0;
		const hMatch = trimmed.match(/(\d+)\s*h/);
		const mMatch = trimmed.match(/(\d+)\s*m/);
		if (hMatch) totalSecs += parseInt(hMatch[1]) * 3600;
		if (mMatch) totalSecs += parseInt(mMatch[1]) * 60;
		if (!hMatch && !mMatch) {
			const num = parseInt(trimmed);
			if (isNaN(num)) return null;
			totalSecs = num * 60;
		}
		return totalSecs;
	}

	async function saveDurationEdit(entry: TimeEntryResponse): Promise<void> {
		const secs = parseDurationInput(durationEditValue);
		if (secs == null || secs <= 0) {
			editingEntryId = null;
			return;
		}
		try {
			await timeEntryApi.update(spaceId, taskId, entry.id, { elapsedSecs: secs });
			await loadEntries();
			onchange?.();
		} catch (e) {
			console.error('Failed to update time entry:', e);
		}
		editingEntryId = null;
	}

	/** Computed total including live running entries. */
	let computedTotal = $derived.by(() => {
		let total = 0;
		for (const entry of entries) {
			total += entryDuration(entry);
		}
		return total;
	});
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="sm:max-w-lg">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<Clock class="size-5 text-muted-foreground" />
				Time Entries
			</Dialog.Title>
			<Dialog.Description>
				Add, edit, or remove time records for this task.
			</Dialog.Description>
		</Dialog.Header>

		<div class="space-y-3 py-4">
			{#if loading}
				<div class="flex items-center justify-center py-6 text-muted-foreground">
					<Loader2 class="size-4 animate-spin mr-2" />
					Loading...
				</div>
			{:else if entries.length === 0}
				<p class="text-sm text-muted-foreground text-center py-4">
					No time entries yet.
				</p>
			{:else}
				<div class="max-h-72 overflow-y-auto space-y-1" data-testid="time-entry-list">
					{#each entries as entry (entry.id)}
						<div
							class="flex items-center gap-2 rounded-md border px-3 py-2 text-sm {!entry.endedAt ? 'border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-950/30' : ''}"
							data-testid="time-entry-row"
						>
							<!-- Date + time range -->
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
									<span>{formatDate(entry.startedAt)}</span>
									<span>{formatTime(entry.startedAt)}</span>
									<span>&ndash;</span>
									{#if !entry.endedAt}
										<span class="text-green-600 dark:text-green-400 font-medium">running</span>
									{:else}
										<span>{formatTime(entry.endedAt)}</span>
									{/if}
								</div>
								{#if entry.notes}
									<div class="text-xs text-muted-foreground truncate mt-0.5">{entry.notes}</div>
								{/if}
							</div>

							<!-- Duration (editable) -->
							{#if editingEntryId === entry.id}
								<Input
									type="text"
									bind:value={durationEditValue}
									class="w-24 h-7 text-xs text-right"
									autofocus
									onblur={() => saveDurationEdit(entry)}
									onkeydown={(e: KeyboardEvent) => {
										if (e.key === 'Enter') (e.target as HTMLInputElement).blur();
										if (e.key === 'Escape') { editingEntryId = null; }
									}}
									data-testid="duration-edit-input"
								/>
							{:else}
								<button
									class="flex items-center gap-1 text-xs font-mono text-right min-w-[4rem] hover:text-foreground text-muted-foreground cursor-pointer"
									onclick={() => startDurationEdit(entry)}
									title="Click to edit duration"
									data-testid="duration-display"
								>
									{formatDuration(entryDuration(entry))}
									<Pencil class="size-2.5 opacity-0 group-hover:opacity-100" />
								</button>
							{/if}

							<!-- Delete button -->
							<Button
								variant="ghost"
								size="icon-sm"
								class="size-6 text-muted-foreground hover:text-destructive shrink-0"
								onclick={() => deleteEntry(entry.id)}
								disabled={deletingId === entry.id}
								title="Delete entry"
								data-testid="delete-entry-btn"
							>
								{#if deletingId === entry.id}
									<Loader2 class="size-3 animate-spin" />
								{:else}
									<Trash2 class="size-3" />
								{/if}
							</Button>
						</div>
					{/each}
				</div>
			{/if}

			<!-- Add entry button -->
			<Button
				variant="outline"
				size="sm"
				class="w-full gap-1.5"
				onclick={addManualEntry}
				disabled={addingEntry}
				data-testid="add-time-entry-btn"
			>
				{#if addingEntry}
					<Loader2 class="size-3.5 animate-spin" />
				{:else}
					<Plus class="size-3.5" />
				{/if}
				Add Time Entry
			</Button>

			<!-- Total -->
			<div class="flex items-center justify-between pt-2 border-t">
				<span class="text-sm font-medium flex items-center gap-1.5">
					<Clock class="size-4 text-muted-foreground" />
					Total
				</span>
				<span class="text-sm font-mono font-medium" data-testid="time-entry-total">
					{formatDuration(computedTotal)}
				</span>
			</div>
		</div>

		<Dialog.Footer>
			<Button onclick={() => { open = false; }}>
				Done
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
