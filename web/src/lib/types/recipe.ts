import type { TaskResponse } from './task';

// --- Enums ---

export type RecipeVersionStatus = 'draft' | 'published' | 'archived';

// --- Response types ---

export interface RecipeResponse {
	id: string;
	spaceId: string;
	name: string;
	slug: string;
	description?: string | null;
	categoryId?: string | null;
	currentVersionId?: number | null;
	extendsRecipeId?: string | null;
	createdById: string;
	isActive: boolean;
	createdAt: string;
	updatedAt: string;
	currentVersion?: RecipeVersionResponse | null;
}

export interface RecipeVersionResponse {
	id: number;
	recipeId: string;
	versionNumber: number;
	status: RecipeVersionStatus;
	content?: string | null;
	rootTaskId?: string | null;
	changeSummary?: string | null;
	authorId: string;
	publishedAt?: string | null;
	taskTree?: TaskTreeNode | null;
	createdAt: string;
	updatedAt: string;
}

/** Recursive task tree node matching backend dtos.TaskTreeResponse. */
export interface TaskTreeNode extends TaskResponse {
	children: TaskTreeNode[];
}

export interface RecipeListResponse {
	items: RecipeResponse[];
	total: number;
	offset: number;
	limit: number;
}

// --- Request types ---

export interface CreateRecipeRequest {
	name: string;
	description?: string;
	categoryId?: string;
}

export interface UpdateRecipeRequest {
	name?: string;
	description?: string;
	categoryId?: string;
	isActive?: boolean;
}

export interface RollRecipeRequest {
	order_qty?: number;
	customer_id?: string;
	due_date?: string;
	title?: string;
}

export interface CreateRecipeVersionRequest {
	content?: string;
	changeSummary?: string;
}
