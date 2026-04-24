import type { PaginatedResponse } from './common';

// --- Enums ---

export type TaskStatus = 'open' | 'active' | 'paused' | 'done' | 'skipped' | 'cancelled';

export type TaskType = 'job' | 'task' | 'milestone' | 'gate';

// --- Response types ---

/** Matches backend dtos.TaskResponse exactly. */
export interface TaskResponse {
	id: string;
	spaceId: string;
	parentId?: string | null;
	recipeId?: string | null;
	recipeVersionId?: number | null;
	stationId?: string | null;
	customerId?: string | null;
	assignedToId?: string | null;
	createdById: string;
	type: TaskType;
	status: TaskStatus;
	title: string;
	description?: string | null;
	quantity: number;
	priority: number;
	displayOrder: number;
	dueDate?: string | null;
	startedAt?: string | null;
	pausedAt?: string | null;
	completedAt?: string | null;
	actualTimeSeconds: number;
	estimatedTimeSeconds?: number | null;
	deviationNotes?: string | null;
	metadata?: Record<string, unknown> | null;
	createdAt: string;
	updatedAt: string;
}

/** Paginated list of tasks. Matches backend dtos.TaskListResponse. */
export type TaskListResponse = PaginatedResponse<TaskResponse>;

/** Matches backend dtos.TaskDepResponse. */
export interface TaskDep {
	id: string;
	fromTaskId: string;
	toTaskId: string;
	type: string;
	createdAt: string;
}

// --- Request types ---

export interface CreateTaskRequest {
	title: string;
	description?: string;
	type: TaskType;
	parentId?: string;
	stationId?: string;
	priority?: number;
}

export interface UpdateTaskRequest {
	title?: string;
	description?: string;
	stationId?: string;
	priority?: number;
	status?: TaskStatus;
	assignedToId?: string; // UUID string to assign, empty string to unassign
}

export interface AddChildTaskRequest {
	title: string;
	description?: string;
	type?: TaskType;
	stationId?: string;
	afterId?: string;
}

export interface AddNoteRequest {
	text: string;
}

/** Tasks returned by GET /tasks/ready have the same shape as TaskResponse. */
export type ReadyTask = TaskResponse;

// --- Sub-task types ---

export interface SubTaskImage {
	id: string;
	subTaskId: string;
	imageUrl: string;
	displayOrder: number;
	createdAt: string;
}

export interface SubTaskResponse {
	id: string;
	taskId: string;
	title: string;
	description?: string | null;
	displayOrder: number;
	images: SubTaskImage[];
	createdAt: string;
	updatedAt: string;
}

export interface SubTaskListResponse {
	items: SubTaskResponse[];
	total: number;
}

export interface CreateSubTaskRequest {
	title: string;
	description?: string;
	displayOrder?: number;
}

export interface UpdateSubTaskRequest {
	title?: string;
	description?: string;
}

export interface ReorderSubTasksRequest {
	ids: string[];
}

// --- Completion flow types ---

/** Request body for POST /tasks/:id/complete. All fields optional. */
export interface CompleteTaskRequest {
	actualTimeSeconds?: number;
}

/** Response from POST /tasks/:id/complete. Extends TaskResponse with navigation hints. */
export interface CompleteTaskResponse extends TaskResponse {
	nextTaskId?: string | null;
	nextTaskTitle?: string | null;
}
