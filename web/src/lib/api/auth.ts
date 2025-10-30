import { apiClient } from './client';

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  firstName: string;
  lastName: string;
  email: string;
  password: string;
}

export interface LoginResponse {
  accessToken: string;
  userId: string;
  userEmail: string;
  firstName: string;
  lastName: string;
}

export interface User {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  globalRole?: 'ADMIN' | 'USER' | 'VIEWER';
  defaultAccountId?: string;
}

export const authApi = {
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    return apiClient.post<LoginResponse>('/auth/login', credentials);
  },

  async register(userData: RegisterRequest): Promise<User> {
    return apiClient.post<User>('/auth/register', userData);
  },

  async getCurrentUser(): Promise<User> {
    return apiClient.get<User>('/user/me');
  },
};