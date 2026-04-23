<script lang="ts">
	import { onMount } from 'svelte';
	import { costApi, type CostSummary } from '$lib/api/cost';
	import { spaceStore } from '$lib/stores/space';
	import type { TaskTreeResponse } from '$lib/api/task';
	import * as Card from '$lib/components/ui/card';
	import * as Table from '$lib/components/ui/table';
	import * as Collapsible from '$lib/components/ui/collapsible';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Separator } from '$lib/components/ui/separator';
	import {
		DollarSign,
		Clock,
		TrendingUp,
		TrendingDown,
		Minus,
		ChevronDown,
		ChevronRight,
		ArrowUpDown,
	} from '@lucide/svelte';
	import { formatDuration } from '$lib/utils/time';

	interface Props {
		jobId: string;
		tree: TaskTreeResponse;
		stationMap: Map<string, string>;
	}

	let { jobId, tree, stationMap }: Props = $props();

	let spaceId = $derived($spaceStore.currentSpace?.id ?? '');

	let costSummary = $state<CostSummary | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	// Task breakdown state
	let taskBreakdownOpen = $state(false);
	let sortBy = $state<'variance' | 'title' | 'station'>('variance');
	let sortAsc = $state(false);

	// Parse decimal string to number for display
	function toNum(val: string): number {
		return parseFloat(val) || 0;
	}

	function formatCurrency(val: string): string {
		const num = toNum(val);
		return num.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
	}

	function formatHours(val: string): string {
		const num = toNum(val);
		return num.toLocaleString(undefined, { minimumFractionDigits: 1, maximumFractionDigits: 1 });
	}

	function hoursToSeconds(hours: string): number {
		return Math.round(toNum(hours) * 3600);
	}

	// Flatten tree into task list (excluding root job)
	interface TaskCostRow {
		id: string;
		title: string;
		stationId: string | null | undefined;
		stationName: string;
		estimatedSecs: number;
		actualSecs: number;
		varianceSecs: number;
	}

	function flattenTasks(node: TaskTreeResponse): TaskCostRow[] {
		const rows: TaskCostRow[] = [];
		function walk(n: TaskTreeResponse): void {
			// Skip root job, only include children
			if (n.id !== tree.id) {
				const est = n.estimatedTimeSeconds ?? 0;
				const act = n.actualTimeSeconds;
				rows.push({
					id: n.id,
					title: n.title,
					stationId: n.stationId,
					stationName: n.stationId ? (stationMap.get(n.stationId) ?? n.stationId.slice(0, 8)) : '--',
					estimatedSecs: est,
					actualSecs: act,
					varianceSecs: act - est,
				});
			}
			if (n.children) {
				for (const child of n.children) {
					walk(child);
				}
			}
		}
		walk(node);
		return rows;
	}

	let taskRows = $derived.by((): TaskCostRow[] => {
		const rows = flattenTasks(tree);
		rows.sort((a, b) => {
			let cmp = 0;
			switch (sortBy) {
				case 'variance':
					cmp = Math.abs(b.varianceSecs) - Math.abs(a.varianceSecs);
					break;
				case 'title':
					cmp = a.title.localeCompare(b.title);
					break;
				case 'station':
					cmp = a.stationName.localeCompare(b.stationName);
					break;
			}
			return sortAsc ? -cmp : cmp;
		});
		return rows;
	});

	function toggleSort(col: 'variance' | 'title' | 'station'): void {
		if (sortBy === col) {
			sortAsc = !sortAsc;
		} else {
			sortBy = col;
			sortAsc = false;
		}
	}

	// Variance formatting helpers
	function varianceColor(val: string | number): string {
		const num = typeof val === 'string' ? toNum(val) : val;
		if (num > 0) return 'text-red-500';
		if (num < 0) return 'text-green-500';
		return 'text-muted-foreground';
	}

	function varianceBg(val: string | number): string {
		const num = typeof val === 'string' ? toNum(val) : val;
		if (num > 0) return 'bg-red-500/10';
		if (num < 0) return 'bg-green-500/10';
		return 'bg-muted';
	}

	function varianceSign(val: string | number): string {
		const num = typeof val === 'string' ? toNum(val) : val;
		if (num > 0) return '+';
		return '';
	}

	onMount(async () => {
		try {
			costSummary = await costApi.getJobCostSummary(spaceId, jobId);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load cost data';
		} finally {
			isLoading = false;
		}
	});
</script>

