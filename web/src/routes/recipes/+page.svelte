<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { recipeStore } from '$lib/stores/recipe';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Card from '$lib/components/ui/card';
	import * as Alert from '$lib/components/ui/alert';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { CircleAlert } from '@lucide/svelte';

	let searchQuery = $state('');
	let showCreateDialog = $state(false);
	let newRecipeName = $state('');
	let newRecipeDescription = $state('');
	let isCreating = $state(false);
	let createError = $state('');

	onMount(() => {
		recipeStore.loadRecipes();
	});

	const filteredRecipes = $derived(
		searchQuery.trim()
			? $recipeStore.recipes.filter(
					(r) =>
						r.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
						r.description?.toLowerCase().includes(searchQuery.toLowerCase())
				)
			: $recipeStore.recipes
	);

	function formatDate(dateString: string): string {
		return new Date(dateString).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
		});
	}

	function getVersionStatusBadge(recipe: (typeof $recipeStore.recipes)[0]) {
		if (!recipe.currentVersion) return 'draft';
		return recipe.currentVersion.status;
	}

	async function handleCreate() {
		if (!newRecipeName.trim() || isCreating) return;
		isCreating = true;
		createError = '';
		try {
			const recipe = await recipeStore.createRecipe({
				name: newRecipeName.trim(),
				description: newRecipeDescription.trim() || undefined,
			});
			showCreateDialog = false;
			newRecipeName = '';
			newRecipeDescription = '';
			goto(`/recipes/${recipe.id}`);
		} catch (error) {
			createError = error instanceof Error ? error.message : 'Failed to create recipe';
		} finally {
			isCreating = false;
		}
	}
</script>

<div class="container mx-auto px-4 py-8">
	<!-- Header -->
	<div class="flex justify-between items-center mb-8">
		<div>
			<h1 class="text-3xl font-bold text-foreground">Recipes</h1>
			<p class="text-muted-foreground mt-1">Browse and manage your recipe templates</p>
		</div>
		<Button onclick={() => (showCreateDialog = true)}>New Recipe</Button>
	</div>

	<!-- Search Bar -->
	<div class="mb-6">
		<Input
			type="text"
			bind:value={searchQuery}
			placeholder="Search recipes by name or description..."
			class="h-10"
		/>
	</div>

	<!-- Recipe Grid -->
	{#if $recipeStore.loading}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
			{#each Array(6) as _}
				<Card.Root>
					<Card.Header>
						<div class="flex items-start justify-between">
							<Skeleton class="h-6 w-3/4" />
							<Skeleton class="h-5 w-12" />
						</div>
					</Card.Header>
					<Card.Content class="px-0 py-0">
						<div class="px-4">
							<Skeleton class="h-4 w-full mb-2" />
							<Skeleton class="h-4 w-2/3" />
						</div>
					</Card.Content>
					<Card.Footer class="px-4 py-3">
						<div class="flex items-center justify-between w-full">
							<Skeleton class="h-3 w-16" />
							<Skeleton class="h-3 w-24" />
						</div>
					</Card.Footer>
				</Card.Root>
			{/each}
		</div>
	{:else if $recipeStore.error}
		<Alert.Root variant="destructive">
			<CircleAlert />
			<Alert.Title>Error loading recipes</Alert.Title>
			<Alert.Description>{$recipeStore.error}</Alert.Description>
		</Alert.Root>
	{:else if $recipeStore.recipes.length === 0}
		<div class="text-center py-12">
			<p class="text-muted-foreground mb-4">
				No recipes yet. Create your first recipe to get started.
			</p>
			<Button onclick={() => (showCreateDialog = true)}>Create Your First Recipe</Button>
		</div>
	{:else if filteredRecipes.length === 0}
		<div class="text-center py-12">
			<p class="text-muted-foreground">No recipes match your search.</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
			{#each filteredRecipes as recipe (recipe.id)}
				<a href="/recipes/{recipe.id}" class="block group">
					<Card.Root class="h-full transition-all hover:shadow-lg hover:ring-primary/50">
						<Card.Header>
							<div class="flex items-start justify-between">
								<Card.Title class="text-xl group-hover:text-primary transition-colors">
									{recipe.name}
								</Card.Title>
								<div class="flex gap-2 shrink-0 ml-2">
									{#if getVersionStatusBadge(recipe) === 'published'}
										<Badge variant="secondary" class="bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 border-green-200 dark:border-green-800">
											Published
										</Badge>
									{:else}
										<Badge variant="outline" class="bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 border-yellow-200 dark:border-yellow-800">
											Draft
										</Badge>
									{/if}
									{#if recipe.currentVersion}
										<Badge variant="secondary">
											v{recipe.currentVersion.versionNumber}
										</Badge>
									{/if}
								</div>
							</div>
						</Card.Header>
						<Card.Content class="px-0 py-0">
							<div class="px-4">
								{#if recipe.description}
									<p class="text-sm text-muted-foreground line-clamp-3">
										{recipe.description}
									</p>
								{:else}
									<p class="text-sm text-muted-foreground/70 italic">No description</p>
								{/if}
							</div>
						</Card.Content>
						<Card.Footer class="px-4 py-3">
							<div
								class="flex items-center justify-between w-full text-xs text-muted-foreground border-t border-border pt-3"
							>
								<span>
									{#if !recipe.isActive}
										<Badge variant="outline" class="text-xs">Archived</Badge>
									{/if}
								</span>
								<span>Updated {formatDate(recipe.updatedAt)}</span>
							</div>
						</Card.Footer>
					</Card.Root>
				</a>
			{/each}
		</div>

		<div class="mt-8 text-center text-sm text-muted-foreground">
			Showing {filteredRecipes.length} of {$recipeStore.total} recipe{$recipeStore.total === 1
				? ''
				: 's'}
		</div>
	{/if}
</div>

<!-- Create Recipe Dialog -->
<Dialog.Root
	bind:open={showCreateDialog}
	onOpenChange={(open) => {
		if (!open) {
			newRecipeName = '';
			newRecipeDescription = '';
			createError = '';
		}
	}}
>
	<Dialog.Content class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Create New Recipe</Dialog.Title>
			<Dialog.Description>Create a new recipe template to define repeatable workflows.</Dialog.Description>
		</Dialog.Header>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleCreate();
			}}
		>
			<div class="grid gap-4 py-2">
				<div class="grid gap-2">
					<Label for="recipe-name">Name</Label>
					<Input
						id="recipe-name"
						type="text"
						bind:value={newRecipeName}
						placeholder="e.g., Oak Dining Table, Custom Bookshelf"
						disabled={isCreating}
					/>
				</div>
				<div class="grid gap-2">
					<Label for="recipe-description">Description (optional)</Label>
					<Input
						id="recipe-description"
						type="text"
						bind:value={newRecipeDescription}
						placeholder="Brief description of this recipe"
						disabled={isCreating}
					/>
				</div>
				{#if createError}
					<Alert.Root variant="destructive">
						<CircleAlert class="size-4" />
						<Alert.Description>{createError}</Alert.Description>
					</Alert.Root>
				{/if}
			</div>
			<Dialog.Footer class="pt-2">
				<Button
					type="button"
					variant="outline"
					onclick={() => (showCreateDialog = false)}
					disabled={isCreating}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={!newRecipeName.trim() || isCreating}>
					{isCreating ? 'Creating...' : 'Create Recipe'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
