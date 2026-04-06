import type {
	TaskResponse,
	TaskListResponse,
	TaskDep,
	CreateTaskRequest,
	UpdateTaskRequest,
	AddChildTaskRequest,
	AddNoteRequest,
} from '$lib/types/task';
import { apiClient } from './client';

const BASE = '/api/v1/tasks';

/** Params accepted by the list tasks endpoint. */
export interface ListTasksParams {
	status?: string;
	type?: string;
	stationId?: string;
	assigneeId?: string;
	parentId?: string;
	offset?: number;
	limit?: number;
}

/** Params accepted by the ready tasks endpoint. */
export interface ReadyTasksParams {
	stationId?: string;
	assigneeId?: string;
}

/** Tree response: task with nested children. Matches backend dtos.TaskTreeResponse. */
export interface TaskTreeResponse extends TaskResponse {
	children: TaskTreeResponse[];
}

/** Deps response: blockers and dependents. Matches backend dtos.TaskDepsResponse. */
export interface TaskDepsResponse {
	blockers: TaskDep[];
	dependents: TaskDep[];
}

/** Build a query string from an object of optional params. */
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

class TaskApi {
	async listTasks(params?: ListTasksParams): Promise<TaskListResponse> {
		return apiClient.get<TaskListResponse>(`${BASE}${toQueryString(params && { ...params })}`);
	}

	async getTask(id: string): Promise<TaskResponse> {
		return apiClient.get<TaskResponse>(`${BASE}/${id}`);
	}

	async createTask(data: CreateTaskRequest): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(BASE, data);
	}

	async updateTask(id: string, data: UpdateTaskRequest): Promise<TaskResponse> {
		return apiClient.put<TaskResponse>(`${BASE}/${id}`, data);
	}

	async deleteTask(id: string): Promise<void> {
		return apiClient.delete<void>(`${BASE}/${id}`);
	}

	async claimTask(id: string): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`${BASE}/${id}/claim`);
	}

	async completeTask(id: string): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`${BASE}/${id}/complete`);
	}

	async pauseTask(id: string): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`${BASE}/${id}/pause`);
	}

	async resumeTask(id: string): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`${BASE}/${id}/resume`);
	}

	async skipTask(id: string): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`${BASE}/${id}/skip`);
	}

	async addChildTask(parentId: string, data: AddChildTaskRequest): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`${BASE}/${parentId}/children`, data);
	}

	async addNote(id: string, data: AddNoteRequest): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`${BASE}/${id}/notes`, data);
	}

	async getReadyTasks(params?: ReadyTasksParams): Promise<TaskResponse[]> {
		const qs = toQueryString(params && { ...params });
		const result = await apiClient.get<{ items: TaskResponse[]; total: number }>(
			`${BASE}/ready${qs}`,
		);
		return result.items;
	}

	async getTaskTree(id: string): Promise<TaskTreeResponse> {
		return apiClient.get<TaskTreeResponse>(`${BASE}/${id}/tree`);
	}

	async getTaskDeps(id: string): Promise<TaskDepsResponse> {
		return apiClient.get<TaskDepsResponse>(`${BASE}/${id}/deps`);
	}
}

export const taskApi = new TaskApi();
