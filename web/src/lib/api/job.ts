import type { TaskResponse, TaskListResponse } from '$lib/types/task';
import type { TaskTreeResponse } from './task';
import type { RecipeResponse } from '$lib/types/recipe';
import { apiClient } from './client';

export interface ListJobsParams {
	status?: string;
	offset?: number;
	limit?: number;
}

function toQueryString(params?: Record<string, unknown>): string {
	if (!params) return '';
	const entries: [string, string][] = [];
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== null) {
			entries.push([k, String(v)]);
		}
	}
	if (entries.length === 0) return '';
	return `?${new URLSearchParams(entries).toString()}`;
}

class JobApi {
	async listJobs(spaceId: string, params?: ListJobsParams): Promise<TaskListResponse> {
		return apiClient.get<TaskListResponse>(`/api/v1/spaces/${spaceId}/jobs${toQueryString(params && { ...params })}`);
	}

	async getJob(spaceId: string, id: string): Promise<TaskResponse> {
		return apiClient.get<TaskResponse>(`/api/v1/spaces/${spaceId}/jobs/${id}`);
	}

	async getJobTasks(spaceId: string, id: string): Promise<TaskTreeResponse> {
		return apiClient.get<TaskTreeResponse>(`/api/v1/spaces/${spaceId}/jobs/${id}/tasks`);
	}

	async getJobCostSummary(spaceId: string, id: string): Promise<unknown> {
		return apiClient.get(`/api/v1/spaces/${spaceId}/jobs/${id}/cost-summary`);
	}

	async saveAsRecipe(
		spaceId: string,
		jobId: string,
		data: { name: string; description?: string; backfillEstimatedFromActual?: boolean },
	): Promise<RecipeResponse> {
		return apiClient.post<RecipeResponse>(`/api/v1/spaces/${spaceId}/jobs/${jobId}/save-as-recipe`, data);
	}
}

export const jobApi = new JobApi();
