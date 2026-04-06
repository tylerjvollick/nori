import { apiClient } from './client';

// --- User Management Types ---

export interface AdminUser {
	id: string;
	email: string;
	firstName?: string;
	lastName?: string;
	role?: 'admin' | 'user';
	mustChangePassword: boolean;
	createdAt?: string;
	updatedAt?: string;
}

export interface CreateUserRequest {
	email: string;
	firstName: string;
	lastName: string;
	tempPassword: string;
	role: 'admin' | 'user';
}

export interface UpdateUserRequest {
	firstName?: string;
	lastName?: string;
	role?: 'admin' | 'user';
}

// --- API Key Types ---

export interface APIKey {
	id: string;
	accountId: string;
	name: string;
	lastUsedAt?: string;
	expiresAt?: string;
	isActive: boolean;
	createdAt: string;
	createdById: string;
}

export interface CreateAPIKeyRequest {
	name: string;
	expiresAt?: string;
}

export interface CreateAPIKeyResponse {
	rawKey: string;
	apiKey: APIKey;
}

// --- Space Member Types ---

export interface SpaceMember {
	id: string;
	userId: string;
	spaceId: string;
	createdAt: string;
	user: AdminUser;
}

export interface AddSpaceMemberRequest {
	userId: string;
}

// --- Admin API ---

export const adminApi = {
	// User Management
	async listUsers(): Promise<AdminUser[]> {
		return apiClient.get<AdminUser[]>('/admin/users');
	},

	async createUser(data: CreateUserRequest): Promise<AdminUser> {
		return apiClient.post<AdminUser>('/admin/users', data);
	},

	async updateUser(id: string, data: UpdateUserRequest): Promise<AdminUser> {
		return apiClient.put<AdminUser>(`/admin/users/${id}`, data);
	},

	async deleteUser(id: string): Promise<void> {
		return apiClient.delete<void>(`/admin/users/${id}`);
	},

	// API Key Management
	async listAPIKeys(): Promise<APIKey[]> {
		return apiClient.get<APIKey[]>('/admin/api-keys');
	},

	async createAPIKey(data: CreateAPIKeyRequest): Promise<CreateAPIKeyResponse> {
		return apiClient.post<CreateAPIKeyResponse>('/admin/api-keys', data);
	},

	async revokeAPIKey(id: string): Promise<void> {
		return apiClient.delete<void>(`/admin/api-keys/${id}`);
	},

	// Space Member Management
	async getSpaceMembers(spaceId: string): Promise<SpaceMember[]> {
		return apiClient.get<SpaceMember[]>(`/admin/spaces/${spaceId}/members`);
	},

	async addSpaceMember(spaceId: string, data: AddSpaceMemberRequest): Promise<SpaceMember> {
		return apiClient.post<SpaceMember>(`/admin/spaces/${spaceId}/members`, data);
	},

	async removeSpaceMember(spaceId: string, userId: string): Promise<void> {
		return apiClient.delete<void>(`/admin/spaces/${spaceId}/members/${userId}`);
	},
};
