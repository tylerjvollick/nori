<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { jobApi } from '$lib/api/job';
	import { spaceStore } from '$lib/stores/space';
	import type { TaskResponse } from '$lib/types/task';
	import * as Card from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import * as Alert from '$lib/components/ui/alert';
	import { CircleAlert } from '@lucide/svelte';

	let currentSpace = $derived($spaceStore.currentSpace);
	let slug = $derived($page.params.slug);

	let jobs = $state<TaskResponse[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		const space = currentSpace;
		if (!space) return;
		loading = true;
		error = null;
		jobApi.listJobs(space.id)
			.then((res) => { jobs = res.items; })
			.catch((err) => { error = err instanceof Error ? err.message : 'Failed to load jobs'; })
			.finally(() => { loading = false; });
	});

	const STATUS_COLORS: Record<string, string> = {
		open: 'bg-slate-50 dark:bg-slate-950 text-slate-700 dark:text-slate-300 border-slate-200 dark:border-slate-700',
		active: 'bg-blue-50 dark:bg-blue-950 text-blue-700 dark:text-blue-300 border-blue-200 dark:border-blue-800',
		paused: 'bg-yellow-50 dark:bg-yellow-950 text-yellow-700 dark:text-yellow-300 border-yellow-200 dark:border-yellow-800',
		done: 'bg-green-50 dark:bg-green-950 text-green-700 dark:text-green-300 border-green-200 dark:border-green-800',
		skipped: 'bg-muted text-muted-foreground',
	};

	function formatDate(dateString: string): string {
		return new Date(dateString).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
		});
	}

	function priorityLabel(p: number): string {
		return ['P0', 'P1', 'P2', 'P3', 'P4'][p] ?? `P${p}`;
	}
</script>

<svelte:head>
	<title>{currentSpace?.name ?? 'Space'} – Jobs - Nori</title>
</svelte:head>

<div class="container mx-auto px-4 py-8">
	<div class="flex justify-between items-center mb-8">
		<div>
			<h1 class="text-2xl font-bold text-foreground">Jobs</h1>
			<p class="text-muted-foreground mt-1 text-sm">Active production orders in this space</p>
		</div>
	</div>

	{#if loading}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each Array(6) as _}
				<Card.Root>
					<Card.Header>
						<div class="flex items-start justify-between gap-2">
							<Skeleton class="h-5 w-3/4" />
							<Skeleton class="h-5 w-12" />
						</div>
					</Card.Header>
					<Card.Footer>
						<Skeleton class="h-3 w-24" />
					</Card.Footer>
				</Card.Root>
			{/each}
		</div>
	{:else if error}
		<Alert.Root variant="destructive">
			<CircleAlert class="size-4" />
			<Alert.Title>Error loading jobs</Alert.Title>
			<Alert.Description>{error}</Alert.Description>
		</Alert.Root>
	{:else if jobs.length === 0}
		<div class="text-center py-16">
			<p class="text-muted-foreground">No jobs yet in this space.</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each jobs as job (job.id)}
				<a href="/spaces/{slug}/{job.id}" class="block group">
					<Card.Root class="h-full transition-all hover:shadow-md hover:ring-1 hover:ring-primary/30">
						<Card.Header class="pb-2">
							<div class="flex items-start justify-between gap-2">
								<Card.Title class="text-base group-hover:text-primary transition-colors line-clamp-2">
									{job.title}
								</Card.Title>
								<Badge
									variant="outline"
									class="shrink-0 text-xs capitalize {STATUS_COLORS[job.status] ?? ''}"
								>
									{job.status}
								</Badge>
							</div>
							{#if job.description}
								<p class="text-xs text-muted-foreground line-clamp-2 mt-1">{job.description}</p>
							{/if}
						</Card.Header>
						<Card.Footer class="pt-0">
							<div class="flex items-center justify-between w-full text-xs text-muted-foreground">
								<span>{priorityLabel(job.priority)}</span>
								<span>{formatDate(job.createdAt)}</span>
							</div>
						</Card.Footer>
					</Card.Root>
				</a>
			{/each}
		</div>
	{/if}
</div>
