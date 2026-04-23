import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { spaceApi, type Space } from '$lib/api/space';

interface SpaceState {
  spaces: Space[];
  recentSpaces: Space[];
  currentSpace: Space | null;
  isLoading: boolean;
  error: string | null;
}

function createSpaceStore() {
  const initialState: SpaceState = {
    spaces: [],
    recentSpaces: [],
    currentSpace: null,
    isLoading: false,
    error: null,
  };

  const { subscribe, set, update } = writable<SpaceState>(initialState);

  return {
    subscribe,

    async loadSpaces() {
      if (!browser) return;

      update(state => ({ ...state, isLoading: true, error: null }));

      try {
        const spaces = await spaceApi.getAll();
        update(state => ({
          ...state,
          spaces,
          isLoading: false,
        }));
      } catch (error) {
        console.error('Failed to load spaces:', error);
        update(state => ({
          ...state,
          error: error instanceof Error ? error.message : 'Failed to load spaces',
          isLoading: false,
        }));
      }
    },

    async loadRecentSpaces() {
      if (!browser) return;

      try {
        const recentSpaces = await spaceApi.getRecent();
        update(state => ({
          ...state,
          recentSpaces,
        }));
      } catch (error) {
        console.error('Failed to load recent spaces:', error);
      }
    },

    async loadSpace(id: string) {
      if (!browser) return;

      update(state => ({ ...state, isLoading: true, error: null }));

      try {
        const space = await spaceApi.getById(id);
        await spaceApi.recordVisit(id); // Record the visit
        update(state => ({
          ...state,
          currentSpace: space,
          isLoading: false,
        }));
        // Refresh recent spaces after visit
        this.loadRecentSpaces();
      } catch (error) {
        console.error('Failed to load space:', error);
        update(state => ({
          ...state,
          error: error instanceof Error ? error.message : 'Failed to load space',
          isLoading: false,
        }));
      }
    },

    async loadSpaceBySlug(slug: string) {
      if (!browser) return;

      update(state => ({ ...state, isLoading: true, error: null }));

      try {
        const space = await spaceApi.getBySlug(slug);
        await spaceApi.recordVisit(space.id); // Record the visit
        update(state => ({
          ...state,
          currentSpace: space,
          isLoading: false,
        }));
        // Refresh recent spaces after visit
        this.loadRecentSpaces();
      } catch (error) {
        console.error('Failed to load space:', error);
        update(state => ({
          ...state,
          error: error instanceof Error ? error.message : 'Failed to load space',
          isLoading: false,
        }));
      }
    },

    async createSpace(name: string, slug?: string) {
      if (!browser) return null;

      try {
        const newSpace = await spaceApi.create({ name, slug });
        update(state => ({
          ...state,
          spaces: [...state.spaces, newSpace],
        }));
        return newSpace;
      } catch (error) {
        console.error('Failed to create space:', error);
        update(state => ({
          ...state,
          error: error instanceof Error ? error.message : 'Failed to create space',
        }));
        return null;
      }
    },

    async updateSpace(id: string, data: { name?: string; slug?: string }) {
      if (!browser) return null;

      try {
        const updatedSpace = await spaceApi.update(id, data);
        update(state => ({
          ...state,
          spaces: state.spaces.map(s => (s.id === id ? updatedSpace : s)),
          currentSpace: state.currentSpace?.id === id ? updatedSpace : state.currentSpace,
        }));
        return updatedSpace;
      } catch (error) {
        console.error('Failed to update space:', error);
        update(state => ({
          ...state,
          error: error instanceof Error ? error.message : 'Failed to update space',
        }));
        return null;
      }
    },

    async deleteSpace(id: string) {
      if (!browser) return false;

      try {
        await spaceApi.delete(id);
        update(state => ({
          ...state,
          spaces: state.spaces.filter(s => s.id !== id),
          recentSpaces: state.recentSpaces.filter(s => s.id !== id),
          currentSpace: state.currentSpace?.id === id ? null : state.currentSpace,
        }));
        return true;
      } catch (error) {
        console.error('Failed to delete space:', error);
        update(state => ({
          ...state,
          error: error instanceof Error ? error.message : 'Failed to delete space',
        }));
        return false;
      }
    },

    async recordVisit(id: string) {
      if (!browser) return;

      try {
        await spaceApi.recordVisit(id);
        this.loadRecentSpaces();
      } catch (error) {
        console.error('Failed to record visit:', error);
      }
    },

    clearError() {
      update(state => ({ ...state, error: null }));
    },

    reset() {
      set(initialState);
    },
  };
}

export const spaceStore = createSpaceStore();
