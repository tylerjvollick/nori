import { browser } from '$app/environment';

// Empty string is a valid value: it makes all requests same-origin relative
// URLs (production, where a reverse proxy routes /api etc. to the server).
// The localhost fallback only applies when the var is not set at build time.
const API_BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8081';

export class ApiClient {
  private baseURL: string;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;
  }

  private getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {};

    // Auth token from localStorage (fallback; primary auth is via HTTP-only cookie)
    if (browser) {
      const token = localStorage.getItem('accessToken');
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
    }

    return headers;
  }

  async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...this.getHeaders(),
      ...(options.headers as Record<string, string> || {}),
    };

    const response = await fetch(url, {
      ...options,
      headers,
      credentials: 'include', // Send HTTP-only cookies with every request
    });

    if (!response.ok) {
      // Handle 401 Unauthorized — redirect to login (unless already on an auth page)
      if (response.status === 401) {
        if (browser) {
          localStorage.removeItem('accessToken');
          const path = window.location.pathname;
          if (path !== '/login' && path !== '/change-password') {
            window.location.href = '/login';
          }
        }
        throw new Error('Unauthorized');
      }

      // Handle 403 Forbidden — check for MUST_CHANGE_PASSWORD code
      if (response.status === 403) {
        const body = await response.json().catch(() => ({}));
        if (body.code === 'MUST_CHANGE_PASSWORD') {
          if (browser) {
            window.location.href = '/change-password';
          }
          throw new Error('Must change password');
        }
        throw new Error(body.error || 'Forbidden');
      }

      const error = await response.json().catch(() => ({ error: 'Network error' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    // Handle 204 No Content responses
    if (response.status === 204) {
      return undefined as T;
    }

    return response.json();
  }

  async get<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'GET' });
  }

  async post<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: 'DELETE' });
  }

  async patch<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async uploadFile<T>(endpoint: string, formData: FormData): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const headers = {
      // Don't set Content-Type, let browser set it with boundary
      ...this.getHeaders(),
    };

    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: formData,
      credentials: 'include',
    });

    if (!response.ok) {
      if (response.status === 401) {
        if (browser) {
          localStorage.removeItem('accessToken');
          const path = window.location.pathname;
          if (path !== '/login' && path !== '/change-password') {
            window.location.href = '/login';
          }
        }
        throw new Error('Unauthorized');
      }

      if (response.status === 403) {
        const body = await response.json().catch(() => ({}));
        if (body.code === 'MUST_CHANGE_PASSWORD') {
          if (browser) {
            window.location.href = '/change-password';
          }
          throw new Error('Must change password');
        }
        throw new Error(body.error || 'Forbidden');
      }

      const error = await response.json().catch(() => ({ error: 'Network error' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response.json();
  }
}

export const apiClient = new ApiClient();
