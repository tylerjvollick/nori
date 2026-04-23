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

// Media-specific types (photos and videos)
export interface SOPStepMedia {
  id: number;
  sopStepId: number;
  uuid: string;
  filePath: string;
  fileName: string;
  mimeType: string;
  fileSize: number;
  duration?: number; // Duration in seconds (for videos)
  order: string;
  createdAt: string;
}

// Backwards compatibility alias
export type SOPStepPhoto = SOPStepMedia;

export interface ReorderMediaRequest {
  beforeMediaId?: number;
  afterMediaId?: number;
}

// Backwards compatibility alias
export type ReorderPhotoRequest = ReorderMediaRequest;

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

  // Media operations (photos and videos)
  async uploadStepMedia(templateId: number, stepId: number, file: File): Promise<SOPStepMedia> {
    const formData = new FormData();
    formData.append('media', file);
    return apiClient.uploadFile<SOPStepMedia>(`/sops/${templateId}/steps/${stepId}/media`, formData);
  }

  async getStepMedia(templateId: number, stepId: number): Promise<SOPStepMedia[]> {
    return apiClient.get<SOPStepMedia[]>(`/sops/${templateId}/steps/${stepId}/media`);
  }

  async deleteStepMedia(mediaId: number): Promise<void> {
    return apiClient.delete<void>(`/media/${mediaId}`);
  }

  async reorderStepMedia(mediaId: number, data: ReorderMediaRequest): Promise<SOPStepMedia> {
    return apiClient.patch<SOPStepMedia>(`/media/${mediaId}/reorder`, data);
  }

  getMediaUrl(uuid: string): string {
    // Return the full URL for the media
    const baseUrl = import.meta.env.VITE_API_URL || 'http://localhost:8081';
    return `${baseUrl}/media/${uuid}`;
  }

  // Backwards compatibility methods (deprecated, use media methods instead)
  /** @deprecated Use uploadStepMedia instead */
  async uploadStepPhoto(templateId: number, stepId: number, file: File): Promise<SOPStepMedia> {
    return this.uploadStepMedia(templateId, stepId, file);
  }

  /** @deprecated Use getStepMedia instead */
  async getStepPhotos(templateId: number, stepId: number): Promise<SOPStepMedia[]> {
    return this.getStepMedia(templateId, stepId);
  }

  /** @deprecated Use deleteStepMedia instead */
  async deleteStepPhoto(photoId: number): Promise<void> {
    return this.deleteStepMedia(photoId);
  }

  /** @deprecated Use reorderStepMedia instead */
  async reorderStepPhoto(photoId: number, data: ReorderMediaRequest): Promise<SOPStepMedia> {
    return this.reorderStepMedia(photoId, data);
  }

  /** @deprecated Use getMediaUrl instead */
  getPhotoUrl(uuid: string): string {
    return this.getMediaUrl(uuid);
  }
}

export const sopApi = new SOPApi();
