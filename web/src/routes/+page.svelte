<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { authStore } from '$lib/stores/auth';
  import type { User } from '$lib/api/auth';
  import { Button } from '$lib/components/ui/button';

  let user: User | null = null;
  let isLoading = true;

  onMount(() => {
    authStore.initialize();
  });

  // Subscribe to auth store changes
  authStore.subscribe((state) => {
    user = state.user;
    isLoading = state.isLoading;

    // Redirect to login if not authenticated
    if (!isLoading && !state.isAuthenticated) {
      goto('/login');
    }
  });
</script>

<svelte:head>
  <title>Dashboard - Nori</title>
</svelte:head>

{#if isLoading}
  <div class="min-h-screen flex items-center justify-center bg-background">
    <div class="animate-spin rounded-full h-32 w-32 border-b-2 border-primary"></div>
  </div>
{:else if user}
  <div class="min-h-screen bg-background">
    <!-- Main content -->
    <main class="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
      <div class="px-4 py-6 sm:px-0">
        <div class="text-center mb-12">
          <h2 class="text-3xl font-bold text-foreground mb-4">
            Welcome back{user.firstName ? `, ${user.firstName}` : ''}!
          </h2>
          <p class="text-muted-foreground text-lg">
            A thin layer that holds everything together for your business processes.
          </p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 max-w-5xl mx-auto">
          <a 
            href="/sops"
            class="bg-card p-8 rounded-lg shadow-md border-2 border-transparent hover:border-primary transition-all hover:shadow-lg"
          >
            <div class="text-4xl mb-4">📋</div>
            <h3 class="text-xl font-semibold text-foreground mb-3">SOPs</h3>
            <p class="text-muted-foreground text-sm">
              Document and manage your standard operating procedures with ease.
            </p>
          </a>

          <Button
            variant="outline"
            class="bg-card p-8 h-auto justify-start flex-col items-start opacity-50 cursor-not-allowed"
            disabled
          >
            <div class="text-4xl mb-4">🎯</div>
            <h3 class="text-xl font-semibold text-foreground mb-3">Tasks</h3>
            <p class="text-muted-foreground text-sm">
              Track tasks and workflows using Lean methodologies.
            </p>
            <span class="text-xs text-primary font-medium mt-2 block">Coming Soon</span>
          </Button>

          <Button
            variant="outline"
            class="bg-card p-8 h-auto justify-start flex-col items-start opacity-50 cursor-not-allowed"
            disabled
          >
            <div class="text-4xl mb-4">📊</div>
            <h3 class="text-xl font-semibold text-foreground mb-3">Analytics</h3>
            <p class="text-muted-foreground text-sm">
              Get insights into your process efficiency and bottlenecks.
            </p>
            <span class="text-xs text-primary font-medium mt-2 block">Coming Soon</span>
          </Button>
        </div>

        <div class="mt-12 bg-primary/5 border border-primary/20 rounded-lg p-6 max-w-3xl mx-auto">
          <h3 class="text-lg font-semibold text-foreground mb-2">Quick Start</h3>
          <p class="text-foreground text-sm mb-4">
            Get started by creating your first SOP template or task from the "+Create" button in the top navigation.
          </p>
          <div class="flex gap-3 text-sm">
            <span class="px-3 py-1 bg-card rounded border border-primary/30 text-foreground">
              ➕ Create Task → From SOP template
            </span>
            <span class="px-3 py-1 bg-card rounded border border-primary/30 text-foreground">
              ➕ Create SOP → New template
            </span>
          </div>
        </div>
      </div>
    </main>
  </div>
{/if}
