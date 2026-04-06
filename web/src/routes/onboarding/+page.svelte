<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { spaceApi } from '$lib/api/space';
	import { setActiveSpaceID } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';

	let user = $derived($page.data.user);
	let isAdmin = $derived(user?.role === 'admin');

	let spaceName = $state('');
	let selectedTemplate = $state('blank');
	let isLoading = $state(false);
	let error = $state('');

	const templates = [
		{
			id: 'blank',
			name: 'Blank Space',
			description: 'Start from scratch with an empty workspace.',
			icon: 'M12 6v6m0 0v6m0-6h6m-6 0H6',
		},
		{
			id: 'woodworking_shop',
			name: 'Woodworking Shop',
			description: 'Pre-configured stations and recipes for a woodworking shop.',
			icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6',
		},
		{
			id: 'general_manufacturing',
			name: 'General Manufacturing',
			description: 'A general-purpose layout for small-batch manufacturing.',
			icon: 'M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z',
		},
	];

	async function handleCreateSpace(event: SubmitEvent) {
		event.preventDefault();

		if (!spaceName.trim()) {
			error = 'Please enter a space name';
			return;
		}

		isLoading = true;
		error = '';

		try {
			const space = await spaceApi.create({
				name: spaceName.trim(),
			});

			// Set the new space as the active space
			setActiveSpaceID(space.id);

			// Redirect to the new space
			goto(`/spaces/${space.id}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create space';
		} finally {
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>Get Started - Nori</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center bg-background py-12 px-4 sm:px-6 lg:px-8">
	<div class="max-w-lg w-full space-y-8">
		<div>
			<div class="flex justify-center">
				<div class="w-16 h-16 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-lg flex items-center justify-center">
					<svg class="w-10 h-10 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M13 10V3L4 14h7v7l9-11h-7z"
						/>
					</svg>
				</div>
			</div>
			<h1 class="mt-4 text-center text-4xl font-bold text-foreground">
				Nori
			</h1>

			{#if isAdmin}
				<h2 class="mt-6 text-center text-2xl font-extrabold text-foreground">
					Create your first space
				</h2>
				<p class="mt-2 text-center text-sm text-muted-foreground">
					A space is where your shop's work happens. Name it and pick a starting template.
				</p>
			{:else}
				<h2 class="mt-6 text-center text-2xl font-extrabold text-foreground">
					No spaces available
				</h2>
			{/if}
		</div>

		{#if isAdmin}
			<form class="mt-8 space-y-6" onsubmit={handleCreateSpace}>
				<div>
					<label for="spaceName" class="block text-sm font-medium text-foreground mb-1">
						Space name
					</label>
					<input
						id="spaceName"
						name="spaceName"
						type="text"
						required
						class="appearance-none relative block w-full px-3 py-2 border border-border placeholder-muted-foreground text-foreground bg-card rounded-md focus:outline-none focus:ring-ring focus:border-ring focus:z-10 sm:text-sm"
						placeholder="e.g. Main Workshop"
						bind:value={spaceName}
					/>
				</div>

				<fieldset>
					<legend class="block text-sm font-medium text-foreground mb-3">
						Choose a template
					</legend>
					<div class="space-y-3">
						{#each templates as template (template.id)}
							<button
								type="button"
								class="w-full text-left p-4 rounded-lg border-2 transition-all {selectedTemplate === template.id
									? 'border-primary bg-primary/5'
									: 'border-border bg-card hover:border-muted-foreground/50'}"
								onclick={() => (selectedTemplate = template.id)}
							>
								<div class="flex items-start gap-3">
									<div class="flex-shrink-0 w-10 h-10 rounded-md bg-primary/10 flex items-center justify-center">
										<svg class="w-5 h-5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d={template.icon}
											/>
										</svg>
									</div>
									<div>
										<div class="text-sm font-medium text-foreground">{template.name}</div>
										<div class="text-xs text-muted-foreground mt-0.5">{template.description}</div>
									</div>
									{#if selectedTemplate === template.id}
										<svg class="w-5 h-5 text-primary ml-auto flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M5 13l4 4L19 7"
											/>
										</svg>
									{/if}
								</div>
							</button>
						{/each}
					</div>
				</fieldset>

				{#if error}
					<div class="rounded-md bg-destructive/10 p-4">
						<div class="text-sm text-destructive">{error}</div>
					</div>
				{/if}

				<div>
					<Button
						type="submit"
						disabled={isLoading}
						class="w-full"
					>
						{#if isLoading}
							Creating space...
						{:else}
							Create space
						{/if}
					</Button>
				</div>
			</form>
		{:else}
			<div class="mt-8 text-center">
				<div class="mx-auto w-16 h-16 rounded-full bg-muted flex items-center justify-center mb-4">
					<svg class="w-8 h-8 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
						/>
					</svg>
				</div>
				<p class="text-muted-foreground">
					No spaces are available yet. Contact your admin to get started.
				</p>
			</div>
		{/if}
	</div>
</div>
