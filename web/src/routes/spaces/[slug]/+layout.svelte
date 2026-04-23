<script lang="ts">
	import { page } from '$app/stores';
	import { spaceStore } from '$lib/stores/space';
	import { spaceMembersStore } from '$lib/stores/spaceMembers';
	import { Button } from '$lib/components/ui/button';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb';
	import { Keyboard, Share2, Settings } from '@lucide/svelte';
	import { isEditableTarget } from '$lib/utils/keyboard.svelte';
	import KeyboardHelp from '$lib/components/flow/KeyboardHelp.svelte';

	let { children } = $props();

	// ---- Space context ----
	let slug = $derived($page.params.slug);
	let currentSpace = $derived($spaceStore.currentSpace);

	// Load space when slug changes
	$effect(() => {
		if (slug) {
			spaceStore.loadSpaceBySlug(slug);
		}
	});

	// Load space members when the current space is loaded
	$effect(() => {
		const space = currentSpace;
		if (space) {
			spaceMembersStore.loadMembers(space.id);
		}
	});

	// ---- Help overlay ----
	let showHelp = $state(false);

	// ---- Check if we're on task detail page ----
	let isTaskDetail = $derived(!!$page.params.taskId);

	// ---- Tab definitions ----
	const TABS = [
		{ value: 'jobs', label: 'Jobs' },
		{ value: 'tasks', label: 'Tasks' },
		{ value: 'recipes', label: 'Recipes' },
		{ value: 'schedule', label: 'Schedule' },
		{ value: 'insights', label: 'Insights' },
	] as const;

	let activeTab = $derived($page.url.pathname.split('/')[3] ?? '');

	// ---- Keyboard shortcuts ----

	function handleKeydown(e: KeyboardEvent): void {
		// ? works even in help overlay — toggle it
		if (e.key === '?' && !isEditableTarget(e)) {
			e.preventDefault();
			showHelp = !showHelp;
			return;
		}

		// Esc: close help overlay
		if (e.key === 'Escape' && showHelp) {
			showHelp = false;
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="flex h-full flex-col overflow-hidden">
	<!-- Space header + tabs (hidden on task detail pages — they have their own nav) -->
	{#if !isTaskDetail}
		<div class="flex-shrink-0 border-b bg-background px-4 pt-3 pb-0">
			<!-- Breadcrumb + actions -->
			<div class="flex items-center justify-between mb-2">
				<Breadcrumb.Root>
					<Breadcrumb.List>
						<Breadcrumb.Item>
							<Breadcrumb.Link href="/spaces">Spaces</Breadcrumb.Link>
						</Breadcrumb.Item>
						<Breadcrumb.Separator />
						<Breadcrumb.Item>
							<Breadcrumb.Page>
								{#if currentSpace}
									{currentSpace.name}
								{:else}
									<span class="text-muted-foreground">Loading...</span>
								{/if}
							</Breadcrumb.Page>
						</Breadcrumb.Item>
					</Breadcrumb.List>
				</Breadcrumb.Root>

				<!-- Space actions -->
				<div class="flex items-center gap-1.5">
					<Button variant="ghost" size="sm" class="gap-1.5 text-muted-foreground" disabled>
						<Share2 class="size-3.5" />
						Share
					</Button>
					<Button variant="ghost" size="sm" class="gap-1.5 text-muted-foreground" disabled>
						<Settings class="size-3.5" />
						Edit
					</Button>
				</div>
			</div>

			<!-- Tab bar -->
			<div class="flex gap-0" role="tablist">
				{#each TABS as tab (tab.value)}
					<a
						href="/spaces/{slug}/{tab.value}"
						role="tab"
						aria-selected={activeTab === tab.value}
						class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
							{activeTab === tab.value
								? 'border-foreground text-foreground'
								: 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted-foreground/40'}"
					>
						{tab.label}
					</a>
				{/each}
			</div>
		</div>
	{/if}

	<!-- Child page content -->
	<div class="flex flex-col flex-1 {isTaskDetail ? 'overflow-hidden' : 'overflow-auto'}">
		{@render children()}
	</div>

	<!-- Keyboard shortcut hint (hidden on task detail pages) -->
	{#if !isTaskDetail}
		<div class="fixed bottom-4 right-4 z-40">
			<Button
				variant="outline"
				size="sm"
				class="gap-1.5 opacity-60 hover:opacity-100 transition-opacity"
				onclick={() => (showHelp = true)}
			>
				<Keyboard class="size-3.5" />
				<kbd class="font-mono text-[10px]">?</kbd>
			</Button>
		</div>
	{/if}

	<!-- Keyboard help overlay -->
	<KeyboardHelp open={showHelp} onclose={() => (showHelp = false)} />
</div>
