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
  description?: string;
  materials?: string[];
  equipment?: string[];
  changeSummary?: string;
  createdAt: string;
  isActive: boolean;
  steps?: SOPStep[];
}

export interface SOPTemplate {
  id: number;
  name: string;
  createdAt: string;
  updatedAt: string;
  currentVersion?: SOPVersion;
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
}

export const sopApi = new SOPApi();
