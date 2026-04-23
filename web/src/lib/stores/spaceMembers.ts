import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { spaceApi, type SpaceMember } from '$lib/api/space';

interface SpaceMembersState {
	members: SpaceMember[];
	/** The space ID these members belong to. Used to avoid redundant fetches. */
	spaceId: string | null;
	isLoading: boolean;
}

function createSpaceMembersStore() {
	const initialState: SpaceMembersState = {
		members: [],
		spaceId: null,
		isLoading: false,
	};

	const { subscribe, set, update } = writable<SpaceMembersState>(initialState);

	return {
		subscribe,

		/**
		 * Load members for a space. Skips the fetch if already loaded for this spaceId.
		 * Pass `force: true` to reload.
		 */
		async loadMembers(spaceId: string, opts?: { force?: boolean }) {
			if (!browser) return;

			// Check if already loaded for this space
			let currentState: SpaceMembersState | undefined;
			const unsub = subscribe((s) => (currentState = s));
			unsub();

			if (currentState?.spaceId === spaceId && !opts?.force && currentState.members.length > 0) {
				return; // Already loaded
			}

			update((s) => ({ ...s, isLoading: true }));

			try {
				const members = await spaceApi.getMembers(spaceId);
				set({ members, spaceId, isLoading: false });
			} catch (err) {
				console.error('Failed to load space members:', err);
				update((s) => ({ ...s, isLoading: false }));
			}
		},

		reset() {
			set(initialState);
		},
	};
}

export const spaceMembersStore = createSpaceMembersStore();
