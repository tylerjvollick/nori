import { writable } from 'svelte/store';
import { browser } from '$app/environment';
import { authApi, type User, type LoginResponse } from '$lib/api/auth';
import { setActiveSpaceID } from '$lib/api/client';

interface AuthState {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  mustChangePassword: boolean;
}

function createAuthStore() {
  const initialState: AuthState = {
    user: null,
    isLoading: true,
    isAuthenticated: false,
    mustChangePassword: false,
  };

  const { subscribe, set, update } = writable<AuthState>(initialState);

  return {
    subscribe,

    /**
     * Initialize auth state by calling GET /auth/me.
     * The backend reads the JWT from the HTTP-only `nori_token` cookie,
     * so no token needs to be sent manually. The apiClient sends
     * `credentials: 'include'` to ensure the cookie is attached.
     *
     * Falls back to localStorage token (Authorization header) for
     * backwards compatibility during migration.
     */
    async initialize() {
      if (!browser) return;

      try {
        const user = await authApi.getCurrentUser();

        // Persist the active space ID for the X-Space-ID header
        if (user.activeSpaceId) {
          setActiveSpaceID(user.activeSpaceId);
        }

        set({
          user,
          isLoading: false,
          isAuthenticated: true,
          mustChangePassword: false,
        });
      } catch {
        // Token/cookie is invalid or missing — user is not authenticated
        localStorage.removeItem('accessToken');
        setActiveSpaceID(null);
        set({
          user: null,
          isLoading: false,
          isAuthenticated: false,
          mustChangePassword: false,
        });
      }
    },

    /**
     * Process a successful login response.
     * The backend sets the HTTP-only cookie automatically on POST /auth/login.
     * We also store the token in localStorage as a fallback for the
     * Authorization header (the apiClient sends both).
     */
    async login(loginResponse: LoginResponse) {
      if (!browser) return;

      // Store token in localStorage as fallback
      localStorage.setItem('accessToken', loginResponse.accessToken);

      // Persist active space ID
      if (loginResponse.activeSpaceId) {
        setActiveSpaceID(loginResponse.activeSpaceId);
      }

      if (loginResponse.mustChangePassword) {
        set({
          user: null,
          isLoading: false,
          isAuthenticated: true,
          mustChangePassword: true,
        });
        return;
      }

      // Fetch full user profile (includes accessibleSpaces, role, etc.)
      try {
        const user = await authApi.getCurrentUser();
        set({
          user,
          isLoading: false,
          isAuthenticated: true,
          mustChangePassword: false,
        });
      } catch {
        // Fallback: use the login response data directly
        set({
          user: {
            id: loginResponse.userId,
            email: loginResponse.userEmail,
            firstName: loginResponse.firstName,
            lastName: loginResponse.lastName,
            mustChangePassword: false,
            activeSpaceId: loginResponse.activeSpaceId,
            accessibleSpaces: [],
          },
          isLoading: false,
          isAuthenticated: true,
          mustChangePassword: false,
        });
      }
    },

    /**
     * Log out the user. Calls POST /auth/logout to clear the HTTP-only cookie
     * on the backend, then clears local state.
     */
    async logout() {
      if (!browser) return;

      try {
        await authApi.logout();
      } catch {
        // Best-effort: even if the server call fails, clear local state
      }

      localStorage.removeItem('accessToken');
      setActiveSpaceID(null);
      set({
        user: null,
        isLoading: false,
        isAuthenticated: false,
        mustChangePassword: false,
      });
    },

    setLoading(loading: boolean) {
      update(state => ({ ...state, isLoading: loading }));
    },
  };
}

export const authStore = createAuthStore();
