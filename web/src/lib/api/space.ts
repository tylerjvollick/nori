import { apiClient } from './client';

export interface Space {
  id: string;
  name: string;
  accountId: string;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateSpaceRequest {
  name: string;
}

export interface UpdateSpaceRequest {
  name?: string;
}

export const spaceApi = {
  async getAll(): Promise<Space[]> {
    return apiClient.get<Space[]>('/api/spaces');
  },

  async getById(id: string): Promise<Space> {
    return apiClient.get<Space>(`/api/spaces/${id}`);
  },

  async getRecent(): Promise<Space[]> {
    return apiClient.get<Space[]>('/api/spaces/recent');
  },

  async create(data: CreateSpaceRequest): Promise<Space> {
    return apiClient.post<Space>('/api/spaces', data);
  },

  async update(id: string, data: UpdateSpaceRequest): Promise<Space> {
    return apiClient.put<Space>(`/api/spaces/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete<void>(`/api/spaces/${id}`);
  },

  async recordVisit(id: string): Promise<{ message: string }> {
    return apiClient.post<{ message: string }>(`/api/spaces/${id}/visit`);
  },
};
