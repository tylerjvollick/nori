import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import type { User, LoginResponse } from '$lib/api/auth';

interface AuthState {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  isAuthenticated: boolean;
}

function createAuthStore() {
  const initialState: AuthState = {
    user: null,
    token: null,
    isLoading: true,
    isAuthenticated: false,
  };

  const { subscribe, set, update } = writable<AuthState>(initialState);

  return {
    subscribe,

    async initialize() {
      if (!browser) return;

      const token = localStorage.getItem('accessToken');
      if (!token) {
        update(state => ({ ...state, isLoading: false }));
        return;
      }

      try {
        // Verify token by fetching user profile
        const response = await fetch('http://localhost:8080/user/me', {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });

        if (response.ok) {
          const user = await response.json();
          set({
            user,
            token,
            isLoading: false,
            isAuthenticated: true,
          });
        } else {
          // Token is invalid, clear it
          localStorage.removeItem('accessToken');
          set({
            user: null,
            token: null,
            isLoading: false,
            isAuthenticated: false,
          });
        }
      } catch (error) {
        console.error('Auth initialization error:', error);
        localStorage.removeItem('accessToken');
        set({
          user: null,
          token: null,
          isLoading: false,
          isAuthenticated: false,
        });
      }
    },

    async login(loginResponse: LoginResponse) {
      if (!browser) return;

      localStorage.setItem('accessToken', loginResponse.accessToken);
      set({
        user: {
          id: loginResponse.userId,
          email: loginResponse.userEmail,
          firstName: loginResponse.firstName,
          lastName: loginResponse.lastName,
        },
        token: loginResponse.accessToken,
        isLoading: false,
        isAuthenticated: true,
      });
    },

    logout() {
      if (!browser) return;

      localStorage.removeItem('accessToken');
      set({
        user: null,
        token: null,
        isLoading: false,
        isAuthenticated: false,
      });
    },

    setLoading(loading: boolean) {
      update(state => ({ ...state, isLoading: loading }));
    },
  };
}

export const authStore = createAuthStore();