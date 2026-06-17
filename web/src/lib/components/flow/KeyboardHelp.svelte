<script lang="ts">
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
				{ key: 'o', description: 'Switch to overview' },
				{ key: 'b', description: 'Switch to board view' },
				{ key: 'g', description: 'Switch to graph view' },
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
				{ key: 'c', description: 'Start selected task' },
				{ key: 'd', description: 'Complete selected task' },
			],
		},
		{
			title: 'List View',
			shortcuts: [
				{ key: 'j', description: 'Select next row' },
				{ key: 'k', description: 'Select previous row' },
				{ key: 'Enter', description: 'Open selected task' },
				{ key: 'c', description: 'Start selected task' },
				{ key: 'd', description: 'Complete selected task' },
			],
		},
		{
			title: 'Graph View',
			shortcuts: [
				{ key: 'j', description: 'Navigate downstream (dependents)' },
				{ key: 'k', description: 'Navigate upstream (blockers)' },
				{ key: 'h', description: 'Previous sibling' },
				{ key: 'l / Tab', description: 'Next sibling / cycle' },
				{ key: 'Enter', description: 'Open selected task' },
				{ key: 'Del / ⌫', description: 'Delete selected node' },
				{ key: 'Click, Del', description: 'Delete selected dependency' },
				{ key: 'Ctrl+R', description: 'Rename selected node' },
				{ key: 'Alt+S', description: 'Add serial node' },
				{ key: 'Alt+P', description: 'Add parallel node' },
				{ key: 'Alt+L', description: 'Re-layout graph' },
				{ key: '+ / =', description: 'Zoom in' },
				{ key: '-', description: 'Zoom out' },
				{ key: '0', description: 'Fit graph to view' },
			],
		},
		{
			title: 'Inline Editing',
			shortcuts: [
				{ key: 'Enter', description: 'Commit title' },
				{ key: '⌘ Enter', description: 'Commit title + add serial node' },
				{ key: 'Esc', description: 'Cancel and revert' },
			],
		},
	];
</script>

{#if open}
	<!-- Backdrop -->
	<button
		class="fixed inset-0 z-50 bg-black/50"
		onclick={onclose}
		aria-label="Close keyboard shortcuts"
	></button>

	<!-- Modal -->
	<div
		class="fixed top-1/2 left-1/2 z-50 w-full max-w-2xl -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-popover p-6 text-popover-foreground shadow-lg max-h-[80vh] overflow-y-auto"
		role="dialog"
		aria-label="Keyboard Shortcuts"
	>
		<div class="flex items-center gap-2 mb-1">
			<Keyboard class="size-5" />
			<h2 class="text-lg font-semibold" data-testid="keyboard-shortcuts-heading">Keyboard Shortcuts</h2>
		</div>
		<p class="text-sm text-muted-foreground mb-4">
			Vim-style navigation. Shortcuts are disabled when typing in inputs.
		</p>

		<div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
			{#each groups as group (group.title)}
				<div>
					<h3 class="mb-2 text-sm font-semibold text-foreground">{group.title}</h3>
					<div class="space-y-1">
						{#each group.shortcuts as shortcut (shortcut.key)}
							<div class="flex items-center justify-between py-0.5">
								<span class="text-sm text-muted-foreground">{shortcut.description}</span>
								<kbd
									class="ml-2 inline-flex items-center justify-center rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground min-w-[28px] shrink-0"
								>
									{shortcut.key}
								</kbd>
							</div>
						{/each}
					</div>
				</div>
			{/each}
		</div>

		<div class="mt-6 flex justify-end">
			<Button variant="outline" size="sm" onclick={onclose}>
				Close
				<kbd class="ml-2 rounded border border-border bg-muted px-1.5 py-0 font-mono text-[10px]">Esc</kbd>
			</Button>
		</div>
	</div>
{/if}
