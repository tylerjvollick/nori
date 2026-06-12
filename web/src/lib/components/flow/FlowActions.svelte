<script lang="ts">
	import { useSvelteFlow } from '@xyflow/svelte';

	/**
	 * Thin wrapper that must live inside <SvelteFlow> to access useSvelteFlow().
	 * Exposes flow actions (setCenter, fitView, zoomIn, zoomOut) to the parent
	 * via a bindable handle object.
	 */

	interface FlowHandle {
		setCenter: (x: number, y: number, opts?: { zoom?: number; duration?: number }) => void;
		fitView: (opts?: { padding?: number; duration?: number }) => void;
		zoomIn: (opts?: { duration?: number }) => void;
		zoomOut: (opts?: { duration?: number }) => void;
	}

	let { handle = $bindable() }: { handle?: FlowHandle } = $props();

	const { setCenter, fitView, zoomIn, zoomOut } = useSvelteFlow();

	handle = { setCenter, fitView, zoomIn, zoomOut };
</script>
