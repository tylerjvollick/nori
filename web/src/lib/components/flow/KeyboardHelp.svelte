<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { Keyboard } from '@lucide/svelte';

	interface Props {
		open: boolean;
		onclose: () => void;
	}

	let { open, onclose }: Props = $props();

	interface ShortcutGroup {
		title: string;
		shortcuts: { key: string; description: string }[];
	}

	const groups: ShortcutGroup[] = [
		{
			title: 'Global',
			shortcuts: [
				{ key: 'b', description: 'Switch to board view' },
				{ key: 'g', description: 'Switch to graph view' },
				{ key: 'l', description: 'Switch to list view' },
				{ key: '/', description: 'Focus filter bar' },
				{ key: 'Esc', description: 'Close panel / clear selection' },
				{ key: '?', description: 'Toggle this help overlay' },
			],
		},
		{
			title: 'Board View',
			shortcuts: [
				{ key: 'h', description: 'Select previous column' },
				{ key: 'j', description: 'Select next card down' },
				{ key: 'k', description: 'Select previous card up' },
				{ key: 'l', description: 'Select next column' },
				{ key: 'Enter', description: 'Open selected task' },
				{ key: 'c', description: 'Start selected task' },				{ key: 'd', description: 'Complete selected task' },
			],
		},
		{
			title: 'List View',
			shortcuts: [
				{ key: 'j', description: 'Select next row' },
				{ key: 'k', description: 'Select previous row' },
				{ key: 'Enter', description: 'Open selected task' },
				{ key: 'c', description: 'Start selected task' },				{ key: 'd', description: 'Complete selected task' },
			],
		},
		{
			title: 'Graph View',
			shortcuts: [
				{ key: 'Enter', description: 'Open selected node' },
				{ key: '+ / =', description: 'Zoom in' },
				{ key: '-', description: 'Zoom out' },
				{ key: '0', description: 'Fit graph to view' },
			],
		},
	];
</script>

<Dialog.Root
	{open}
	onOpenChange={(v) => {
		if (!v) onclose();
	}}
>
	<Dialog.Content class="max-w-2xl max-h-[80vh] overflow-y-auto">
		<Dialog.Header>
			<Dialog.Title class="flex items-center gap-2">
				<Keyboard class="size-5" />
				Keyboard Shortcuts
			</Dialog.Title>
			<Dialog.Description>
				Vim-style navigation for the flow board. Shortcuts are disabled when typing in inputs.
			</Dialog.Description>
		</Dialog.Header>

		<div class="mt-4 space-y-6">
			{#each groups as group (group.title)}
				<div>
					<h3 class="mb-2 text-sm font-semibold text-foreground">{group.title}</h3>
					<div class="space-y-1">
						{#each group.shortcuts as shortcut (shortcut.key)}
							<div class="flex items-center justify-between py-1">
								<span class="text-sm text-muted-foreground">{shortcut.description}</span>
								<kbd
									class="inline-flex items-center justify-center rounded border border-border bg-muted px-2 py-0.5 font-mono text-xs text-foreground min-w-[28px]"
								>
									{shortcut.key}
								</kbd>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		</div>

		<Dialog.Footer class="mt-6">
			<Button variant="outline" size="sm" onclick={onclose}>
				Close
				<kbd class="ml-2 rounded border border-border bg-muted px-1.5 py-0 font-mono text-[10px]">Esc</kbd>
			</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
