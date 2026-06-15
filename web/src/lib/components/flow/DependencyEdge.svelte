<script lang="ts">
	import { getSmoothStepPath, BaseEdge, EdgeReconnectAnchor } from '@xyflow/svelte';
	import type { EdgeProps } from '@xyflow/svelte';

	// Custom edge that renders the same smoothstep path as before but adds two
	// reconnect anchors (one per endpoint). In @xyflow/svelte v1 there is no
	// `edge.reconnectable` flag — dragging an endpoint to a new node is only
	// possible from inside a custom edge via <EdgeReconnectAnchor>. The anchor
	// drives onbeforereconnect/onreconnect on <SvelteFlow> (see GraphView).
	let {
		id,
		sourceX,
		sourceY,
		targetX,
		targetY,
		sourcePosition,
		targetPosition,
		markerStart,
		markerEnd,
		style,
		interactionWidth,
	}: EdgeProps = $props();

	let [path] = $derived(
		getSmoothStepPath({
			sourceX,
			sourceY,
			targetX,
			targetY,
			sourcePosition,
			targetPosition,
		}),
	);
</script>

<BaseEdge {id} {path} {markerStart} {markerEnd} {style} {interactionWidth} />

<!-- Upstream (source) endpoint anchor -->
<EdgeReconnectAnchor type="source" position={{ x: sourceX, y: sourceY }}>
	<div class="nori-edge-anchor"></div>
</EdgeReconnectAnchor>

<!-- Downstream (target) endpoint anchor -->
<EdgeReconnectAnchor type="target" position={{ x: targetX, y: targetY }}>
	<div class="nori-edge-anchor"></div>
</EdgeReconnectAnchor>

<style>
	/* Faint by default; brightened when the edge is hovered (see GraphView's
	   :global rule) so the drag affordance is discoverable without cluttering. */
	.nori-edge-anchor {
		width: 8px;
		height: 8px;
		border-radius: 9999px;
		background: var(--background, #fff);
		border: 2px solid #64748b; /* slate-500 */
		opacity: 0;
		transition: opacity 120ms ease;
		cursor: crosshair;
	}
</style>
