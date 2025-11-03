import { writable } from 'svelte/store';
import { sopApi, type SOPTemplate, type SOPVersion, type DraftListItem } from '$lib/api/sop';

interface SOPStore {
  sops: SOPTemplate[];
  currentSOP: SOPTemplate | null;
  currentVersions: SOPVersion[];
  drafts: DraftListItem[];
  currentDraft: SOPVersion | null;
  loading: boolean;
  error: string | null;
}

function createSOPStore() {
  const { subscribe, set, update } = writable<SOPStore>({
    sops: [],
    currentSOP: null,
    currentVersions: [],
    drafts: [],
    currentDraft: null,
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

    async loadUserDrafts() {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        const drafts = await sopApi.getUserDrafts();
        update(state => ({ ...state, drafts, loading: false }));
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to load drafts';
        update(state => ({ ...state, error: message, loading: false }));
      }
    },

    async saveDraft(id: number, data: Parameters<typeof sopApi.saveDraft>[1]) {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        const updatedSOP = await sopApi.saveDraft(id, data);
        update(state => ({ 
          ...state, 
          currentSOP: updatedSOP,
          loading: false 
        }));
        return updatedSOP;
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to save draft';
        update(state => ({ ...state, error: message, loading: false }));
        throw error;
      }
    },

    async updateDraft(draftId: number, data: Parameters<typeof sopApi.updateDraft>[1]) {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        const updatedSOP = await sopApi.updateDraft(draftId, data);
        update(state => ({ 
          ...state, 
          currentSOP: updatedSOP,
          loading: false 
        }));
        return updatedSOP;
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to update draft';
        update(state => ({ ...state, error: message, loading: false }));
        throw error;
      }
    },

    async publishDraft(draftId: number, changeSummary: string) {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        const updatedSOP = await sopApi.publishDraft(draftId, { changeSummary });
        update(state => ({
          ...state,
          sops: Array.isArray(state.sops) ? state.sops.map(sop => sop.id === updatedSOP.id ? updatedSOP : sop) : [updatedSOP],
          currentSOP: state.currentSOP?.id === updatedSOP.id ? updatedSOP : state.currentSOP,
          currentDraft: null,
          drafts: state.drafts.filter(d => d.id !== draftId),
          loading: false
        }));
        return updatedSOP;
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to publish draft';
        update(state => ({ ...state, error: message, loading: false }));
        throw error;
      }
    },

    async deleteDraft(draftId: number) {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        await sopApi.deleteDraft(draftId);
        update(state => ({
          ...state,
          drafts: state.drafts.filter(d => d.id !== draftId),
          currentDraft: state.currentDraft?.id === draftId ? null : state.currentDraft,
          loading: false
        }));
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to delete draft';
        update(state => ({ ...state, error: message, loading: false }));
        throw error;
      }
    },

    async loadDraft(draftId: number) {
      update(state => ({ ...state, loading: true, error: null }));
      try {
        const draft = await sopApi.getDraft(draftId);
        update(state => ({ ...state, currentDraft: draft, loading: false }));
        return draft;
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to load draft';
        update(state => ({ ...state, error: message, loading: false }));
        throw error;
      }
    },

    async ensureDraft(sopId: number): Promise<SOPTemplate> {
      update(state => ({ ...state, loading: true, error: null }));
      
      try {
        // Reload the SOP to get the latest state
        const currentSOP = await sopApi.getSOP(sopId);
        
        // If we already have a draft, return it
        if (currentSOP.activeDraftId) {
          update(state => ({ ...state, currentSOP, loading: false }));
          return currentSOP;
        }

        // Create a draft from the current published version
        if (!currentSOP.currentVersion) {
          throw new Error('No current version found to create draft from');
        }

        const draftData = {
          description: currentSOP.currentVersion.description || '',
          materials: currentSOP.currentVersion.materials || [],
          equipment: currentSOP.currentVersion.equipment || [],
          changeSummary: 'Draft created for editing',
          steps: (currentSOP.currentVersion.steps || []).map((s: any) => ({
            order: s.order,
            title: s.title,
            instructions: s.instructions,
            estimatedTimeMinutes: s.estimatedTimeMinutes,
            imageUrl: s.imageUrl,
            videoUrl: s.videoUrl,
            requiresApproval: s.requiresApproval
          }))
        };

        return await this.saveDraft(sopId, draftData);
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed to ensure draft';
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
        drafts: [],
        currentDraft: null,
        loading: false,
        error: null
      });
    }
  };
}

export const sopStore = createSOPStore();
