import { writable } from 'svelte/store';
import { browser } from '$app/environment';

/** Dagre layout direction for the dependency graph. */
export type GraphDirection = 'LR' | 'RL' | 'TB' | 'BT';

const STORAGE_KEY = 'graph-direction';
const DEFAULT_DIRECTION: GraphDirection = 'TB';

const VALID_DIRECTIONS = new Set<GraphDirection>(['LR', 'RL', 'TB', 'BT']);

function getInitialDirection(): GraphDirection {
	if (!browser) return DEFAULT_DIRECTION;
	const stored = localStorage.getItem(STORAGE_KEY) as GraphDirection | null;
	if (stored && VALID_DIRECTIONS.has(stored)) return stored;
	return DEFAULT_DIRECTION;
}

function createGraphDirectionStore() {
	const { subscribe, set, update } = writable<GraphDirection>(getInitialDirection());

	return {
		subscribe,
		set: (dir: GraphDirection) => {
			if (browser) localStorage.setItem(STORAGE_KEY, dir);
			set(dir);
		},
		/** Cycle through directions: LR → TB → RL → BT → LR */
		cycle: () =>
			update((current) => {
				const order: GraphDirection[] = ['LR', 'TB', 'RL', 'BT'];
				const idx = order.indexOf(current);
				const next = order[(idx + 1) % order.length];
				if (browser) localStorage.setItem(STORAGE_KEY, next);
				return next;
			}),
	};
}

export const graphDirection = createGraphDirectionStore();
