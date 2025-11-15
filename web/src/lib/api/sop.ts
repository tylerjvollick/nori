import { apiClient } from './client';

// Types matching the backend DTOs
export interface SOPStep {
  id: number;
  order: string;
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

// Step-specific request types
export interface CreateStepRequest {
  afterStepId?: number;
  title: string;
  instructions?: string;
  estimatedTimeMinutes?: number;
  imageUrl?: string;
  videoUrl?: string;
  requiresApproval?: boolean;
}

export interface UpdateStepRequest {
  title?: string;
  instructions?: string;
  estimatedTimeMinutes?: number;
  imageUrl?: string;
  videoUrl?: string;
  requiresApproval?: boolean;
}

export interface ReorderStepRequest {
  beforeStepId?: number;
  afterStepId?: number;
}

// Photo-specific types
export interface SOPStepPhoto {
  id: number;
  sopStepId: number;
  uuid: string;
  filePath: string;
  fileName: string;
  mimeType: string;
  fileSize: number;
  order: string;
  createdAt: string;
}

export interface ReorderPhotoRequest {
  beforePhotoId?: number;
  afterPhotoId?: number;
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

  async saveDraft(id: number, data: SaveDraftRequest): Promise<SOPTemplate> {
    return apiClient.post<SOPTemplate>(`/sops/${id}/drafts`, data);
  }

  async updateDraft(draftId: number, data: SaveDraftRequest): Promise<SOPTemplate> {
    return apiClient.put<SOPTemplate>(`/sops/drafts/${draftId}`, data);
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

  // Individual step operations (using template ID - backend auto-resolves to draft)
  async createStep(templateId: number, data: CreateStepRequest): Promise<SOPStep> {
    return apiClient.post<SOPStep>(`/sops/${templateId}/steps`, data);
  }

  async updateStep(templateId: number, stepId: number, data: UpdateStepRequest): Promise<SOPStep> {
    return apiClient.put<SOPStep>(`/sops/${templateId}/steps/${stepId}`, data);
  }

  async deleteStep(templateId: number, stepId: number): Promise<void> {
    return apiClient.delete<void>(`/sops/${templateId}/steps/${stepId}`);
  }

  async reorderStep(templateId: number, stepId: number, data: ReorderStepRequest): Promise<SOPStep> {
    return apiClient.patch<SOPStep>(`/sops/${templateId}/steps/${stepId}/reorder`, data);
  }

  // Photo operations
  async uploadStepPhoto(templateId: number, stepId: number, file: File): Promise<SOPStepPhoto> {
    const formData = new FormData();
    formData.append('photo', file);
    return apiClient.uploadFile<SOPStepPhoto>(`/sops/${templateId}/steps/${stepId}/photos`, formData);
  }

  async getStepPhotos(templateId: number, stepId: number): Promise<SOPStepPhoto[]> {
    return apiClient.get<SOPStepPhoto[]>(`/sops/${templateId}/steps/${stepId}/photos`);
  }

  async deleteStepPhoto(photoId: number): Promise<void> {
    return apiClient.delete<void>(`/photos/${photoId}`);
  }

  async reorderStepPhoto(photoId: number, data: ReorderPhotoRequest): Promise<SOPStepPhoto> {
    return apiClient.patch<SOPStepPhoto>(`/photos/${photoId}/reorder`, data);
  }

  getPhotoUrl(uuid: string): string {
    // Return the full URL for the photo
    const baseUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080';
    return `${baseUrl}/photos/${uuid}`;
  }
}

export const sopApi = new SOPApi();
