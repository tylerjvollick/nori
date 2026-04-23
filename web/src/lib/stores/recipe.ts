import { writable } from 'svelte/store';
import { recipeApi } from '$lib/api/recipe';
import type {
	RecipeResponse,
	RecipeVersionResponse,
	CreateRecipeRequest,
} from '$lib/types/recipe';

interface RecipeStore {
	recipes: RecipeResponse[];
	total: number;
	currentRecipe: RecipeResponse | null;
	versions: RecipeVersionResponse[];
	loading: boolean;
	error: string | null;
}

function createRecipeStore() {
	const { subscribe, update } = writable<RecipeStore>({
		recipes: [],
		total: 0,
		currentRecipe: null,
		versions: [],
		loading: false,
		error: null,
	});

	return {
		subscribe,

		async loadRecipes() {
			update((state) => ({ ...state, loading: true, error: null }));
			try {
				const result = await recipeApi.listRecipes({ limit: 100 });
				update((state) => ({
					...state,
					recipes: result.items || [],
					total: result.total,
					loading: false,
				}));
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to load recipes';
				update((state) => ({ ...state, error: message, loading: false }));
			}
		},

		async loadRecipe(id: string) {
			update((state) => ({ ...state, loading: true, error: null }));
			try {
				const recipe = await recipeApi.getRecipe(id);
				update((state) => ({ ...state, currentRecipe: recipe, loading: false }));
				return recipe;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to load recipe';
				update((state) => ({ ...state, error: message, loading: false }));
				throw error;
			}
		},

		async loadVersions(recipeId: string) {
			try {
				const versions = await recipeApi.listVersions(recipeId);
				update((state) => ({ ...state, versions }));
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to load versions';
				update((state) => ({ ...state, error: message }));
			}
		},

		async createRecipe(data: CreateRecipeRequest) {
			update((state) => ({ ...state, loading: true, error: null }));
			try {
				const recipe = await recipeApi.createRecipe(data);
				update((state) => ({
					...state,
					recipes: [...state.recipes, recipe],
					loading: false,
				}));
				return recipe;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to create recipe';
				update((state) => ({ ...state, error: message, loading: false }));
				throw error;
			}
		},

		async publishVersion(recipeId: string) {
			try {
				await recipeApi.publishVersion(recipeId);
				// Reload recipe to get updated state
				const recipe = await recipeApi.getRecipe(recipeId);
				update((state) => ({ ...state, currentRecipe: recipe }));
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to publish version';
				update((state) => ({ ...state, error: message }));
				throw error;
			}
		},

		async createNewVersion(recipeId: string, changeSummary?: string) {
			try {
				await recipeApi.createVersion(recipeId, { changeSummary });
				// Reload recipe to get updated state with new draft
				const recipe = await recipeApi.getRecipe(recipeId);
				update((state) => ({ ...state, currentRecipe: recipe }));
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to create new version';
				update((state) => ({ ...state, error: message }));
				throw error;
			}
		},

		clearError() {
			update((state) => ({ ...state, error: null }));
		},

		clearCurrent() {
			update((state) => ({ ...state, currentRecipe: null, versions: [] }));
		},
	};
}

export const recipeStore = createRecipeStore();
