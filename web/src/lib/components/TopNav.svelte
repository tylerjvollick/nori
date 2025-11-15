<script lang="ts">
  import { authStore } from '$lib/stores/auth';
  import { themeStore } from '$lib/stores/theme';
  import { sidebarStore } from '$lib/stores/sidebar';
  import { goto } from '$app/navigation';
  import { Button } from '$lib/components/ui/button';
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

<header class="h-(--header-height) border-b border-border bg-background/80 backdrop-blur-sm sticky top-0 z-40 flex items-center justify-between gap-4 px-2">
      <!-- Sidebar Toggle Button -->
      <Button
        onclick={toggleSidebar}
        variant="ghost"
        size="icon"
        aria-label="Toggle sidebar"
      >
        <Menu class="w-5 h-5" />
      </Button>

      <!-- Search Bar (Placeholder for future) -->
      <div class="hidden md:flex flex-1 max-w-xl">
        <div class="relative w-full">
          <Search class="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search SOPs, tasks... (coming soon)"
            disabled
            class="w-full pl-10 pr-4 py-2 border border-border rounded-lg bg-background text-muted-foreground cursor-not-allowed"
          />
        </div>
      </div>

      <!-- Right side: Create Button + User Menu -->
      <div class="flex items-center gap-3">
        <!-- Create Dropdown -->
        <div class="relative">
          <Button
            onclick={(e) => {
              e.stopPropagation();
              showDropdown = !showDropdown;
            }}
            class="flex items-center gap-2"
          >
            <Plus class="w-4 h-4" />
            Create
          </Button>

          {#if showDropdown}
            <div 
              role="menu"
              class="absolute right-0 mt-2 w-56 bg-card rounded-lg shadow-lg border border-border py-1 z-50"
              onclick={(e) => e.stopPropagation()}
              onkeydown={(e) => e.key === 'Escape' && closeDropdown()}
            >
              <Button
                onclick={handleCreateTask}
                variant="ghost"
                class="w-full px-4 py-3 text-left hover:bg-accent transition-colors flex items-start gap-3 h-auto justify-start"
              >
                <svg class="w-5 h-5 text-primary mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                </svg>
                <div>
                  <div class="text-sm font-medium text-foreground">Task</div>
                  <div class="text-xs text-muted-foreground">From SOP template</div>
                </div>
              </Button>
              <Button
                onclick={handleCreateSOP}
                variant="ghost"
                class="w-full px-4 py-3 text-left hover:bg-accent transition-colors flex items-start gap-3 h-auto justify-start"
              >
                <svg class="w-5 h-5 text-primary mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <div>
                  <div class="text-sm font-medium text-foreground">SOP</div>
                  <div class="text-xs text-muted-foreground">Create new template</div>
                </div>
              </Button>
            </div>
          {/if}
        </div>

        <!-- Theme Toggle -->
        <Button
          onclick={toggleTheme}
          variant="ghost"
          size="icon"
          aria-label="Toggle theme"
        >
          {#if theme === 'dark'}
            <Sun class="w-5 h-5" />
          {:else}
            <Moon class="w-5 h-5" />
          {/if}
        </Button>

        <!-- User Menu -->
        {#if user}
          <div class="flex items-center gap-3 pl-3 border-l border-border">
            <span class="text-sm text-foreground hidden sm:inline">
              {user.firstName} {user.lastName}
            </span>
            <Button
              onclick={handleLogout}
              variant="ghost"
              size="sm"
              class="text-sm"
            >
              Logout
            </Button>
          </div>
        {/if}
      </div>
</header>
