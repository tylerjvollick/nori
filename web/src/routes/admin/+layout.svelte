<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { Users, Key, LayoutGrid } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';

	let { children } = $props();

	let pathname = $derived($page.url.pathname);

	const navItems = [
		{ title: 'Users', href: '/admin/users', icon: Users },
		{ title: 'API Keys', href: '/admin/api-keys', icon: Key },
		{ title: 'Spaces', href: '/admin/spaces', icon: LayoutGrid },
	];

	function isActive(href: string): boolean {
		return pathname === href || pathname.startsWith(href + '/');
	}
</script>

<div class="flex h-full flex-col overflow-hidden">
	<!-- Admin Header -->
	<div class="flex-shrink-0 border-b px-6 py-4">
		<h1 class="text-2xl font-bold text-foreground">Admin Settings</h1>
		<p class="text-sm text-muted-foreground mt-1">Manage users, API keys, and space membership.</p>
	</div>

	<div class="flex flex-1 overflow-hidden">
		<!-- Sub-navigation -->
		<nav class="flex-shrink-0 w-52 border-r p-4 space-y-1">
			{#each navItems as item (item.title)}
				<Button
					variant={isActive(item.href) ? 'secondary' : 'ghost'}
					class="w-full justify-start gap-2"
					onclick={() => goto(item.href)}
				>
					<item.icon class="size-4" />
					{item.title}
				</Button>
			{/each}
		</nav>

		<!-- Page content -->
		<div class="flex-1 overflow-auto p-6">
			{@render children()}
		</div>
	</div>
</div>
