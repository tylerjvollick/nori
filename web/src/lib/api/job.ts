import type { TaskResponse, TaskListResponse } from '$lib/types/task';
import type { TaskTreeResponse } from './task';
import type { RecipeResponse } from '$lib/types/recipe';
import { apiClient } from './client';

const BASE = '/api/v1/jobs';

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
	async listJobs(params?: ListJobsParams): Promise<TaskListResponse> {
		return apiClient.get<TaskListResponse>(`${BASE}${toQueryString(params && { ...params })}`);
	}

	async getJob(id: string): Promise<TaskResponse> {
		return apiClient.get<TaskResponse>(`${BASE}/${id}`);
	}

	async getJobTasks(id: string): Promise<TaskTreeResponse> {
		return apiClient.get<TaskTreeResponse>(`${BASE}/${id}/tasks`);
	}

	async getJobCostSummary(id: string): Promise<unknown> {
		return apiClient.get(`${BASE}/${id}/cost-summary`);
	}

	async saveAsRecipe(
		jobId: string,
		data: { name: string; description?: string; backfillEstimatedFromActual?: boolean },
	): Promise<RecipeResponse> {
		return apiClient.post<RecipeResponse>(`${BASE}/${jobId}/save-as-recipe`, data);
	}
}

export const jobApi = new JobApi();
