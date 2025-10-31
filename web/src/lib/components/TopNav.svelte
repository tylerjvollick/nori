<script lang="ts">
  import { authStore } from '$lib/stores/auth';
  import { themeStore } from '$lib/stores/theme';
  import { sidebarStore } from '$lib/stores/sidebar';
  import { goto } from '$app/navigation';
  import { Moon, Sun, Search, Plus, Menu } from 'lucide-svelte';
  import type { User } from '$lib/api/auth';

  interface TopNavProps {
    onCreateTask?: () => void;
    onCreateSOP?: () => void;
  }

  let { onCreateTask, onCreateSOP }: TopNavProps = $props();

  let user: User | null = $state(null);
  let theme: 'light' | 'dark' = $state('light');
  let showDropdown = $state(false);

  // Subscribe to stores
  authStore.subscribe((state) => {
    user = state.user;
  });

  themeStore.subscribe((value) => {
    theme = value;
  });

  function handleLogout() {
    authStore.logout();
    goto('/login');
  }

  function toggleTheme() {
    themeStore.toggle();
  }

  function handleCreateTask() {
    showDropdown = false;
    onCreateTask?.();
  }

  function handleCreateSOP() {
    showDropdown = false;
    onCreateSOP?.();
  }

  function closeDropdown() {
    showDropdown = false;
  }

  function toggleSidebar() {
    sidebarStore.toggle();
  }
</script>

<svelte:window onclick={closeDropdown} />

<header class="border-b border-gray-200 dark:border-gray-700 bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm sticky top-0 z-40">
  <div class="max-w-7xl mx-auto px-6 py-4">
    <div class="flex items-center justify-between gap-4">
      <!-- Sidebar Toggle Button -->
      <button
        onclick={toggleSidebar}
        class="p-2 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        aria-label="Toggle sidebar"
      >
        <Menu class="w-5 h-5" />
      </button>

      <!-- Search Bar (Placeholder for future) -->
      <div class="hidden md:flex flex-1 max-w-xl">
        <div class="relative w-full">
          <Search class="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search SOPs, tasks... (coming soon)"
            disabled
            class="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-900 text-gray-400 cursor-not-allowed"
          />
        </div>
      </div>

      <!-- Right side: Create Button + User Menu -->
      <div class="flex items-center gap-3">
        <!-- Create Dropdown -->
        <div class="relative">
          <button
            onclick={(e) => {
              e.stopPropagation();
              showDropdown = !showDropdown;
            }}
            class="bg-emerald-600 hover:bg-emerald-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
          >
            <Plus class="w-4 h-4" />
            Create
          </button>

          {#if showDropdown}
            <div 
              role="menu"
              class="absolute right-0 mt-2 w-56 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-50"
              onclick={(e) => e.stopPropagation()}
              onkeydown={(e) => e.key === 'Escape' && closeDropdown()}
            >
              <button
                onclick={handleCreateTask}
                class="w-full px-4 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors flex items-start gap-3"
              >
                <svg class="w-5 h-5 text-emerald-600 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                </svg>
                <div>
                  <div class="text-sm font-medium text-gray-900 dark:text-white">Task</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">From SOP template</div>
                </div>
              </button>
              <button
                onclick={handleCreateSOP}
                class="w-full px-4 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors flex items-start gap-3"
              >
                <svg class="w-5 h-5 text-emerald-600 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <div>
                  <div class="text-sm font-medium text-gray-900 dark:text-white">SOP</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">Create new template</div>
                </div>
              </button>
            </div>
          {/if}
        </div>

        <!-- Theme Toggle -->
        <button
          onclick={toggleTheme}
          class="p-2 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          aria-label="Toggle theme"
        >
          {#if theme === 'dark'}
            <Sun class="w-5 h-5" />
          {:else}
            <Moon class="w-5 h-5" />
          {/if}
        </button>

        <!-- User Menu -->
        {#if user}
          <div class="flex items-center gap-3 pl-3 border-l border-gray-200 dark:border-gray-700">
            <span class="text-sm text-gray-700 dark:text-gray-300 hidden sm:inline">
              {user.firstName} {user.lastName}
            </span>
            <button
              onclick={handleLogout}
              class="text-sm text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white font-medium"
            >
              Logout
            </button>
          </div>
        {/if}
      </div>
    </div>
  </div>
</header>
