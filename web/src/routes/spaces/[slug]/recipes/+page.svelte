<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { recipeStore } from '$lib/stores/recipe';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Table from '$lib/components/ui/table';
	import * as Alert from '$lib/components/ui/alert';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { CircleAlert, ChevronUp, ChevronDown, ChevronsUpDown } from '@lucide/svelte';
	import type { RecipeResponse } from '$lib/types/recipe';

	let space = $derived($page.data.space!);
	let spaceId = $derived(space.id);

	let searchQuery = $state('');
	let showCreateDialog = $state(false);
	let newRecipeName = $state('');
	let newRecipeDescription = $state('');
	let isCreating = $state(false);
	let createError = $state('');

	type SortColumn = 'name' | 'status' | 'version' | 'updatedAt';
	type SortDir = 'asc' | 'desc';
	let sortColumn = $state<SortColumn>('name');
	let sortDir = $state<SortDir>('asc');

	$effect(() => {
		if (spaceId) recipeStore.loadRecipes(spaceId);
	});

	function getStatus(recipe: RecipeResponse): string {
		if (!recipe.currentVersion) return 'draft';
		return recipe.currentVersion.status;
	}

	const filteredAndSorted = $derived(() => {
		let list = searchQuery.trim()
			? $recipeStore.recipes.filter(
					(r) =>
						r.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
						r.description?.toLowerCase().includes(searchQuery.toLowerCase())
				)
			: $recipeStore.recipes;

		return [...list].sort((a, b) => {
			let cmp = 0;
			if (sortColumn === 'name') {
				cmp = a.name.localeCompare(b.name);
			} else if (sortColumn === 'status') {
				cmp = getStatus(a).localeCompare(getStatus(b));
			} else if (sortColumn === 'version') {
				const av = a.currentVersion?.versionNumber ?? 0;
				const bv = b.currentVersion?.versionNumber ?? 0;
				cmp = av - bv;
			} else if (sortColumn === 'updatedAt') {
				cmp = new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime();
			}
			return sortDir === 'asc' ? cmp : -cmp;
		});
	});

	function toggleSort(col: SortColumn) {
		if (sortColumn === col) {
			sortDir = sortDir === 'asc' ? 'desc' : 'asc';
		} else {
			sortColumn = col;
			sortDir = 'asc';
		}
	}

	function formatDate(dateString: string): string {
		return new Date(dateString).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
		});
	}

	function truncate(text: string | null | undefined, max = 80): string {
		if (!text) return '';
		return text.length > max ? text.slice(0, max) + '…' : text;
	}

	async function handleCreate() {
		if (!newRecipeName.trim() || isCreating) return;
		isCreating = true;
		createError = '';
		try {
			const recipe = await recipeStore.createRecipe(spaceId, {
				name: newRecipeName.trim(),
				description: newRecipeDescription.trim() || undefined,
			});
			showCreateDialog = false;
			newRecipeName = '';
			newRecipeDescription = '';
			goto(`/spaces/${$page.params.slug}/recipes/${recipe.id}`);
		} catch (error) {
			createError = error instanceof Error ? error.message : 'Failed to create recipe';
		} finally {
			isCreating = false;
		}
	}
</script>

<svelte:head>
	<title>{space.name} – Recipes - Nori</title>
</svelte:head>

