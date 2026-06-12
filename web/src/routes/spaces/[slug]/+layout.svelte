<script lang="ts">
	import { page } from '$app/stores';
	import { spaceMembersStore } from '$lib/stores/spaceMembers';
	import { Button } from '$lib/components/ui/button';
	import * as Breadcrumb from '$lib/components/ui/breadcrumb';
	import { Keyboard, Share2, Settings } from '@lucide/svelte';
	import { isEditableTarget } from '$lib/utils/keyboard.svelte';
	import KeyboardHelp from '$lib/components/flow/KeyboardHelp.svelte';
	import { browser } from '$app/environment';

	let { children } = $props();

	// ---- Space context (resolved server-side in +layout.server.ts) ----
	let slug = $derived($page.params.slug);
	let space = $derived($page.data.space!);

	// Load space members when the space is available
	$effect(() => {
		if (space) {
			spaceMembersStore.loadMembers(space.id);
		}
	});

	// ---- Help overlay ----
	let showHelp = $state(false);

	// ---- Check if we're on a detail page (task/job or recipe) ----
	let isTaskDetail = $derived(!!$page.params.taskId);
	let isRecipeDetail = $derived(!!$page.params.id);
	let isDetailView = $derived(isTaskDetail || isRecipeDetail);

	// ---- Tab definitions ----
	const TABS = [
		{ value: 'jobs', label: 'Jobs' },
		{ value: 'tasks', label: 'Tasks' },
		{ value: 'recipes', label: 'Recipes' },
		{ value: 'schedule', label: 'Schedule' },
		{ value: 'insights', label: 'Insights' },
	] as const;

	// Raw path segment (empty on a bare /spaces/{slug} URL before the redirect).
	let rawTab = $derived($page.url.pathname.split('/')[3] ?? '');
	// Highlight a concrete tab at all times — default to Jobs when the URL has no
	// tab segment yet (bare space URL) so the user never sees "no tab selected".
	let activeTab = $derived(TABS.some((t) => t.value === rawTab) ? rawTab : 'jobs');

	// Persist only a real, concrete tab segment (never the defaulted highlight),
	// so landing on a bare space URL can't clobber the stored last-visited tab.
	$effect(() => {
		if (browser && slug && TABS.some((t) => t.value === rawTab)) {
			localStorage.setItem(`lastVisitedTab:${slug}`, rawTab);
		}
	});

	// Persist the most recently opened space so `/` can forward back to it
	$effect(() => {
		if (browser && slug) {
			localStorage.setItem('lastVisitedSpace', slug);
		}
	});

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
	{#if !isDetailView}
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
							<Breadcrumb.Page>{space.name}</Breadcrumb.Page>
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
	<div class="flex flex-col flex-1 {isDetailView ? 'overflow-hidden' : 'overflow-auto'}">
		{@render children()}
	</div>

	<!-- Keyboard shortcut hint (hidden on task detail pages) -->
	{#if !isDetailView}
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
