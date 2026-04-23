<script lang="ts">
	import { onMount } from 'svelte';
	import { LoaderCircle, CircleAlert } from '@lucide/svelte';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';

	interface Props {
		message?: string;
		timeoutMs?: number;
	}

	let { message = 'Redirecting...', timeoutMs = 5000 }: Props = $props();

	let timedOut = $state(false);

	onMount(() => {
		const timer = setTimeout(() => {
			timedOut = true;
		}, timeoutMs);

		return () => clearTimeout(timer);
	});
</script>

<div class="flex items-center justify-center h-full">
	{#if timedOut}
		<div class="w-full max-w-sm space-y-4">
			<Alert.Root variant="destructive">
				<CircleAlert />
				<Alert.Title>Something went wrong</Alert.Title>
				<Alert.Description>
					The redirect took too long. The page may be unavailable.
				</Alert.Description>
			</Alert.Root>
			<div class="flex justify-center gap-3">
				<Button variant="outline" onclick={() => window.location.reload()}>
					Try again
				</Button>
				<Button href="/">Go home</Button>
			</div>
		</div>
	{:else}
		<div class="flex flex-col items-center gap-3">
			<LoaderCircle class="size-6 animate-spin text-muted-foreground" />
			<p class="text-sm text-muted-foreground">{message}</p>
		</div>
	{/if}
</div>