<div class="container mx-auto px-4 py-8">
	<!-- Header -->
	<div class="flex justify-between items-center mb-6">
		<div>
			<h1 class="text-2xl font-bold text-foreground">Recipes</h1>
			<p class="text-muted-foreground mt-1 text-sm">Process templates for this space</p>
		</div>
		<Button onclick={() => (showCreateDialog = true)}>New Recipe</Button>
	</div>

	<!-- Search Bar -->
	<div class="mb-4">
		<Input
			type="text"
			bind:value={searchQuery}
			placeholder="Search recipes by name or description..."
			class="h-10"
		/>
	</div>

	<!-- Table -->
	{#if $recipeStore.loading}
		<Table.Root>
			<Table.Header>
				<Table.Row>
					<Table.Head>Name</Table.Head>
					<Table.Head>Description</Table.Head>
					<Table.Head>Status</Table.Head>
					<Table.Head>Version</Table.Head>
					<Table.Head>Updated</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each Array(5) as _}
					<Table.Row>
						<Table.Cell><Skeleton class="h-4 w-40" /></Table.Cell>
						<Table.Cell><Skeleton class="h-4 w-64" /></Table.Cell>
						<Table.Cell><Skeleton class="h-5 w-16" /></Table.Cell>
						<Table.Cell><Skeleton class="h-4 w-8" /></Table.Cell>
						<Table.Cell><Skeleton class="h-4 w-24" /></Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
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
	{:else if filteredAndSorted().length === 0}
		<div class="text-center py-12">
			<p class="text-muted-foreground">No recipes match your search.</p>
		</div>
	{:else}
		<Table.Root>
			<Table.Header>
				<Table.Row>
					<Table.Head>
						<button
							class="flex items-center gap-1 hover:text-foreground transition-colors"
							onclick={() => toggleSort('name')}
						>
							Name
							{#if sortColumn === 'name'}
								{#if sortDir === 'asc'}<ChevronUp class="size-3" />{:else}<ChevronDown class="size-3" />{/if}
							{:else}
								<ChevronsUpDown class="size-3 opacity-40" />
							{/if}
						</button>
					</Table.Head>
					<Table.Head>Description</Table.Head>
					<Table.Head>
						<button
							class="flex items-center gap-1 hover:text-foreground transition-colors"
							onclick={() => toggleSort('status')}
						>
							Status
							{#if sortColumn === 'status'}
								{#if sortDir === 'asc'}<ChevronUp class="size-3" />{:else}<ChevronDown class="size-3" />{/if}
							{:else}
								<ChevronsUpDown class="size-3 opacity-40" />
							{/if}
						</button>
					</Table.Head>
					<Table.Head>
						<button
							class="flex items-center gap-1 hover:text-foreground transition-colors"
							onclick={() => toggleSort('version')}
						>
							Version
							{#if sortColumn === 'version'}
								{#if sortDir === 'asc'}<ChevronUp class="size-3" />{:else}<ChevronDown class="size-3" />{/if}
							{:else}
								<ChevronsUpDown class="size-3 opacity-40" />
							{/if}
						</button>
					</Table.Head>
					<Table.Head>
						<button
							class="flex items-center gap-1 hover:text-foreground transition-colors"
							onclick={() => toggleSort('updatedAt')}
						>
							Updated
							{#if sortColumn === 'updatedAt'}
								{#if sortDir === 'asc'}<ChevronUp class="size-3" />{:else}<ChevronDown class="size-3" />{/if}
							{:else}
								<ChevronsUpDown class="size-3 opacity-40" />
							{/if}
						</button>
					</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each filteredAndSorted() as recipe (recipe.id)}
					{@const status = getStatus(recipe)}
					<Table.Row
						class="cursor-pointer hover:bg-muted/50"
						onclick={() => goto(`/spaces/${$page.params.slug}/recipes/${recipe.id}`)}
					>
						<Table.Cell class="font-medium">
							{recipe.name}
							{#if !recipe.isActive}
								<Badge variant="outline" class="ml-2 text-xs">Archived</Badge>
							{/if}
						</Table.Cell>
						<Table.Cell class="text-muted-foreground max-w-xs">
							{#if recipe.description}
								{truncate(recipe.description)}
							{:else}
								<span class="italic opacity-50">No description</span>
							{/if}
						</Table.Cell>
						<Table.Cell>
							{#if status === 'published'}
								<Badge variant="secondary" class="bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 border-green-200 dark:border-green-800">
									Published
								</Badge>
							{:else}
								<Badge variant="outline" class="bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 border-yellow-200 dark:border-yellow-800">
									Draft
								</Badge>
							{/if}
						</Table.Cell>
						<Table.Cell class="text-muted-foreground">
							{#if recipe.currentVersion}
								v{recipe.currentVersion.versionNumber}
							{:else}
								—
							{/if}
						</Table.Cell>
						<Table.Cell class="text-muted-foreground whitespace-nowrap">
							{formatDate(recipe.updatedAt)}
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>

		<div class="mt-4 text-sm text-muted-foreground">
			Showing {filteredAndSorted().length} of {$recipeStore.total} recipe{$recipeStore.total === 1 ? '' : 's'}
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
