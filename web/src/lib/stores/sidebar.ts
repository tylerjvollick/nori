import { writable } from 'svelte/store';
import { browser } from '$app/environment';

interface SidebarState {
	collapsed: boolean;
	spacesExpanded: boolean;
}

function createSidebarStore() {
	const defaultState: SidebarState = {
		collapsed: false,
		spacesExpanded: true
	};

	// Load from localStorage if available
	const storedState = browser ? localStorage.getItem('sidebar-state') : null;
	const initialState = storedState ? JSON.parse(storedState) : defaultState;

	const { subscribe, set, update } = writable<SidebarState>(initialState);

	return {
		subscribe,
		toggle: () =>
			update((state) => {
				const newState = { ...state, collapsed: !state.collapsed };
				if (browser) localStorage.setItem('sidebar-state', JSON.stringify(newState));
				return newState;
			}),
		toggleSpaces: () =>
			update((state) => {
				const newState = { ...state, spacesExpanded: !state.spacesExpanded };
				if (browser) localStorage.setItem('sidebar-state', JSON.stringify(newState));
				return newState;
			}),
		setCollapsed: (collapsed: boolean) =>
			update((state) => {
				const newState = { ...state, collapsed };
				if (browser) localStorage.setItem('sidebar-state', JSON.stringify(newState));
				return newState;
			}),
		reset: () => {
			if (browser) localStorage.removeItem('sidebar-state');
			set(defaultState);
		}
	};
}

export const sidebarStore = createSidebarStore();
