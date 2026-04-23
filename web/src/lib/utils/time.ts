/**
 * Shared time formatting utilities for task time display.
 */

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
 * Compute the effective time spent on a task, including live elapsed time
 * for active tasks.
 *
 * For active tasks: actualTimeSeconds + elapsed time since startedAt.
 * For all other statuses: actualTimeSeconds as-is.
 *
 * @param actualTimeSeconds - Accumulated time from the task model.
 * @param status - Current task status.
 * @param startedAt - ISO date string for when the task was last started (may be null).
 * @param now - Current time (pass in for reactivity / ticking).
 * @returns Total seconds spent.
 */
export function computeTimeSpent(
	actualTimeSeconds: number,
	status: string,
	startedAt: string | null | undefined,
	now: Date = new Date(),
): number {
	if (status === 'active' && startedAt) {
		const startTime = new Date(startedAt).getTime();
		const elapsedSecs = Math.max(0, Math.floor((now.getTime() - startTime) / 1000));
		return actualTimeSeconds + elapsedSecs;
	}
	return actualTimeSeconds;
}

/**
 * Format the time spent on a task as a human-readable string.
 * Delegates to computeTimeSpent + formatDuration.
 */
export function formatTimeSpent(
	actualTimeSeconds: number,
	status: string,
	startedAt: string | null | undefined,
	now: Date = new Date(),
): string {
	const total = computeTimeSpent(actualTimeSeconds, status, startedAt, now);
	return formatDuration(total);
}
