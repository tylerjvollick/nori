import { apiClient } from './client';

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  userId: string;
  userEmail: string;
  firstName: string;
  lastName: string;
  mustChangePassword: boolean;
  activeSpaceId?: string;
}

export interface Space {
  id: string;
  name: string;
  accountId: string;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface User {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  role?: 'admin' | 'user';
  activeSpaceId?: string;
  accessibleSpaces: Space[];
}

export interface ChangePasswordRequest {
  currentPassword: string;
  newPassword: string;
}

export const authApi = {
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    return apiClient.post<LoginResponse>('/auth/login', credentials);
  },

  async getCurrentUser(): Promise<User> {
    return apiClient.get<User>('/auth/me');
  },

  async changePassword(data: ChangePasswordRequest): Promise<LoginResponse> {
    return apiClient.post<LoginResponse>('/auth/change-password', data);
  },

  async logout(): Promise<void> {
    return apiClient.post<void>('/auth/logout');
  },
};
