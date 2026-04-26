<script lang="ts">
	import type { TaskStatus } from '$lib/types/task';
	import { timeEntryApi, type TimeEntryResponse } from '$lib/api/timeEntry';
	import { taskApi } from '$lib/api/task';
	import { Button } from '$lib/components/ui/button';
	import { Play, Pause, Square, Loader2 } from '@lucide/svelte';
	import { formatDuration } from '$lib/utils/time';

	interface Props {
		spaceId: string;
		taskId: string;
		taskStatus: TaskStatus;
		/** All time entries for this task. */
		entries: TimeEntryResponse[];
		/** Called after a timer action completes (entries or status may have changed). */
		onupdate?: () => void;
	}

	let { spaceId, taskId, taskStatus, entries, onupdate }: Props = $props();

	let actionInProgress = $state<'start' | 'pause' | 'resume' | 'stop' | null>(null);

	// --- Derived state from entries ---

	/** The currently active (non-ended) entry, if any. */
	let activeEntry = $derived(entries.find((e) => !e.endedAt) ?? null);

	/** Whether the timer is running (active + not paused). */
	let isRunning = $derived(activeEntry !== null && !activeEntry.isPaused);

	/** Whether the timer is paused (active + paused). */
	let isPaused = $derived(activeEntry !== null && activeEntry.isPaused);

	/** Whether the timer is stopped (no active entry). */
	let isStopped = $derived(activeEntry === null);

	// --- Live elapsed time ---

	let now = $state(Date.now());
	let tickInterval: ReturnType<typeof setInterval> | null = null;

	$effect(() => {
		if (isRunning) {
			tickInterval = setInterval(() => { now = Date.now(); }, 1000);
		} else {
			if (tickInterval) { clearInterval(tickInterval); tickInterval = null; }
		}
		return () => { if (tickInterval) clearInterval(tickInterval); };
	});

	/** Total elapsed seconds across all entries, including live segment. */
	let totalElapsed = $derived.by(() => {
		let total = 0;
		for (const entry of entries) {
			total += entry.elapsedSecs;
			if (!entry.endedAt && !entry.isPaused) {
				const activeStart = entry.resumedAt || entry.startedAt;
				total += Math.max(0, Math.floor((now - new Date(activeStart).getTime()) / 1000));
			}
		}
		return total;
	});

	// --- Actions ---

	async function startTimer(): Promise<void> {
		actionInProgress = 'start';
		try {
			// Auto-set status to in_progress if currently open
			if (taskStatus === 'open') {
				await taskApi.setStatus(spaceId, taskId, 'in_progress');
			}
			await timeEntryApi.start(spaceId, taskId);
			onupdate?.();
		} catch (e) {
			console.error('Failed to start timer:', e);
		} finally {
			actionInProgress = null;
		}
	}

	async function pauseTimer(): Promise<void> {
		actionInProgress = 'pause';
		try {
			await timeEntryApi.pause(spaceId, taskId);
			onupdate?.();
		} catch (e) {
			console.error('Failed to pause timer:', e);
		} finally {
			actionInProgress = null;
		}
	}

	async function resumeTimer(): Promise<void> {
		actionInProgress = 'resume';
		try {
			await timeEntryApi.resume(spaceId, taskId);
			onupdate?.();
		} catch (e) {
			console.error('Failed to resume timer:', e);
		} finally {
			actionInProgress = null;
		}
	}

	async function stopTimer(): Promise<void> {
		actionInProgress = 'stop';
		try {
			await timeEntryApi.stop(spaceId, taskId);
			onupdate?.();
		} catch (e) {
			console.error('Failed to stop timer:', e);
		} finally {
			actionInProgress = null;
		}
	}

	let isTerminal = $derived(['done', 'skipped', 'cancelled'].includes(taskStatus));
</script>

<div class="flex items-center gap-2">
	<!-- Timer display -->
	<div class="flex items-center gap-1.5">
		{#if isRunning}
			<span class="relative flex size-2">
				<span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
				<span class="relative inline-flex rounded-full size-2 bg-green-500"></span>
			</span>
		{/if}
		<span class="font-mono text-sm {isRunning ? 'text-foreground font-medium' : 'text-muted-foreground'}">
			{formatDuration(totalElapsed)}
		</span>
	</div>

	<!-- Controls -->
	{#if !isTerminal}
		{#if isStopped}
			<!-- No active session → Start -->
			<Button
				variant="outline"
				size="icon-sm"
				class="size-7 text-green-600 hover:text-green-700 hover:bg-green-50 dark:hover:bg-green-950"
				onclick={startTimer}
				disabled={actionInProgress !== null}
				title="Start timer"
			>
				{#if actionInProgress === 'start'}
					<Loader2 class="size-3.5 animate-spin" />
				{:else}
					<Play class="size-3.5" />
				{/if}
			</Button>
		{:else if isRunning}
			<!-- Running → Pause or Stop -->
			<Button
				variant="outline"
				size="icon-sm"
				class="size-7"
				onclick={pauseTimer}
				disabled={actionInProgress !== null}
				title="Pause timer"
			>
				{#if actionInProgress === 'pause'}
					<Loader2 class="size-3.5 animate-spin" />
				{:else}
					<Pause class="size-3.5" />
				{/if}
			</Button>
			<Button
				variant="outline"
				size="icon-sm"
				class="size-7 text-red-500 hover:text-red-600"
				onclick={stopTimer}
				disabled={actionInProgress !== null}
				title="Stop timer"
			>
				{#if actionInProgress === 'stop'}
					<Loader2 class="size-3.5 animate-spin" />
				{:else}
					<Square class="size-3" />
				{/if}
			</Button>
		{:else if isPaused}
			<!-- Paused → Resume or Stop -->
			<Button
				variant="outline"
				size="icon-sm"
				class="size-7 text-green-600 hover:text-green-700 hover:bg-green-50 dark:hover:bg-green-950"
				onclick={resumeTimer}
				disabled={actionInProgress !== null}
				title="Resume timer"
			>
				{#if actionInProgress === 'resume'}
					<Loader2 class="size-3.5 animate-spin" />
				{:else}
					<Play class="size-3.5" />
				{/if}
			</Button>
			<Button
				variant="outline"
				size="icon-sm"
				class="size-7 text-red-500 hover:text-red-600"
				onclick={stopTimer}
				disabled={actionInProgress !== null}
				title="Stop timer"
			>
				{#if actionInProgress === 'stop'}
					<Loader2 class="size-3.5 animate-spin" />
				{:else}
					<Square class="size-3" />
				{/if}
			</Button>
		{/if}
	{/if}
</div>