<div class="p-4 sm:p-6 space-y-6 max-w-5xl mx-auto overflow-y-auto">
	{#if isLoading}
		<div class="space-y-4">
			<Skeleton class="h-8 w-48" />
			<div class="grid grid-cols-3 gap-4">
				{#each Array(3) as _}
					<Skeleton class="h-32 rounded-lg" />
				{/each}
			</div>
			<Skeleton class="h-48 rounded-lg" />
		</div>

	{:else if error}
		<Card.Root>
			<Card.Content class="p-6 text-center">
				<DollarSign class="w-10 h-10 text-muted-foreground mx-auto mb-3" />
				<p class="text-sm text-muted-foreground">{error}</p>
			</Card.Content>
		</Card.Root>

	{:else if costSummary}
		<!-- Summary Cards -->
		<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
			<!-- Estimated -->
			<Card.Root>
				<Card.Header class="pb-2">
					<Card.Description class="flex items-center gap-1.5">
						<Clock class="w-3.5 h-3.5" />
						Estimated
					</Card.Description>
					<Card.Title class="text-2xl font-bold">
						${formatCurrency(costSummary.estimated.total)}
					</Card.Title>
				</Card.Header>
				<Card.Content>
					<p class="text-sm text-muted-foreground">
						{formatHours(costSummary.estimated.laborHours)} labor hours
					</p>
				</Card.Content>
			</Card.Root>

			<!-- Actual -->
			<Card.Root>
				<Card.Header class="pb-2">
					<Card.Description class="flex items-center gap-1.5">
						<DollarSign class="w-3.5 h-3.5" />
						Actual
					</Card.Description>
					<Card.Title class="text-2xl font-bold">
						${formatCurrency(costSummary.actual.total)}
					</Card.Title>
				</Card.Header>
				<Card.Content>
					<p class="text-sm text-muted-foreground">
						{formatHours(costSummary.actual.laborHours)} labor hours
					</p>
				</Card.Content>
			</Card.Root>

			<!-- Variance -->
			<Card.Root class="border-l-4 {toNum(costSummary.variance.total) > 0 ? 'border-l-red-500' : toNum(costSummary.variance.total) < 0 ? 'border-l-green-500' : 'border-l-muted-foreground'}">
				<Card.Header class="pb-2">
					<Card.Description class="flex items-center gap-1.5">
						{#if toNum(costSummary.variance.total) > 0}
							<TrendingUp class="w-3.5 h-3.5 text-red-500" />
						{:else if toNum(costSummary.variance.total) < 0}
							<TrendingDown class="w-3.5 h-3.5 text-green-500" />
						{:else}
							<Minus class="w-3.5 h-3.5" />
						{/if}
						Variance
					</Card.Description>
					<Card.Title class="text-2xl font-bold {varianceColor(costSummary.variance.total)}">
						{varianceSign(costSummary.variance.total)}${formatCurrency(costSummary.variance.total)}
					</Card.Title>
				</Card.Header>
				<Card.Content>
					<Badge class="{varianceBg(costSummary.variance.percent)} {varianceColor(costSummary.variance.percent)} border-transparent">
						{varianceSign(costSummary.variance.percent)}{formatHours(costSummary.variance.percent)}%
					</Badge>
					<span class="text-sm text-muted-foreground ml-2">
						{varianceSign(costSummary.variance.laborHours)}{formatHours(costSummary.variance.laborHours)}h
					</span>
				</Card.Content>
			</Card.Root>
		</div>

		<!-- Estimated vs Actual Bar Comparison -->
		{@const maxCost = Math.max(toNum(costSummary.estimated.total), toNum(costSummary.actual.total), 1)}
		<Card.Root>
			<Card.Header>
				<Card.Title class="text-sm font-medium">Cost Comparison</Card.Title>
			</Card.Header>
			<Card.Content class="space-y-3">
				<div class="space-y-1">
					<div class="flex items-center justify-between text-sm">
						<span class="text-muted-foreground">Estimated</span>
						<span class="font-medium">${formatCurrency(costSummary.estimated.total)}</span>
					</div>
					<div class="h-3 bg-muted rounded-full overflow-hidden">
						<div
							class="h-full bg-blue-500 rounded-full transition-all duration-500"
							style="width: {(toNum(costSummary.estimated.total) / maxCost) * 100}%"
						></div>
					</div>
				</div>
				<div class="space-y-1">
					<div class="flex items-center justify-between text-sm">
						<span class="text-muted-foreground">Actual</span>
						<span class="font-medium">${formatCurrency(costSummary.actual.total)}</span>
					</div>
					<div class="h-3 bg-muted rounded-full overflow-hidden">
						<div
							class="h-full rounded-full transition-all duration-500 {toNum(costSummary.variance.total) > 0 ? 'bg-red-500' : 'bg-green-500'}"
							style="width: {(toNum(costSummary.actual.total) / maxCost) * 100}%"
						></div>
					</div>
				</div>
			</Card.Content>
		</Card.Root>

		<!-- Station Breakdown -->
		{#if costSummary.byStation.length > 0}
			<Card.Root>
				<Card.Header>
					<Card.Title class="text-sm font-medium">Station Breakdown</Card.Title>
					<Card.Description>Estimated vs actual hours per station</Card.Description>
				</Card.Header>
				<Card.Content>
					<Table.Root>
						<Table.Header>
							<Table.Row>
								<Table.Head>Station</Table.Head>
								<Table.Head class="text-right">Est. Hours</Table.Head>
								<Table.Head class="text-right">Actual Hours</Table.Head>
								<Table.Head class="text-right">Variance</Table.Head>
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each costSummary.byStation as station (station.stationId)}
								<Table.Row>
									<Table.Cell class="font-medium">{station.stationName || station.stationId.slice(0, 8)}</Table.Cell>
									<Table.Cell class="text-right">{formatHours(station.estimatedHours)}</Table.Cell>
									<Table.Cell class="text-right">{formatHours(station.actualHours)}</Table.Cell>
									<Table.Cell class="text-right">
										<span class="{varianceColor(station.variance)} font-medium">
											{varianceSign(station.variance)}{formatHours(station.variance)}h
										</span>
									</Table.Cell>
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>
				</Card.Content>
			</Card.Root>
		{/if}

		<!-- Task-Level Breakdown (Expandable) -->
		{#if taskRows.length > 0}
			<Collapsible.Root bind:open={taskBreakdownOpen}>
				<Card.Root>
					<Card.Header>
						<div class="flex items-center justify-between">
							<div>
								<Card.Title class="text-sm font-medium">Task Breakdown</Card.Title>
								<Card.Description>Per-task estimated vs actual time</Card.Description>
							</div>
							<Collapsible.Trigger
								class="inline-flex items-center justify-center gap-1 rounded-md text-sm font-medium h-9 px-3 hover:bg-accent hover:text-accent-foreground"
							>
								{#if taskBreakdownOpen}
									<ChevronDown class="size-4" />
								{:else}
									<ChevronRight class="size-4" />
								{/if}
								{taskRows.length} tasks
							</Collapsible.Trigger>
						</div>
					</Card.Header>
					<Collapsible.Content>
						<Card.Content>
							<Table.Root>
								<Table.Header>
									<Table.Row>
										<Table.Head>
											<button class="flex items-center gap-1 hover:text-foreground" onclick={() => toggleSort('title')}>
												Task
												{#if sortBy === 'title'}
													<ArrowUpDown class="size-3" />
												{/if}
											</button>
										</Table.Head>
										<Table.Head>
											<button class="flex items-center gap-1 hover:text-foreground" onclick={() => toggleSort('station')}>
												Station
												{#if sortBy === 'station'}
													<ArrowUpDown class="size-3" />
												{/if}
											</button>
										</Table.Head>
										<Table.Head class="text-right">Estimated</Table.Head>
										<Table.Head class="text-right">Actual</Table.Head>
										<Table.Head class="text-right">
											<button class="flex items-center gap-1 justify-end hover:text-foreground" onclick={() => toggleSort('variance')}>
												Variance
												{#if sortBy === 'variance'}
													<ArrowUpDown class="size-3" />
												{/if}
											</button>
										</Table.Head>
									</Table.Row>
								</Table.Header>
								<Table.Body>
									{#each taskRows as row (row.id)}
										<Table.Row>
											<Table.Cell>
												<div class="max-w-[200px]">
													<span class="text-sm font-medium truncate block">{row.title}</span>
													<span class="text-xs text-muted-foreground font-mono">{row.id}</span>
												</div>
											</Table.Cell>
											<Table.Cell class="text-sm">{row.stationName}</Table.Cell>
											<Table.Cell class="text-right text-sm">
												{row.estimatedSecs > 0 ? formatDuration(row.estimatedSecs) : '--'}
											</Table.Cell>
											<Table.Cell class="text-right text-sm">
												{row.actualSecs > 0 ? formatDuration(row.actualSecs) : '--'}
											</Table.Cell>
											<Table.Cell class="text-right">
												{#if row.estimatedSecs > 0 || row.actualSecs > 0}
													<span class="{varianceColor(row.varianceSecs)} text-sm font-medium">
														{varianceSign(row.varianceSecs)}{formatDuration(Math.abs(row.varianceSecs))}
													</span>
												{:else}
													<span class="text-sm text-muted-foreground">--</span>
												{/if}
											</Table.Cell>
										</Table.Row>
									{/each}
								</Table.Body>
							</Table.Root>
						</Card.Content>
					</Collapsible.Content>
				</Card.Root>
			</Collapsible.Root>
		{/if}
	{/if}
</div>
