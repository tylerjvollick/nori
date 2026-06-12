import { apiClient } from './client';

/** A single time entry (work session) record. */
export interface TimeEntryResponse {
	id: string;
	taskId: string;
	spaceId: string;
	loggedById: string;
	startedAt: string;
	endedAt?: string | null;
	elapsedSecs: number;
	isPaused: boolean;
	pausedAt?: string | null;
	resumedAt?: string | null;
	notes?: string | null;
	createdAt: string;
	updatedAt: string;
}

/** List response with computed total. */
export interface TimeEntryListResponse {
	items: TimeEntryResponse[];
	totalDurationSecs: number;
}

/** Request body for manual time entry creation. */
export interface CreateTimeEntryRequest {
	startedAt: string;
	endedAt?: string;
	elapsedSecs?: number;
	notes?: string;
}

/** Request body for editing a time entry. */
export interface UpdateTimeEntryRequest {
	startedAt?: string;
	endedAt?: string;
	elapsedSecs?: number;
	notes?: string;
}

class TimeEntryApi {
	private basePath(spaceId: string, taskId: string): string {
		return `/api/v1/spaces/${spaceId}/tasks/${taskId}/time-entries`;
	}

	/** Start a timer on a task. Creates a new running time entry. */
	async start(spaceId: string, taskId: string): Promise<TimeEntryResponse> {
		return apiClient.post<TimeEntryResponse>(
			`${this.basePath(spaceId, taskId)}/start`,
		);
	}

	/** Pause the running timer on a task. */
	async pause(spaceId: string, taskId: string): Promise<TimeEntryResponse> {
		return apiClient.post<TimeEntryResponse>(
			`${this.basePath(spaceId, taskId)}/pause`,
		);
	}

	/** Resume the paused timer on a task. */
	async resume(spaceId: string, taskId: string): Promise<TimeEntryResponse> {
		return apiClient.post<TimeEntryResponse>(
			`${this.basePath(spaceId, taskId)}/resume`,
		);
	}

	/** Stop the timer on a task (finalizes the session). */
	async stop(spaceId: string, taskId: string): Promise<TimeEntryResponse> {
		return apiClient.post<TimeEntryResponse>(
			`${this.basePath(spaceId, taskId)}/stop`,
		);
	}

	/** List all time entries for a task. */
	async list(spaceId: string, taskId: string): Promise<TimeEntryListResponse> {
		return apiClient.get<TimeEntryListResponse>(
			this.basePath(spaceId, taskId),
		);
	}

	/** Create a manual time entry. */
	async create(
		spaceId: string,
		taskId: string,
		data: CreateTimeEntryRequest,
	): Promise<TimeEntryResponse> {
		return apiClient.post<TimeEntryResponse>(
			this.basePath(spaceId, taskId),
			data,
		);
	}

	/** Update an existing time entry. */
	async update(
		spaceId: string,
		taskId: string,
		entryId: string,
		data: UpdateTimeEntryRequest,
	): Promise<TimeEntryResponse> {
		return apiClient.put<TimeEntryResponse>(
			`${this.basePath(spaceId, taskId)}/${entryId}`,
			data,
		);
	}

	/** Delete a time entry. */
	async remove(
		spaceId: string,
		taskId: string,
		entryId: string,
	): Promise<void> {
		return apiClient.delete<void>(
			`${this.basePath(spaceId, taskId)}/${entryId}`,
		);
	}
}

export const timeEntryApi = new TimeEntryApi();
