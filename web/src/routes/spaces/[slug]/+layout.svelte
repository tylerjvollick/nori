<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { spaceStore } from '$lib/stores/space';
	import { spaceMembersStore } from '$lib/stores/spaceMembers';
	import { stationApi } from '$lib/api/station';
	import type { StationResponse } from '$lib/types/station';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import * as Select from '$lib/components/ui/select';
	import { LayoutGrid, List, Funnel, X, Keyboard, Settings2 } from '@lucide/svelte';
	import { isEditableTarget, getToast, clearToast } from '$lib/utils/keyboard.svelte';
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

	// ---- View mode ----
	type ViewMode = 'board' | 'list';
	const VIEW_MODES: { value: ViewMode; label: string; icon: typeof LayoutGrid }[] = [
		{ value: 'board', label: 'Board', icon: LayoutGrid },
		{ value: 'list', label: 'List', icon: List },
	];

	let currentView = $derived<ViewMode>(
		(($page.url.searchParams.get('view') as ViewMode) || 'board') as ViewMode,
	);

	// ---- Filters ----
	let stations = $state<StationResponse[]>([]);
	let stationsLoaded = $state(false);

	// Read filters from URL search params
	let stationFilter = $derived($page.url.searchParams.get('station') || '');
	let statusFilter = $derived($page.url.searchParams.get('status') || '');
	let priorityFilter = $derived($page.url.searchParams.get('priority') || '');

	const STATUS_OPTIONS = [
		{ value: 'open', label: 'Open' },
		{ value: 'active', label: 'Active' },
		{ value: 'paused', label: 'Paused' },
		{ value: 'done', label: 'Done' },
		{ value: 'skipped', label: 'Skipped' },
	];

	const PRIORITY_OPTIONS = [
		{ value: '0', label: 'P0 - Critical' },
		{ value: '1', label: 'P1 - High' },
		{ value: '2', label: 'P2 - Medium' },
		{ value: '3', label: 'P3 - Low' },
		{ value: '4', label: 'P4 - Backlog' },
	];

	/** Count of active filters (non-empty filter params). */
	let activeFilterCount = $derived(
		[stationFilter, statusFilter, priorityFilter].filter(Boolean).length,
	);

	let showFilters = $state(false);

	// ---- Help overlay ----
	let showHelp = $state(false);

	// ---- Toast display ----
	let toastMessage = $derived(getToast());

	// ---- Check if we're on task detail page ----
	let isTaskDetail = $derived(!!$page.params.taskId);

	// ---- URL helpers ----

	/** Update a URL search param and navigate. Removes param if value is empty. */
	function setParam(key: string, value: string): void {
		const url = new URL($page.url);
		if (value) {
			url.searchParams.set(key, value);
		} else {
			url.searchParams.delete(key);
		}
		goto(url.toString(), { replaceState: true, keepFocus: true, noScroll: true });
	}

	function setView(mode: ViewMode): void {
		setParam('view', mode === 'board' ? '' : mode);
	}

	function clearAllFilters(): void {
		const url = new URL($page.url);
		url.searchParams.delete('station');
		url.searchParams.delete('status');
		url.searchParams.delete('priority');
		goto(url.toString(), { replaceState: true, keepFocus: true, noScroll: true });
	}

	// ---- Keyboard shortcuts ----

	function handleKeydown(e: KeyboardEvent): void {
		// ? works even in help overlay — toggle it
		if (e.key === '?' && !isEditableTarget(e)) {
			e.preventDefault();
			showHelp = !showHelp;
			return;
		}

		// Esc: close help overlay, close filter bar, or blur focused element
		if (e.key === 'Escape') {
			if (showHelp) {
				showHelp = false;
				return;
			}
			if (showFilters) {
				showFilters = false;
				return;
			}
			// Blur any focused element so view shortcuts can fire
			if (document.activeElement instanceof HTMLElement) {
				document.activeElement.blur();
			}
			return;
		}

		// Don't fire other shortcuts when help overlay is open
		if (showHelp) return;

		// Don't trigger shortcuts when typing in inputs, selects, textareas
		if (isEditableTarget(e)) return;

		// Don't fire view shortcuts on task detail pages (they have their own)
		if (isTaskDetail) return;

		switch (e.key) {
			case 'b':
				setView('board');
				break;
			// Note: 'l' for list view is handled here but also means "move right"
			// in board view. Board view handles 'l' first; if board doesn't
			// consume it, it falls through to here. We use 'L' (shift+l) as
			// a fallback, but the spec says lowercase 'l'. The board view
			// will stopPropagation when it handles 'l' for column navigation.
			// Layout only handles view switching, views handle their own keys.
			case 'l':
				// Only switch view if NOT on the board view (board uses 'l' for navigation)
				if (currentView !== 'board') {
					setView('list');
				}
				break;
			case '/':
				e.preventDefault();
				showFilters = true;
				// Focus the first select trigger in the filter bar after it renders
				requestAnimationFrame(() => {
					const firstTrigger = document.querySelector(
						'[data-slot="select-trigger"]',
					) as HTMLElement;
					firstTrigger?.focus();
				});
				break;
		}
	}

	// ---- Data loading ----

	// Load stations when the current space becomes available
	$effect(() => {
		const space = currentSpace;
		if (space && !stationsLoaded) {
			stationApi.listStations(space.id)
				.then((s) => { stations = s; })
				.catch(() => {})
				.finally(() => { stationsLoaded = true; });
		}
	});
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="flex h-full flex-col overflow-hidden">
	<!-- Flow toolbar (hidden on task detail pages — they have their own nav) -->
	{#if !isTaskDetail}
		<div class="flex-shrink-0 border-b bg-background px-4 py-2">
			<div class="flex items-center justify-between gap-4">
				<!-- Space name + slug badge -->
				<div class="flex items-center gap-2 min-w-0">
					{#if currentSpace}
						<h2 class="text-sm font-semibold text-foreground truncate">{currentSpace.name}</h2>
						<span class="shrink-0 rounded bg-muted px-1.5 py-0.5 text-[10px] font-mono font-medium text-muted-foreground">
							{currentSpace.slug}
						</span>
					{/if}
				</div>

				<!-- View mode switcher (segmented control) -->
				<div class="flex items-center rounded-lg border bg-muted/50 p-0.5">
					{#each VIEW_MODES as mode (mode.value)}
						<Button
							variant={currentView === mode.value ? 'secondary' : 'ghost'}
							size="sm"
							class="gap-1.5 rounded-md px-3 {currentView === mode.value
								? 'bg-background shadow-sm'
								: 'hover:bg-transparent hover:text-foreground'}"
							onclick={() => setView(mode.value)}
						>
							<mode.icon class="size-4" />
							{mode.label}
						</Button>
					{/each}
				</div>

				<!-- Filter toggle -->
				<div class="flex items-center gap-2">
					<Button
						variant={showFilters ? 'secondary' : 'outline'}
						size="sm"
						class="gap-1.5"
						onclick={() => (showFilters = !showFilters)}
					>
						<Funnel class="size-4" />
						Filters
						{#if activeFilterCount > 0}
							<Badge variant="default" class="ml-0.5 h-5 min-w-5 px-1.5 text-[10px]">
								{activeFilterCount}
							</Badge>
						{/if}
					</Button>
					{#if activeFilterCount > 0}
						<Button variant="ghost" size="sm" class="gap-1 text-muted-foreground" onclick={clearAllFilters}>
							<X class="size-3.5" />
							Clear
						</Button>
					{/if}
				</div>
			</div>

			<!-- Filter bar (collapsible) -->
			{#if showFilters}
				<div class="mt-2 flex flex-wrap items-center gap-3 border-t pt-2">
					<!-- Station filter -->
					<div class="flex items-center gap-1.5">
						<span class="text-xs font-medium text-muted-foreground">Station</span>
						<Select.Root
							type="single"
							value={stationFilter}
							onValueChange={(v) => setParam('station', v ?? '')}
						>
							<Select.Trigger size="sm" class="h-7 min-w-[120px] text-xs">
								{#if stationFilter}
									{stations.find((s) => s.id === stationFilter)?.name ?? 'Station'}
								{:else}
									All
								{/if}
							</Select.Trigger>
							<Select.Content>
								<Select.Item value="" label="All">All stations</Select.Item>
								{#each stations as station (station.id)}
									<Select.Item value={station.id} label={station.name}>
										{station.name}
									</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					</div>

					<!-- Status filter -->
					<div class="flex items-center gap-1.5">
						<span class="text-xs font-medium text-muted-foreground">Status</span>
						<Select.Root
							type="single"
							value={statusFilter}
							onValueChange={(v) => setParam('status', v ?? '')}
						>
							<Select.Trigger size="sm" class="h-7 min-w-[100px] text-xs">
								{#if statusFilter}
									{STATUS_OPTIONS.find((s) => s.value === statusFilter)?.label ?? statusFilter}
								{:else}
									All
								{/if}
							</Select.Trigger>
							<Select.Content>
								<Select.Item value="" label="All">All statuses</Select.Item>
								{#each STATUS_OPTIONS as opt (opt.value)}
									<Select.Item value={opt.value} label={opt.label}>
										{opt.label}
									</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					</div>

					<!-- Priority filter -->
					<div class="flex items-center gap-1.5">
						<span class="text-xs font-medium text-muted-foreground">Priority</span>
						<Select.Root
							type="single"
							value={priorityFilter}
							onValueChange={(v) => setParam('priority', v ?? '')}
						>
							<Select.Trigger size="sm" class="h-7 min-w-[120px] text-xs">
								{#if priorityFilter}
									{PRIORITY_OPTIONS.find((p) => p.value === priorityFilter)?.label ?? `P${priorityFilter}`}
								{:else}
									All
								{/if}
							</Select.Trigger>
							<Select.Content>
								<Select.Item value="" label="All">All priorities</Select.Item>
								{#each PRIORITY_OPTIONS as opt (opt.value)}
									<Select.Item value={opt.value} label={opt.label}>
										{opt.label}
									</Select.Item>
								{/each}
							</Select.Content>
						</Select.Root>
					</div>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Child page content -->
	<!-- Task detail pages manage their own scrolling; overflow-auto here breaks
	     SvelteFlow's height chain (it needs a fixed-height ancestor, not a scrollable one). -->
	<div class="flex flex-col flex-1 {isTaskDetail ? 'overflow-hidden' : 'overflow-auto'}">
		{@render children()}
	</div>

	<!-- Toast notification (keyboard action confirmations) -->
	{#if toastMessage}
		<div class="fixed bottom-4 left-1/2 -translate-x-1/2 z-50 animate-in fade-in slide-in-from-bottom-2 duration-200">
			<div class="rounded-lg border bg-background px-4 py-2 shadow-lg text-sm text-foreground">
				{toastMessage}
			</div>
		</div>
	{/if}

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
