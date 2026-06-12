<script lang="ts">
	import {
		timeEntryApi,
		type TimeEntryResponse,
		type CreateTimeEntryRequest,
	} from '$lib/api/timeEntry';
	import { formatDuration } from '$lib/utils/time';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import * as Sheet from '$lib/components/ui/sheet';
	import * as Popover from '$lib/components/ui/popover';
	import { Calendar } from '$lib/components/ui/calendar';
	import { Loader2, Plus, Trash2, Clock, CalendarIcon } from '@lucide/svelte';
	import {
		CalendarDate,
		today,
		getLocalTimeZone,
		type DateValue,
	} from '@internationalized/date';

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

	// --- Add entry form ---
	let newDuration = $state('30m');
	let newDate = $state<DateValue>(today(getLocalTimeZone()));
	let datePickerOpen = $state(false);

	// --- Duration editing ---
	let editingEntryId = $state<string | null>(null);
	let durationEditValue = $state('');

	// Load entries when sheet opens
	$effect(() => {
		if (open) {
			editingEntryId = null;
			newDuration = '30m';
			newDate = today(getLocalTimeZone());
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

	function formatDateStr(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString(undefined, {
			month: 'short',
			day: 'numeric',
		});
	}

	function formatDateValue(dv: DateValue): string {
		const d = new Date(dv.year, dv.month - 1, dv.day);
		return d.toLocaleDateString(undefined, {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
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

	// --- Duration parsing ---

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

	// --- Add entry ---

	async function addManualEntry(): Promise<void> {
		const secs = parseDurationInput(newDuration);
		if (secs == null || secs <= 0) return;

		addingEntry = true;
		try {
			// Build start/end from selected date + duration
			const endDate = new Date(newDate.year, newDate.month - 1, newDate.day);
			// Default to "now" time on the selected date, or noon if it's a past date
			const todayVal = today(getLocalTimeZone());
			if (newDate.compare(todayVal) === 0) {
				const now = new Date();
				endDate.setHours(now.getHours(), now.getMinutes(), now.getSeconds());
			} else {
				endDate.setHours(17, 0, 0); // Default to 5pm for past dates
			}
			const startDate = new Date(endDate.getTime() - secs * 1000);

			const data: CreateTimeEntryRequest = {
				startedAt: startDate.toISOString(),
				endedAt: endDate.toISOString(),
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

	// --- Delete entry ---

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

	// --- Edit duration ---

	function startDurationEdit(entry: TimeEntryResponse): void {
		editingEntryId = entry.id;
		const dur = entryDuration(entry);
		const h = Math.floor(dur / 3600);
		const m = Math.floor((dur % 3600) / 60);
		durationEditValue = h > 0 ? `${h}h ${m}m` : `${m}m`;
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

	// --- Edit date ---

	let editingDateEntryId = $state<string | null>(null);
	let editDateValue = $state<DateValue>(today(getLocalTimeZone()));
	let editDatePickerOpen = $state(false);

	function startDateEdit(entry: TimeEntryResponse): void {
		editingDateEntryId = entry.id;
		const d = new Date(entry.startedAt);
		editDateValue = new CalendarDate(d.getFullYear(), d.getMonth() + 1, d.getDate());
		editDatePickerOpen = true;
	}

	async function saveDateEdit(entry: TimeEntryResponse, newDateVal: DateValue): Promise<void> {
		editDatePickerOpen = false;
		editingDateEntryId = null;

		const oldStart = new Date(entry.startedAt);
		const newStart = new Date(newDateVal.year, newDateVal.month - 1, newDateVal.day,
			oldStart.getHours(), oldStart.getMinutes(), oldStart.getSeconds());

		const updateData: Record<string, string> = {
			startedAt: newStart.toISOString(),
		};

		if (entry.endedAt) {
			const oldEnd = new Date(entry.endedAt);
			const diff = oldEnd.getTime() - oldStart.getTime();
			const newEnd = new Date(newStart.getTime() + diff);
			updateData.endedAt = newEnd.toISOString();
		}

		try {
			await timeEntryApi.update(spaceId, taskId, entry.id, updateData);
			await loadEntries();
			onchange?.();
		} catch (e) {
			console.error('Failed to update time entry date:', e);
		}
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

<Sheet.Root bind:open>
	<Sheet.Content side="right" class="w-full sm:max-w-md">
		<Sheet.Header>
			<Sheet.Title class="flex items-center gap-2">
				<Clock class="size-5 text-muted-foreground" />
				Time Entries
			</Sheet.Title>
			<Sheet.Description>
				Add, edit, or remove time records for this task.
			</Sheet.Description>
		</Sheet.Header>

		<div class="space-y-4 py-4 px-1">
			<!-- Add new entry form -->
			<div class="flex items-end gap-2" data-testid="add-entry-form">
				<div class="flex-1 space-y-1">
					<label for="new-duration" class="text-xs text-muted-foreground">Duration</label>
					<Input
						id="new-duration"
						type="text"
						bind:value={newDuration}
						placeholder="e.g. 1h 30m"
						class="h-8 text-sm"
						onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter') addManualEntry(); }}
						data-testid="new-duration-input"
					/>
				</div>
				<div class="space-y-1">
					<label class="text-xs text-muted-foreground">Date</label>
					<Popover.Root bind:open={datePickerOpen}>
						<Popover.Trigger>
							{#snippet child({ props })}
								<button
									{...props}
									class="inline-flex items-center gap-1.5 h-8 px-2.5 rounded-md border text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer whitespace-nowrap"
									data-testid="new-date-picker"
								>
									<CalendarIcon class="size-3.5" />
									{formatDateValue(newDate)}
								</button>
							{/snippet}
						</Popover.Trigger>
						<Popover.Content class="w-auto p-0" align="end">
							<Calendar
								type="single"
								bind:value={newDate}
								onValueChange={() => { datePickerOpen = false; }}
							/>
						</Popover.Content>
					</Popover.Root>
				</div>
				<Button
					variant="outline"
					size="sm"
					class="gap-1 h-8 shrink-0"
					onclick={addManualEntry}
					disabled={addingEntry}
					data-testid="add-time-entry-btn"
				>
					{#if addingEntry}
						<Loader2 class="size-3.5 animate-spin" />
					{:else}
						<Plus class="size-3.5" />
					{/if}
					Add
				</Button>
			</div>

			<!-- Entry list -->
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
				<div class="space-y-1" data-testid="time-entry-list">
					{#each entries as entry (entry.id)}
						<div
							class="flex items-center gap-2 rounded-md border px-3 py-2 text-sm {!entry.endedAt ? 'border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-950/30' : ''}"
							data-testid="time-entry-row"
						>
							<!-- Date (editable) -->
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
									{#if editingDateEntryId === entry.id}
										<Popover.Root bind:open={editDatePickerOpen}>
											<Popover.Trigger>
												{#snippet child({ props })}
													<button
														{...props}
														class="inline-flex items-center gap-1 text-xs text-foreground hover:text-primary cursor-pointer"
													>
														<CalendarIcon class="size-3" />
														{formatDateValue(editDateValue)}
													</button>
												{/snippet}
											</Popover.Trigger>
											<Popover.Content class="w-auto p-0" align="start">
												<Calendar
													type="single"
													bind:value={editDateValue}
													onValueChange={(v) => { if (v) saveDateEdit(entry, v); }}
												/>
											</Popover.Content>
										</Popover.Root>
									{:else}
										<button
											class="hover:text-foreground cursor-pointer transition-colors"
											onclick={() => startDateEdit(entry)}
											title="Click to edit date"
										>
											{formatDateStr(entry.startedAt)}
										</button>
									{/if}
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
									class="text-xs font-mono text-right min-w-[4rem] hover:text-foreground text-muted-foreground cursor-pointer transition-colors"
									onclick={() => startDurationEdit(entry)}
									title="Click to edit duration"
									data-testid="duration-display"
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

			<!-- Total -->
			<div class="flex items-center justify-between pt-3 border-t">
				<span class="text-sm font-medium flex items-center gap-1.5">
					<Clock class="size-4 text-muted-foreground" />
					Total
				</span>
				<span class="text-sm font-mono font-medium" data-testid="time-entry-total">
					{formatDuration(computedTotal)}
				</span>
			</div>
		</div>
	</Sheet.Content>
</Sheet.Root>
