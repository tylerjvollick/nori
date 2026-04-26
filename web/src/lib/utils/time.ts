/**
 * Shared time formatting utilities for task time display.
 */

import type { TimeEntryResponse } from '$lib/api/timeEntry';

/**
 * Format a duration in seconds as a human-readable string.
 * Returns '--' for zero seconds.
 *
 * Examples: '2h 15m', '45m', '5s', '--'
 */
export function formatDuration(seconds: number): string {
	if (seconds <= 0) return '--';
	const h = Math.floor(seconds / 3600);
	const m = Math.floor((seconds % 3600) / 60);
	if (h > 0) return `${h}h ${m}m`;
	if (m > 0) return `${m}m`;
	return `${seconds}s`;
}

/**
 * Compute the live elapsed seconds for a single time entry.
 * If the entry is running (not paused, not ended), adds live delta from resumedAt.
 */
export function computeEntryElapsed(entry: TimeEntryResponse, now: Date = new Date()): number {
	let elapsed = entry.elapsedSecs;
	if (!entry.endedAt && !entry.isPaused) {
		// Timer is actively running — add delta since last resume/start.
		const activeStart = entry.resumedAt || entry.startedAt;
		const startTime = new Date(activeStart).getTime();
		elapsed += Math.max(0, Math.floor((now.getTime() - startTime) / 1000));
	}
	return elapsed;
}

/**
 * Compute total time spent on a task from its time entries.
 * Sums all entry elapsed values, with live delta for running entries.
 */
export function computeTimeSpent(
	entries: TimeEntryResponse[],
	now: Date = new Date(),
): number {
	let total = 0;
	for (const entry of entries) {
		total += computeEntryElapsed(entry, now);
	}
	return total;
}

/**
 * Check if any time entry is currently running (active timer).
 */
export function hasRunningTimer(entries: TimeEntryResponse[]): boolean {
	return entries.some((e) => !e.endedAt && !e.isPaused);
}

/**
 * Get the currently running (or paused but active) entry, if any.
 */
export function getActiveEntry(entries: TimeEntryResponse[]): TimeEntryResponse | undefined {
	return entries.find((e) => !e.endedAt);
}

/**
 * Format the time spent on a task as a human-readable string.
 */
export function formatTimeSpent(
	entries: TimeEntryResponse[],
	now: Date = new Date(),
): string {
	const total = computeTimeSpent(entries, now);
	return formatDuration(total);
}
