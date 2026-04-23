<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { spaceApi } from '$lib/api/space';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import * as Card from '$lib/components/ui/card';
	import * as Alert from '$lib/components/ui/alert';
	import * as RadioGroup from '$lib/components/ui/radio-group';
	import { CircleAlert, LoaderCircle, Lock, Check } from '@lucide/svelte';
	import Logo from '$lib/components/Logo.svelte';

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

			// Redirect to the new space
			goto(`/spaces/${space.slug}`);
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
	<Card.Root class="w-full max-w-lg">
		<Card.Header class="text-center">
			<Logo />
			{#if isAdmin}
				<Card.Title class="text-2xl font-extrabold">Create your first space</Card.Title>
				<Card.Description>
					A space is where your shop's work happens. Name it and pick a starting template.
				</Card.Description>
			{:else}
				<Card.Title class="text-2xl font-extrabold">No spaces available</Card.Title>
			{/if}
		</Card.Header>

		<Card.Content>
			{#if isAdmin}
				<form class="space-y-6" onsubmit={handleCreateSpace}>
					<div class="space-y-2">
						<Label for="spaceName">Space name</Label>
						<Input
							id="spaceName"
							name="spaceName"
							type="text"
							required
							placeholder="e.g. Main Workshop"
							bind:value={spaceName}
						/>
					</div>

					<fieldset class="space-y-3">
						<legend class="text-sm font-medium text-foreground">Choose a template</legend>
						<RadioGroup.Root bind:value={selectedTemplate} class="gap-3">
							{#each templates as template (template.id)}
								<label
									class="flex items-start gap-3 rounded-lg border-2 p-4 cursor-pointer transition-all {selectedTemplate === template.id
										? 'border-primary bg-primary/5'
										: 'border-border bg-card hover:border-muted-foreground/50'}"
								>
									<RadioGroup.Item value={template.id} class="mt-0.5" />
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
									<div class="flex-1">
										<div class="text-sm font-medium text-foreground">{template.name}</div>
										<div class="text-xs text-muted-foreground mt-0.5">{template.description}</div>
									</div>
									{#if selectedTemplate === template.id}
										<Check class="w-5 h-5 text-primary ml-auto flex-shrink-0" />
									{/if}
								</label>
							{/each}
						</RadioGroup.Root>
					</fieldset>

					{#if error}
						<Alert.Root variant="destructive">
							<CircleAlert />
							<Alert.Description>{error}</Alert.Description>
						</Alert.Root>
					{/if}

					<Card.Footer class="px-0 pb-0">
						<Button
							type="submit"
							disabled={isLoading}
							class="w-full"
						>
							{#if isLoading}
								<LoaderCircle class="animate-spin" />
								Creating space...
							{:else}
								Create space
							{/if}
						</Button>
					</Card.Footer>
				</form>
			{:else}
				<div class="text-center py-4">
					<div class="mx-auto w-16 h-16 rounded-full bg-muted flex items-center justify-center mb-4">
						<Lock class="w-8 h-8 text-muted-foreground" />
					</div>
					<p class="text-muted-foreground">
						No spaces are available yet. Contact your admin to get started.
					</p>
				</div>
			{/if}
		</Card.Content>
	</Card.Root>
</div>
