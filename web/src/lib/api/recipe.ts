import { apiClient } from './client';
import type {
	RecipeResponse,
	RecipeListResponse,
	RecipeVersionResponse,
	CreateRecipeRequest,
	UpdateRecipeRequest,
	RollRecipeRequest,
	CreateRecipeVersionRequest,
} from '$lib/types/recipe';
import type { TaskResponse } from '$lib/types/task';

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

export interface ListRecipesParams {
	slug?: string;
	categoryId?: string;
	isActive?: boolean;
	offset?: number;
	limit?: number;
}

class RecipeApi {
	async listRecipes(spaceId: string, params?: ListRecipesParams): Promise<RecipeListResponse> {
		return apiClient.get<RecipeListResponse>(`/api/v1/spaces/${spaceId}/recipes${toQueryString(params && { ...params })}`);
	}

	async getRecipe(spaceId: string, id: string): Promise<RecipeResponse> {
		return apiClient.get<RecipeResponse>(`/api/v1/spaces/${spaceId}/recipes/${id}`);
	}

	async createRecipe(spaceId: string, data: CreateRecipeRequest): Promise<RecipeResponse> {
		return apiClient.post<RecipeResponse>(`/api/v1/spaces/${spaceId}/recipes`, data);
	}

	async updateRecipe(spaceId: string, id: string, data: UpdateRecipeRequest): Promise<RecipeResponse> {
		return apiClient.put<RecipeResponse>(`/api/v1/spaces/${spaceId}/recipes/${id}`, data);
	}

	async deleteRecipe(spaceId: string, id: string): Promise<void> {
		return apiClient.delete<void>(`/api/v1/spaces/${spaceId}/recipes/${id}`);
	}

	async listVersions(spaceId: string, recipeId: string): Promise<RecipeVersionResponse[]> {
		return apiClient.get<RecipeVersionResponse[]>(`/api/v1/spaces/${spaceId}/recipes/${recipeId}/versions`);
	}

	async createVersion(spaceId: string, recipeId: string, data?: CreateRecipeVersionRequest): Promise<RecipeVersionResponse> {
		return apiClient.post<RecipeVersionResponse>(`/api/v1/spaces/${spaceId}/recipes/${recipeId}/versions`, data);
	}

	async publishVersion(spaceId: string, recipeId: string): Promise<RecipeVersionResponse> {
		return apiClient.post<RecipeVersionResponse>(`/api/v1/spaces/${spaceId}/recipes/${recipeId}/publish`);
	}

	async rollRecipe(spaceId: string, recipeId: string, data?: RollRecipeRequest): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`/api/v1/spaces/${spaceId}/recipes/${recipeId}/roll`, data);
	}
}

export const recipeApi = new RecipeApi();
