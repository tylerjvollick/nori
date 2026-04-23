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

const BASE = '/api/v1/recipes';

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
	async listRecipes(params?: ListRecipesParams): Promise<RecipeListResponse> {
		return apiClient.get<RecipeListResponse>(`${BASE}${toQueryString(params && { ...params })}`);
	}

	async getRecipe(id: string): Promise<RecipeResponse> {
		return apiClient.get<RecipeResponse>(`${BASE}/${id}`);
	}

	async createRecipe(data: CreateRecipeRequest): Promise<RecipeResponse> {
		return apiClient.post<RecipeResponse>(BASE, data);
	}

	async updateRecipe(id: string, data: UpdateRecipeRequest): Promise<RecipeResponse> {
		return apiClient.put<RecipeResponse>(`${BASE}/${id}`, data);
	}

	async deleteRecipe(id: string): Promise<void> {
		return apiClient.delete<void>(`${BASE}/${id}`);
	}

	async listVersions(recipeId: string): Promise<RecipeVersionResponse[]> {
		return apiClient.get<RecipeVersionResponse[]>(`${BASE}/${recipeId}/versions`);
	}

	async createVersion(recipeId: string, data?: CreateRecipeVersionRequest): Promise<RecipeVersionResponse> {
		return apiClient.post<RecipeVersionResponse>(`${BASE}/${recipeId}/versions`, data);
	}

	async publishVersion(recipeId: string): Promise<RecipeVersionResponse> {
		return apiClient.post<RecipeVersionResponse>(`${BASE}/${recipeId}/publish`);
	}

	async rollRecipe(recipeId: string, data?: RollRecipeRequest): Promise<TaskResponse> {
		return apiClient.post<TaskResponse>(`${BASE}/${recipeId}/roll`, data);
	}
}

export const recipeApi = new RecipeApi();
