<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { spaceStore } from '$lib/stores/space';
	import { stationApi } from '$lib/api/station';
	import type { StationResponse } from '$lib/types/station';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import * as Select from '$lib/components/ui/select';
	import { LayoutGrid, List, Funnel, X } from '@lucide/svelte';
	import { isEditableTarget, getToast, clearToast } from '$lib/utils/keyboard.svelte';
	import BoardView from '$lib/components/flow/BoardView.svelte';
	import ListView from '$lib/components/flow/ListView.svelte';

	let currentSpace = $derived($spaceStore.currentSpace);

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

	let activeFilterCount = $derived(
		[stationFilter, statusFilter, priorityFilter].filter(Boolean).length,
	);

	let showFilters = $state(false);

	// ---- Toast display ----
	let toastMessage = $derived(getToast());

	// ---- URL helpers ----

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
		if (isEditableTarget(e)) return;

		switch (e.key) {
			case 'b':
				setView('board');
				break;
			case 'l':
				if (currentView !== 'board') {
					setView('list');
				}
				break;
			case '/':
				e.preventDefault();
				showFilters = true;
				requestAnimationFrame(() => {
					const firstTrigger = document.querySelector(
						'[data-slot="select-trigger"]',
					) as HTMLElement;
					firstTrigger?.focus();
				});
				break;
			case 'Escape':
				if (showFilters) {
					showFilters = false;
				}
				break;
		}
	}

	// ---- Data loading ----

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

<svelte:head>
	<title>{currentSpace?.name ?? 'Space'} – Tasks - Nori</title>
</svelte:head>

<svelte:window onkeydown={handleKeydown} />

<div class="flex h-full flex-col overflow-hidden">
	<!-- Tasks toolbar -->
	<div class="flex-shrink-0 border-b bg-background px-4 py-2">
		<div class="flex items-center justify-between gap-4">
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

	<!-- Board or list view -->
	<div class="flex-1 overflow-auto">
		{#if currentView === 'board'}
			<BoardView />
		{:else if currentView === 'list'}
			<ListView />
		{/if}
	</div>

	<!-- Toast notification (keyboard action confirmations) -->
	{#if toastMessage}
		<div class="fixed bottom-4 left-1/2 -translate-x-1/2 z-50 animate-in fade-in slide-in-from-bottom-2 duration-200">
			<div class="rounded-lg border bg-background px-4 py-2 shadow-lg text-sm text-foreground">
				{toastMessage}
			</div>
		</div>
	{/if}
</div>
