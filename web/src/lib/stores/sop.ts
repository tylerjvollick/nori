import { writable } from 'svelte/store';
import { sopApi, type SOPTemplate, type SOPVersion } from '$lib/api/sop';

interface SOPStore {
  sops: SOPTemplate[];
  currentSOP: SOPTemplate | null;
  currentVersions: SOPVersion[];
  loading: boolean;
  error: string | null;
}

function createSOPStore() {
  const { subscribe, set, update } = writable<SOPStore>({
    sops: [],
    currentSOP: null,
    currentVersions: [],
    loading: false,
    error: null
  });

  return {
    subscribe,

    async loadAllSOPs() {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        const sops = await sopApi.getAllSOPs();
        update(state => ({ ...state, sops, loading: false }));
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to load SOPs';
        update(state => ({ ...state, error: message, loading: false }));
      }
    },

    async loadSOP(id: number) {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        const sop = await sopApi.getSOP(id);
        update(state => ({ ...state, currentSOP: sop, loading: false }));
        return sop;
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to load SOP';
        update(state => ({ ...state, error: message, loading: false }));
        throw error;
      }
    },

    async loadSOPVersions(id: number) {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        const versions = await sopApi.getSOPVersions(id);
        update(state => ({ ...state, currentVersions: versions, loading: false }));
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to load versions';
        update(state => ({ ...state, error: message, loading: false }));
      }
    },

    async createSOP(data: Parameters<typeof sopApi.createSOP>[0]) {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        const newSOP = await sopApi.createSOP(data);
        update(state => ({ 
          ...state, 
          sops: Array.isArray(state.sops) ? [...state.sops, newSOP] : [newSOP], 
          loading: false 
        }));
        return newSOP;
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to create SOP';
        update(state => ({ ...state, error: message, loading: false }));
        throw error;
      }
    },

    async updateSOP(id: number, data: Parameters<typeof sopApi.updateSOP>[1]) {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        const updatedSOP = await sopApi.updateSOP(id, data);
        update(state => ({
          ...state,
          sops: Array.isArray(state.sops) ? state.sops.map(sop => sop.id === id ? updatedSOP : sop) : [updatedSOP],
          currentSOP: state.currentSOP?.id === id ? updatedSOP : state.currentSOP,
          loading: false
        }));
        return updatedSOP;
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to update SOP';
        update(state => ({ ...state, error: message, loading: false }));
        throw error;
      }
    },

    async deleteSOP(id: number) {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        await sopApi.deleteSOP(id);
        update(state => ({
          ...state,
          sops: Array.isArray(state.sops) ? state.sops.filter(sop => sop.id !== id) : [],
          currentSOP: state.currentSOP?.id === id ? null : state.currentSOP,
          loading: false
        }));
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to delete SOP';
        update(state => ({ ...state, error: message, loading: false }));
        throw error;
      }
    },

    clearError() {
      update(state => ({ ...state, error: null }));
    },

    reset() {
      set({
        sops: [],
        currentSOP: null,
        currentVersions: [],
        loading: false,
        error: null
      });
    }
  };
}

export const sopStore = createSOPStore();
