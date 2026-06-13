import type {
	TaskResponse,
	TaskListResponse,
	TaskDep,
	CreateTaskRequest,
	UpdateTaskRequest,
	AddChildTaskRequest,
	AddNoteRequest,
	CompleteTaskRequest,
	CompleteTaskResponse,
	SubTaskResponse,
	SubTaskListResponse,
	SubTaskImage,
	CreateSubTaskRequest,
	UpdateSubTaskRequest,
	ReorderSubTasksRequest,
	TaskMedia,
	TaskMediaListResponse,
} from '$lib/types/task';
import { apiClient } from './client';

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
	async listTasks(spaceId: string, params?: ListTasksParams): Promise<TaskListResponse> {
		const base = `/api/v1/spaces/${spaceId}/tasks`;
		return apiClient.get<TaskListResponse>(`${base}${toQueryString(params && { ...params })}`);
	}

	async getTask(spaceId: string, id: string): Promise<TaskResponse> {
		return apiClient.get<TaskResponse>(`/api/v1/spaces/${spaceId}/tasks/${id}`);
	}

	async createTask(spaceId: string, data: CreateTaskRequest): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`/api/v1/spaces/${spaceId}/tasks`, data);
	}

	async updateTask(spaceId: string, id: string, data: UpdateTaskRequest): Promise<TaskResponse> {
		return apiClient.put<TaskResponse>(`/api/v1/spaces/${spaceId}/tasks/${id}`, data);
	}

	async deleteTask(spaceId: string, id: string): Promise<void> {
		return apiClient.delete<void>(`/api/v1/spaces/${spaceId}/tasks/${id}`);
	}

	async setStatus(spaceId: string, id: string, status: import('$lib/types/task').TaskStatus): Promise<TaskResponse> {
		return apiClient.put<TaskResponse>(`/api/v1/spaces/${spaceId}/tasks/${id}/status`, { status });
	}

	async startTask(spaceId: string, id: string): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`/api/v1/spaces/${spaceId}/tasks/${id}/start`);
	}

	async completeTask(spaceId: string, id: string, data?: CompleteTaskRequest): Promise<CompleteTaskResponse> {
		return apiClient.post<CompleteTaskResponse>(`/api/v1/spaces/${spaceId}/tasks/${id}/complete`, data);
	}

	async skipTask(spaceId: string, id: string): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`/api/v1/spaces/${spaceId}/tasks/${id}/skip`);
	}

	async addChildTask(spaceId: string, parentId: string, data: AddChildTaskRequest): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`/api/v1/spaces/${spaceId}/tasks/${parentId}/children`, data);
	}

	async addNote(spaceId: string, id: string, data: AddNoteRequest): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`/api/v1/spaces/${spaceId}/tasks/${id}/notes`, data);
	}

	async getReadyTasks(spaceId: string, params?: ReadyTasksParams): Promise<TaskResponse[]> {
		const qs = toQueryString(params && { ...params });
		const result = await apiClient.get<{ items: TaskResponse[]; total: number }>(
			`/api/v1/spaces/${spaceId}/tasks/ready${qs}`,
		);
		return result.items;
	}

	async getTaskTree(spaceId: string, id: string): Promise<TaskTreeResponse> {
		return apiClient.get<TaskTreeResponse>(`/api/v1/spaces/${spaceId}/tasks/${id}/tree`);
	}

	async getTaskDeps(spaceId: string, id: string): Promise<TaskDepsResponse> {
		return apiClient.get<TaskDepsResponse>(`/api/v1/spaces/${spaceId}/tasks/${id}/deps`);
	}

	/** Add a dependency: blockerId blocks targetTaskId. */
	async addDep(spaceId: string, blockerId: string, targetTaskId: string, type: string = 'blocks'): Promise<TaskDep> {
		return apiClient.post<TaskDep>(`/api/v1/spaces/${spaceId}/tasks/${blockerId}/deps`, { targetTaskId, type });
	}

	/** Remove a dependency edge by its UUID. */
	async removeDep(spaceId: string, taskId: string, depId: string): Promise<void> {
		return apiClient.delete<void>(`/api/v1/spaces/${spaceId}/tasks/${taskId}/deps/${depId}`);
	}

	// --- Sub-task methods ---

	async listSubTasks(spaceId: string, taskId: string): Promise<SubTaskListResponse> {
		return apiClient.get<SubTaskListResponse>(`/api/v1/spaces/${spaceId}/tasks/${taskId}/subtasks`);
	}

	async createSubTask(spaceId: string, taskId: string, data: CreateSubTaskRequest): Promise<SubTaskResponse> {
		return apiClient.post<SubTaskResponse>(`/api/v1/spaces/${spaceId}/tasks/${taskId}/subtasks`, data);
	}

	async updateSubTask(spaceId: string, taskId: string, subtaskId: string, data: UpdateSubTaskRequest): Promise<SubTaskResponse> {
		return apiClient.put<SubTaskResponse>(`/api/v1/spaces/${spaceId}/tasks/${taskId}/subtasks/${subtaskId}`, data);
	}

	async deleteSubTask(spaceId: string, taskId: string, subtaskId: string): Promise<void> {
		return apiClient.delete<void>(`/api/v1/spaces/${spaceId}/tasks/${taskId}/subtasks/${subtaskId}`);
	}

	async reorderSubTasks(spaceId: string, taskId: string, data: ReorderSubTasksRequest): Promise<void> {
		return apiClient.put<void>(`/api/v1/spaces/${spaceId}/tasks/${taskId}/subtasks/reorder`, data);
	}

	// --- Task media (photos/videos) ---

	async listTaskMedia(spaceId: string, taskId: string): Promise<TaskMediaListResponse> {
		return apiClient.get<TaskMediaListResponse>(`/api/v1/spaces/${spaceId}/tasks/${taskId}/media`);
	}

	async uploadTaskMedia(spaceId: string, taskId: string, file: File): Promise<TaskMedia> {
		const form = new FormData();
		form.append('file', file);
		return apiClient.uploadFile<TaskMedia>(`/api/v1/spaces/${spaceId}/tasks/${taskId}/media`, form);
	}

	async deleteTaskMedia(spaceId: string, taskId: string, mediaId: string): Promise<void> {
		return apiClient.delete<void>(`/api/v1/spaces/${spaceId}/tasks/${taskId}/media/${mediaId}`);
	}

	// --- Sub-task images ---

	async uploadSubTaskImage(spaceId: string, taskId: string, subtaskId: string, file: File): Promise<SubTaskImage> {
		const form = new FormData();
		form.append('file', file);
		return apiClient.uploadFile<SubTaskImage>(
			`/api/v1/spaces/${spaceId}/tasks/${taskId}/subtasks/${subtaskId}/images`,
			form,
		);
	}

	async deleteSubTaskImage(spaceId: string, taskId: string, subtaskId: string, imageId: string): Promise<void> {
		return apiClient.delete<void>(
			`/api/v1/spaces/${spaceId}/tasks/${taskId}/subtasks/${subtaskId}/images/${imageId}`,
		);
	}

}

export const taskApi = new TaskApi();
