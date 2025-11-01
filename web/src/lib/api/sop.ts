import { apiClient } from './client';

// Types matching the backend DTOs
export interface SOPStep {
  id?: number;
  stepNumber: number;
  title: string;
  instructions?: string;
  estimatedTimeMinutes?: number;
  imageUrl?: string;
  videoUrl?: string;
  requiresApproval: boolean;
}

export interface SOPVersion {
  id: number;
  versionNumber: number;
  status: 'draft' | 'published';
  description?: string;
  materials?: string[];
  equipment?: string[];
  changeSummary?: string;
  createdAt: string;
  updatedAt: string;
  isActive: boolean;
  steps?: SOPStep[];
}

export interface SOPTemplate {
  id: number;
  name: string;
  createdAt: string;
  updatedAt: string;
  currentVersion?: SOPVersion;
  activeDraftId?: number;
}

export interface CreateSOPRequest {
  name: string;
  description?: string;
  materials?: string[];
  equipment?: string[];
  changeSummary?: string;
  steps: Omit<SOPStep, 'id'>[];
}

export interface UpdateSOPRequest {
  name?: string;
  description?: string;
  materials?: string[];
  equipment?: string[];
  changeSummary: string;
  steps: Omit<SOPStep, 'id'>[];
}

export interface SaveDraftRequest {
  name?: string;
  description?: string;
  materials?: string[];
  equipment?: string[];
  changeSummary?: string;
  steps: Omit<SOPStep, 'id'>[];
}

export interface PublishDraftRequest {
  changeSummary: string;
}

export interface DraftListItem {
  id: number;
  sopTemplateId: number;
  sopTemplateName: string;
  versionNumber: number;
  changeSummary?: string;
  createdAt: string;
  updatedAt: string;
  // Computed fields for display
  templateId?: number;
  templateName?: string;
  description?: string;
  stepCount?: number;
}

class SOPApi {
  async getAllSOPs(): Promise<SOPTemplate[]> {
    return apiClient.get<SOPTemplate[]>('/sops/');
  }

  async getSOP(id: number): Promise<SOPTemplate> {
    return apiClient.get<SOPTemplate>(`/sops/${id}`);
  }

  async getSOPVersions(id: number): Promise<SOPVersion[]> {
    return apiClient.get<SOPVersion[]>(`/sops/${id}/versions`);
  }

  async getSOPVersion(versionId: number): Promise<SOPVersion> {
    return apiClient.get<SOPVersion>(`/sops/versions/${versionId}`);
  }

  async createSOP(data: CreateSOPRequest): Promise<SOPTemplate> {
    return apiClient.post<SOPTemplate>('/sops/', data);
  }

  async updateSOP(id: number, data: UpdateSOPRequest): Promise<SOPTemplate> {
    return apiClient.put<SOPTemplate>(`/sops/${id}`, data);
  }

  async deleteSOP(id: number): Promise<void> {
    return apiClient.delete<void>(`/sops/${id}`);
  }

  // Draft operations
  async getUserDrafts(): Promise<DraftListItem[]> {
    return apiClient.get<DraftListItem[]>('/sops/drafts');
  }

  async getSOPDrafts(id: number): Promise<SOPVersion[]> {
    return apiClient.get<SOPVersion[]>(`/sops/${id}/drafts`);
  }

  async saveDraft(id: number, data: SaveDraftRequest): Promise<SOPVersion> {
    return apiClient.post<SOPVersion>(`/sops/${id}/drafts`, data);
  }

  async updateDraft(draftId: number, data: SaveDraftRequest): Promise<SOPVersion> {
    return apiClient.put<SOPVersion>(`/sops/drafts/${draftId}`, data);
  }

  async publishDraft(draftId: number, data: PublishDraftRequest): Promise<SOPTemplate> {
    return apiClient.post<SOPTemplate>(`/sops/drafts/${draftId}/publish`, data);
  }

  async deleteDraft(draftId: number): Promise<void> {
    return apiClient.delete<void>(`/sops/drafts/${draftId}`);
  }

  async getDraft(draftId: number): Promise<SOPVersion> {
    return apiClient.get<SOPVersion>(`/sops/drafts/${draftId}`);
  }
}

export const sopApi = new SOPApi();
