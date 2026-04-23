import { apiClient } from './client';

export interface Space {
  id: string;
  name: string;
  slug: string;
  accountId: string;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface SpaceMemberUser {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
}

export interface SpaceMember {
  id: string;
  userId: string;
  spaceId: string;
  createdAt: string;
  user: SpaceMemberUser;
}

export interface CreateSpaceRequest {
  name: string;
  slug?: string;
}

export interface UpdateSpaceRequest {
  name?: string;
  slug?: string;
}

export const spaceApi = {
  async getAll(): Promise<Space[]> {
    return apiClient.get<Space[]>('/api/spaces');
  },

  async getById(id: string): Promise<Space> {
    return apiClient.get<Space>(`/api/spaces/${id}`);
  },

  async getBySlug(slug: string): Promise<Space> {
    return apiClient.get<Space>(`/api/spaces/by-slug/${slug}`);
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

  async getMembers(id: string): Promise<SpaceMember[]> {
    return apiClient.get<SpaceMember[]>(`/api/spaces/${id}/members`);
  },
};
